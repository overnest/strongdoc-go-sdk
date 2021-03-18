package docidxv1

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

const (
	docIdxBaseDir = "/tmp/document"
	docPathFmt    = docIdxBaseDir + "/%v/%v" // DocID + DocVer
	docDocPathFmt = docPathFmt + "/doc/%v"   // File Name
	docIdxPathFmt = docPathFmt + "/idx"
	doiPathFmt    = docIdxPathFmt + "/offsets"
	dtiPathFmt    = docIdxPathFmt + "/terms"
)

var booksDir string

func init() {
	dir, err := utils.FetchFileLoc("./testDocuments/books")
	if err == nil {
		booksDir = dir
	}
}

type TestDocumentIdxV1 struct {
	// Original Document
	docFileName  string
	docFilePath  string
	docFileSize  int64
	docFile      *os.File
	docReader    io.ReadCloser
	DocID        string
	DocVer       uint64
	AddedTerms   map[string]bool
	DeletedTerms map[string]bool

	// DOI
	doiFilePath string
	doiFileSize int64
	doiFile     *os.File
	doi         *DocOffsetIdxV1

	// DTI
	dtiFilePath string
	dtiFileSize int64
	dtiFile     *os.File
	dti         *DocTermIdxV1
}

func InitTestDocuments(numDocs int, random bool) ([]*TestDocumentIdxV1, error) {
	files, err := ioutil.ReadDir(booksDir)
	if err != nil {
		return nil, errors.New(err)
	}

	docCount := utils.Min(numDocs, len(files))
	if numDocs <= 0 {
		docCount = len(files)
	}

	documents := make([]*TestDocumentIdxV1, 0, docCount)
	for len(files) > 0 {
		var file os.FileInfo
		if random {
			i := rand.Intn(len(files))
			file = files[i]
			files[i] = files[len(files)-1]
			files = files[:len(files)-1]
		} else {
			file = files[0]
			files = files[1:]
		}

		if file.IsDir() {
			continue
		}

		doc, err := createDocumentIdx(file.Name(), file.Size(), file.Name(), 1)
		if err != nil {
			return nil, err
		}

		documents = append(documents, doc)
		if len(documents) >= numDocs {
			break
		}
	}

	return documents, nil
}

func createDocumentIdx(name string, size int64, id string, ver uint64) (*TestDocumentIdxV1, error) {
	filePath := path.Join(booksDir, name)
	doc := &TestDocumentIdxV1{
		docFileName:  name,
		docFilePath:  filePath,
		docFileSize:  size,
		docFile:      nil,
		docReader:    nil,
		DocID:        id,
		DocVer:       ver,
		AddedTerms:   make(map[string]bool),
		DeletedTerms: make(map[string]bool),
	}

	doc.doiFilePath = fmt.Sprintf(doiPathFmt, doc.DocID, doc.DocVer)
	doc.dtiFilePath = fmt.Sprintf(dtiPathFmt, doc.DocID, doc.DocVer)

	return doc, nil
}

func OpenCreatedDoc(docID string, docVer uint64) (*TestDocumentIdxV1, error) {
	doc, err := createDocumentIdx(docID, 0, docID, docVer)
	if err != nil {
		return nil, err
	}

	_, doc.docFile, doc.docReader, err = utils.OpenFile(doc.docFilePath)
	if err != nil {
		return nil, err
	}

	stat, err := doc.docFile.Stat()
	if err != nil {
		return nil, err
	}

	doc.docFileSize = stat.Size()
	return doc, nil
}

func (doc *TestDocumentIdxV1) CreateModifiedDoc(addTerms, deleteTerms int) (*TestDocumentIdxV1, error) {
	var term string
	var space rune

	newDoc, err := createDocumentIdx(doc.docFileName, 0, doc.docFileName, doc.DocVer+1)
	if err != nil {
		return nil, err
	}

	// Figure out which terms to add and delete
	_, file, freader, err := utils.OpenFile(doc.docFilePath)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(freader)
	termMap := make(map[string]bool)
	for err == nil {
		term, _, err = doc.readNextTerm(reader)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if err == nil && utils.IsAlphaNumeric(term) {
			termMap[strings.ToLower(term)] = true
		}
	}
	file.Close()

	terms := make([]string, 0, len(termMap))
	for term := range termMap {
		terms = append(terms, term)
	}

	// If you want to enable more randomization of the added
	// or removed terms, comment out the following line
	sort.Strings(terms)

	for addTerms > 0 {
		term = strings.ToLower(terms[rand.Intn(len(terms))])
		if !newDoc.AddedTerms[term] {
			newDoc.AddedTerms[term] = true
			addTerms--
		}
	}

	for deleteTerms > 0 {
		term = strings.ToLower(terms[rand.Intn(len(terms))])
		if !newDoc.AddedTerms[term] && !newDoc.DeletedTerms[term] {
			newDoc.DeletedTerms[term] = true
			deleteTerms--
		}
	}

	// Open old file for reading
	_, file, freader, err = utils.OpenFile(doc.docFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader = bufio.NewReader(freader)

	// Create the new file
	newDoc.docFilePath = fmt.Sprintf(docDocPathFmt, newDoc.DocID, newDoc.DocVer, newDoc.docFileName)
	if err := os.MkdirAll(filepath.Dir(newDoc.docFilePath), 0770); err != nil {
		return nil, errors.New(err)
	}

	newDoc.docFile, err = os.Create(newDoc.docFilePath)
	if err != nil {
		return nil, errors.New(err)
	}
	defer newDoc.docFile.Close()

	gzipWriter := gzip.NewWriter(newDoc.docFile)
	newDocWriter := bufio.NewWriter(gzipWriter)

	// Copy old file content to new file
	for err == nil {
		term, space, err = doc.readNextTerm(reader)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if err != nil {
			continue
		}

		if !utils.IsAlphaNumeric(term) {
			_, err = newDocWriter.WriteString(term)
			if err != nil {
				return nil, errors.New(err)
			}

			if space != unicode.ReplacementChar {
				_, err = newDocWriter.WriteRune(space)
				if err != nil {
					return nil, errors.New(err)
				}
			}

			continue
		}

		lterm := strings.ToLower(term)

		if newDoc.AddedTerms[lterm] {
			_, err = newDocWriter.WriteString(term)
			if err != nil {
				return nil, errors.New(err)
			}

			_, err = newDocWriter.WriteString(
				fmt.Sprintf(" %v%v", term, newDoc.getAddTermSuffix()))
			if err != nil {
				return nil, errors.New(err)
			}
		} else if !newDoc.DeletedTerms[lterm] {
			_, err = newDocWriter.WriteString(term)
			if err != nil {
				return nil, errors.New(err)
			}
		}

		if space != unicode.ReplacementChar {
			_, err = newDocWriter.WriteRune(space)
			if err != nil {
				return nil, errors.New(err)
			}
		}
	}

	newAddedTerms := make(map[string]bool)
	for term, v := range newDoc.AddedTerms {
		addTerm := strings.ToLower(fmt.Sprintf("%v%v", term, newDoc.getAddTermSuffix()))
		newAddedTerms[addTerm] = v
	}
	newDoc.AddedTerms = newAddedTerms

	newDocWriter.Flush()
	gzipWriter.Close()
	newDoc.CloseDoc()
	doc.CloseDoc()

	// Get new file size
	f, err := os.Open(newDoc.docFilePath)
	if err != nil {
		return nil, errors.New(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, errors.New(err)
	}
	newDoc.docFileSize = stat.Size()

	return newDoc, nil
}

func (doc *TestDocumentIdxV1) getAddTermSuffix() string {
	return fmt.Sprintf("ADDTERM%v", doc.DocVer)
}

func (doc *TestDocumentIdxV1) readNextTerm(reader *bufio.Reader) (term string, space rune, err error) {
	err = nil
	term = ""
	space = unicode.ReplacementChar

	var buffer bytes.Buffer
	bufWriter := bufio.NewWriter(&buffer)

	for err == nil {
		var r rune
		r, _, err = reader.ReadRune()
		if err != nil && err != io.EOF {
			return term, space, errors.New(err)
		}

		if err == io.EOF || unicode.IsSpace(r) {
			space = r
			break
		} else {
			_, err = bufWriter.WriteRune(r)
			if err != nil {
				return term, space, errors.New(err)
			}
		}
	}

	bufWriter.Flush()
	if buffer.Len() > 0 {
		var berr error
		term, berr = buffer.ReadString(' ')
		if berr != nil && berr != io.EOF {
			return term, space, errors.New(berr)
		}
	}

	return term, space, err
}

func (doc *TestDocumentIdxV1) CloseDoc() error {
	var err error = nil
	if doc.docReader != nil {
		err = firstError(err, doc.docReader.Close())
	}
	if doc.docFile != nil {
		err = firstError(err, doc.docFile.Close())
	}
	doc.docFile = nil
	return err
}

func (doc *TestDocumentIdxV1) CreateDoi(key *sscrypto.StrongSaltKey) error {
	doc.doiFilePath = fmt.Sprintf(doiPathFmt, doc.DocID, doc.DocVer)
	if err := os.MkdirAll(filepath.Dir(doc.doiFilePath), 0770); err != nil {
		return errors.New(err)
	}

	var err error
	doc.doiFile, err = os.Create(doc.doiFilePath)
	if err != nil {
		return errors.New(err)
	}
	defer doc.CloseDoi()

	doc.doi, err = CreateDocOffsetIdxV1(doc.DocID, doc.DocVer, key, doc.doiFile, 0)
	if err != nil {
		return err
	}

	_, file, freader, err := utils.OpenFile(doc.docFilePath)
	if err != nil {
		return err
	}

	i := uint64(0)
	reader := bufio.NewReader(freader)
	for err == nil {
		var term string
		term, _, err = doc.readNextTerm(reader)
		if err != nil && err != io.EOF {
			return err
		}

		if err == nil && utils.IsAlphaNumeric(term) {
			err = doc.doi.AddTermOffset(strings.ToLower(term), i)
			if err != nil {
				return err
			}
		}
		i++
	}
	file.Close()

	return nil
}

func (doc *TestDocumentIdxV1) OpenDoi(key *sscrypto.StrongSaltKey) (*DocOffsetIdxV1, error) {
	var err error

	err = doc.CloseDoi()
	if err != nil {
		return nil, err
	}

	doc.doiFile, err = os.Open(doc.doiFilePath)
	if err != nil {
		return nil, errors.New(err)
	}

	stat, err := doc.doiFile.Stat()
	if err != nil {
		return nil, errors.New(err)
	}

	doc.doiFileSize = stat.Size()
	doc.doi, err = OpenDocOffsetIdxV1(key, doc.doiFile, 0)
	if err != nil {
		return nil, err
	}

	return doc.doi, nil
}

func (doc *TestDocumentIdxV1) CloseDoi() error {
	var err error = nil
	if doc.doi != nil {
		err = firstError(err, doc.doi.Close())
	}
	if doc.doiFile != nil {
		err = firstError(err, doc.doiFile.Close())
	}
	doc.doi = nil
	doc.doiFile = nil
	doc.doiFileSize = 0
	return err
}

func (doc *TestDocumentIdxV1) CreateDti(key *sscrypto.StrongSaltKey) error {
	doc.dtiFilePath = fmt.Sprintf(dtiPathFmt, doc.DocID, doc.DocVer)
	if err := os.MkdirAll(filepath.Dir(doc.dtiFilePath), 0770); err != nil {
		return errors.New(err)
	}

	var err error
	doc.dtiFile, err = os.Create(doc.dtiFilePath)
	if err != nil {
		return errors.New(err)
	}
	defer doc.CloseDti()

	doi, err := doc.OpenDoi(key)
	if err != nil {
		return err
	}
	defer doc.CloseDoi()

	src, err := OpenDocTermSourceDocOffsetV1(doi)
	if err != nil {
		return err
	}
	defer src.Close()

	doc.dti, err = CreateDocTermIdxV1(doc.DocID, doc.DocVer, key, src, doc.dtiFile, 0)
	if err != nil {
		return err
	}
	defer doc.CloseDti()

	for err == nil {
		_, err = doc.dti.WriteNextBlock()
		if err != nil && err != io.EOF {
			return err
		}
	}

	return nil
}

func (doc *TestDocumentIdxV1) OpenDti(key *sscrypto.StrongSaltKey) (*DocTermIdxV1, error) {
	var err error

	err = doc.CloseDti()
	if err != nil {
		return nil, err
	}

	doc.dtiFile, err = os.Open(doc.dtiFilePath)
	if err != nil {
		return nil, errors.New(err)
	}

	stat, err := doc.dtiFile.Stat()
	if err != nil {
		return nil, errors.New(err)
	}

	doc.dtiFileSize = stat.Size()
	doc.dti, err = OpenDocTermIdxV1(key, doc.dtiFile, 0, uint64(doc.dtiFileSize))
	if err != nil {
		return nil, err
	}

	return doc.dti, nil
}

func (doc *TestDocumentIdxV1) CloseDti() error {
	var err error = nil
	if doc.dti != nil {
		err = firstError(err, doc.dti.Close())
	}
	if doc.dtiFile != nil {
		err = firstError(err, doc.dtiFile.Close())
	}
	doc.dti = nil
	doc.dtiFile = nil
	doc.dtiFileSize = 0
	return err
}

func (doc *TestDocumentIdxV1) Close() error {
	var err error = nil

	err = firstError(err, doc.CloseDoc())
	err = firstError(err, doc.CloseDti())
	err = firstError(err, doc.CloseDoi())
	if doc.docFile != nil {
		err = firstError(err, doc.docFile.Close())
	}
	doc.dti = nil
	doc.dtiFile = nil
	return err
}

func (doc *TestDocumentIdxV1) Clean() error {
	var err error = nil

	err = firstError(err, doc.Close())
	err = firstError(err, os.RemoveAll(fmt.Sprintf(docPathFmt, doc.DocID, doc.DocVer)))

	return err
}

func CleanTestDocumentIndexes() error {
	return os.RemoveAll(docIdxBaseDir)
}

// Keep only the first error
func firstError(curErr, newErr error) error {
	if curErr == nil && newErr != nil {
		curErr = newErr
	}
	return curErr
}
