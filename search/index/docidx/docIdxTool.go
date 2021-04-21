package docidx

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

const (
	docIdxBaseDir = "/tmp/document"
	docDocPathFmt = docIdxBaseDir + "/%v/%v/doc/%v"
)

var booksDir string

func init() {
	dir, err := utils.FetchFileLoc("./testDocuments/books")
	if err == nil {
		booksDir = dir
	}
}

type TestDocumentIdxV1 struct {
	docFileName  string
	docFilePath  string
	DocID        string
	DocVer       uint64
	AddedTerms   map[string]bool
	DeletedTerms map[string]bool
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

		doc, err := createDocumentIdx(file.Name(), file.Name(), 1)
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

func createDocumentIdx(name, id string, ver uint64) (*TestDocumentIdxV1, error) {
	filePath := path.Join(booksDir, name)
	doc := &TestDocumentIdxV1{
		docFileName:  name,
		docFilePath:  filePath,
		DocID:        id,
		DocVer:       ver,
		AddedTerms:   make(map[string]bool),
		DeletedTerms: make(map[string]bool),
	}

	return doc, nil
}

func createNewDocumentIdx(oldDoc *TestDocumentIdxV1, addTerms map[string]bool, deleteTerms map[string]bool) (newDoc *TestDocumentIdxV1, err error) {
	newDoc, err = createDocumentIdx(oldDoc.docFileName, oldDoc.DocID, oldDoc.DocVer+1)
	if err != nil {
		return
	}
	newDoc.docFilePath = fmt.Sprintf(docDocPathFmt, newDoc.DocID, newDoc.DocVer, newDoc.docFileName)
	newDoc.AddedTerms = addTerms
	newDoc.DeletedTerms = deleteTerms
	docFile, err := utils.MakeDirAndCreateFile(newDoc.docFilePath)
	if err != nil {
		return
	}
	err = docFile.Close()
	return
}

// get tokenized term (standardized in lower case)
func (doc *TestDocumentIdxV1) getTermsFromRawData() ([]string, error) {
	file, err := utils.OpenLocalFile(doc.docFilePath)
	defer file.Close()
	tokenizer, err := utils.OpenRawFileTokenizer(file)
	if err != nil {
		return nil, err
	}

	// term map
	termMap := make(map[string]bool)
	for token, _, _, err := tokenizer.NextToken(); err != io.EOF; token, _, _, err = tokenizer.NextToken() {
		if err != nil && err != io.EOF {
			return nil, err
		}
		termMap[token] = true
	}

	// term list
	var terms []string
	for term := range termMap {
		terms = append(terms, term)
	}
	return terms, nil
}

// write modified doc
func writeNewDoc(oldDoc *TestDocumentIdxV1, newDoc *TestDocumentIdxV1) error {
	// Open old file for reading
	oldDocFile, err := utils.OpenLocalFile(oldDoc.docFilePath)
	if err != nil {
		return err
	}
	defer oldDocFile.Close()
	tokenizer, err := utils.OpenRawFileTokenizer(oldDocFile)
	if err != nil {
		return err
	}
	// Create the new file
	newDocFile, err := utils.OpenLocalFile(newDoc.docFilePath)
	if err != nil {
		return err
	}
	defer newDocFile.Close()

	gzipWriter := gzip.NewWriter(newDocFile)
	newDocWriter := bufio.NewWriter(gzipWriter)

	// Copy old file content to new file
	for term, space, err := tokenizer.NextRawToken(); err != io.EOF; term, space, err = tokenizer.NextRawToken() {
		if !utils.IsAlphaNumeric(term) {
			_, err = newDocWriter.WriteString(term)
			if err != nil {
				return err
			}

			if space != unicode.ReplacementChar {
				_, err = newDocWriter.WriteRune(space)
				if err != nil {
					return err
				}
			}
			continue
		}

		lterm := strings.ToLower(term)

		if newDoc.AddedTerms[lterm] {
			_, err = newDocWriter.WriteString(term)
			if err != nil {
				return err
			}

			_, err = newDocWriter.WriteString(
				fmt.Sprintf(" %v%v", term, newDoc.getAddTermSuffix()))
			if err != nil {
				return err
			}
		} else if !newDoc.DeletedTerms[lterm] {
			_, err = newDocWriter.WriteString(term)
			if err != nil {
				return err
			}
		}

		if space != unicode.ReplacementChar {
			_, err = newDocWriter.WriteRune(space)
			if err != nil {
				return err
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

	return nil
}

// create new version of local file (add some terms and delete some terms)
// make sure addTerms, deleteTerms < the number of all terms
func (doc *TestDocumentIdxV1) CreateModifiedDoc(addTerms, deleteTerms int) (*TestDocumentIdxV1, error) {
	// Open old file, get all terms
	terms, err := doc.getTermsFromRawData()
	if err != nil {
		return nil, err
	}

	// If you want to enable more randomization of the added
	// or removed terms, comment out the following line
	sort.Strings(terms)

	// Figure out which terms to add and delete
	addTermsMap := make(map[string]bool)
	for addTerms > 0 {
		term := terms[rand.Intn(len(terms))]
		if !addTermsMap[term] {
			addTermsMap[term] = true
			addTerms--
		}
	}

	deleteTermsMap := make(map[string]bool)
	for deleteTerms > 0 {
		term := terms[rand.Intn(len(terms))]
		if !addTermsMap[term] && !deleteTermsMap[term] {
			deleteTermsMap[term] = true
			deleteTerms--
		}
	}

	// Init new doc
	newDoc, err := createNewDocumentIdx(doc, addTermsMap, deleteTermsMap)
	if err != nil {
		return nil, err
	}

	// Copy old doc to new doc
	err = writeNewDoc(doc, newDoc)
	if err != nil {
		return nil, err
	}

	return newDoc, nil
}

func (doc *TestDocumentIdxV1) getAddTermSuffix() string {
	return fmt.Sprintf("ADDTERM%v", doc.DocVer)
}

func (doc *TestDocumentIdxV1) CreateDoiAndDti(sdc client.StrongDocClient, key *sscrypto.StrongSaltKey) error {
	// create doi and dti for new doc
	err := doc.CreateDoi(sdc, key)
	if err != nil {
		return err
	}

	err = doc.CreateDti(sdc, key)
	return err
}

func (doc *TestDocumentIdxV1) CreateDoi(sdc client.StrongDocClient, key *sscrypto.StrongSaltKey) error {
	file, err := utils.OpenLocalFile(doc.docFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tokenizer, err := utils.OpenRawFileTokenizer(file)
	if err != nil {
		return err
	}
	doi, err := CreateDocOffsetIdx(sdc, doc.DocID, doc.DocVer, key, 0)

	if err != nil {
		return err
	}
	defer doi.Close()

	for token, _, wordCounter, err := tokenizer.NextToken(); err != io.EOF; token, _, wordCounter, err = tokenizer.NextToken() {
		if err != nil && err != io.EOF {
			return err
		}
		addErr := doi.AddTermOffset(token, wordCounter)
		if addErr != nil {
			return addErr
		}
	}
	return nil
}

func (doc *TestDocumentIdxV1) OpenDoi(sdc client.StrongDocClient, key *sscrypto.StrongSaltKey) (common.DocOffsetIdx, error) {
	doi, err := OpenDocOffsetIdx(sdc, doc.DocID, doc.DocVer, key)
	if err != nil {
		return nil, err
	}
	return doi, nil
}

func (doc *TestDocumentIdxV1) CreateDti(sdc client.StrongDocClient, key *sscrypto.StrongSaltKey) error {
	return CreateAndSaveDocTermIdxFromDOI(sdc, doc.DocID, doc.DocVer, key)
}

func (doc *TestDocumentIdxV1) OpenDti(sdc client.StrongDocClient, key *sscrypto.StrongSaltKey) (common.DocTermIdx, error) {
	dti, err := OpenDocTermIdx(sdc, doc.DocID, doc.DocVer, key)
	if err != nil {
		return nil, err
	}
	return dti, nil
}

func RemoveTestDocumentsDocIdx(sdc client.StrongDocClient, docs []*TestDocumentIdxV1) error {
	for _, doc := range docs {
		err := common.RemoveDocIndexes(sdc, doc.DocID)
		if err != nil {
			return err
		}
	}
	return nil
}

// clean all tmp files
func CleanAllTmpFiles() error {
	return os.RemoveAll(docIdxBaseDir)
}
