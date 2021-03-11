package searchidx

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
)

const (
	docIdxBaseDir = "/tmp/document"
	docIdxPathFmt = docIdxBaseDir + "/%v/%v/idx" // DocID + DocVer
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

type DocumentIdx struct {
	// Original Document
	docFileName string
	docFilePath string
	docFileSize int64
	docFile     *os.File
	docID       string
	docVer      uint64

	// DOI
	doiFilePath string
	doiFileSize int64
	doiFile     *os.File
	doi         *docidx.DocOffsetIdxV1

	// DTI
	dtiFilePath string
	dtiFileSize int64
	dtiFile     *os.File
	dti         *docidx.DocTermIdxV1
}

func TestTools(t *testing.T) {
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	indexes, err := InitDocuments(10, false)
	assert.NilError(t, err)

	// Create the indexes
	for _, idx := range indexes {
		assert.NilError(t, idx.CreateDoi(key))
		assert.NilError(t, idx.CreateDti(key))
	}
	defer CleanDocumentIndexes()

	// Validate the indexes
	for _, idx := range indexes {
		// Open the DOI
		doi, err := idx.OpenDoi(key)
		assert.NilError(t, err)
		defer idx.CloseDoi()

		termloc := make(map[string][]uint64)
		for err == nil {
			var blk *docidx.DocOffsetIdxBlkV1
			blk, err = doi.ReadNextBlock()
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}
			if blk != nil {
				for term, locs := range blk.TermLoc {
					termloc[term] = append(termloc[term], locs...)
				}
			}
		}

		i := uint64(0)
		tokenizer, err := utils.OpenFileTokenizer(idx.docFilePath)
		assert.NilError(t, err)
		defer tokenizer.Close()

		// Validate the DOI
		for token, _, err := tokenizer.NextToken(); err != io.EOF; token, _, err = tokenizer.NextToken() {
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}

			if token != "" {
				locs, exist := termloc[token]
				assert.Assert(t, exist)
				assert.Assert(t, len(locs) > 0)
				assert.Equal(t, i, locs[0])
				termloc[token] = locs[1:]
				i++
			}
		}

		// Open the DTI
		dti, err := idx.OpenDti(key)
		assert.NilError(t, err)
		defer idx.CloseDti()

		terms := make([]string, 0, len(termloc))
		for term := range termloc {
			terms = append(terms, term)
			delete(termloc, term)
		}
		sort.Strings(terms)

		// Validate the DTI
		for err == nil {
			var blk *docidx.DocTermIdxBlkV1
			blk, err = dti.ReadNextBlock()
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}
			if blk != nil {
				for _, term := range blk.Terms {
					assert.Assert(t, len(terms) > 0)
					assert.Equal(t, term, terms[0])
					terms = terms[1:]
				}
			}
		}

		assert.Equal(t, len(terms), 0)
	} // for _, idx := range indexes
}

func InitDocuments(numDocs int, random bool) ([]*DocumentIdx, error) {
	files, err := ioutil.ReadDir(booksDir)
	if err != nil {
		return nil, errors.New(err)
	}

	docCount := utils.Min(numDocs, len(files))
	if numDocs <= 0 {
		docCount = len(files)
	}

	documents := make([]*DocumentIdx, 0, docCount)
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

func createDocumentIdx(name string, size int64, id string, ver uint64) (*DocumentIdx, error) {
	filePath := path.Join(booksDir, name)
	doc := &DocumentIdx{
		docFileName: name,
		docFilePath: filePath,
		docFileSize: size,
		docFile:     nil,
		docID:       id,
		docVer:      ver,
	}

	return doc, nil
}

func (doc *DocumentIdx) CreateDoi(key *sscrypto.StrongSaltKey) error {
	doc.doiFilePath = fmt.Sprintf(doiPathFmt, doc.docID, doc.docVer)
	if err := os.MkdirAll(filepath.Dir(doc.doiFilePath), 0770); err != nil {
		return err
	}

	var err error
	doc.doiFile, err = os.Create(doc.doiFilePath)
	if err != nil {
		return errors.New(err)
	}
	defer doc.CloseDoi()

	doc.doi, err = docidx.CreateDocOffsetIdx(doc.docID, doc.docVer, key, doc.doiFile, 0)
	if err != nil {
		return err
	}

	tokenizer, err := utils.OpenFileTokenizer(doc.docFilePath)
	if err != nil {
		return err
	}
	defer tokenizer.Close()

	i := uint64(0)
	for token, _, err := tokenizer.NextToken(); err != io.EOF; token, _, err = tokenizer.NextToken() {
		err = doc.doi.AddTermOffset(token, i)
		if err != nil && err != io.EOF {
			return err
		}
		i++
	}

	return nil
}

func (doc *DocumentIdx) OpenDoi(key *sscrypto.StrongSaltKey) (*docidx.DocOffsetIdxV1, error) {
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
	doc.doi, err = docidx.OpenDocOffsetIdxV1(key, doc.doiFile, 0)
	if err != nil {
		return nil, err
	}

	return doc.doi, nil
}

func (doc *DocumentIdx) CloseDoi() error {
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

func (doc *DocumentIdx) CreateDti(key *sscrypto.StrongSaltKey) error {
	doc.dtiFilePath = fmt.Sprintf(dtiPathFmt, doc.docID, doc.docVer)
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

	src, err := docidx.OpenDocTermSourceDocOffsetV1(doi)
	if err != nil {
		return err
	}
	defer src.Close()

	doc.dti, err = docidx.CreateDocTermIdxV1(doc.docID, doc.docVer, key, src, doc.dtiFile, 0)
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

func (doc *DocumentIdx) OpenDti(key *sscrypto.StrongSaltKey) (*docidx.DocTermIdxV1, error) {
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
	doc.dti, err = docidx.OpenDocTermIdxV1(key, doc.dtiFile, 0, uint64(doc.dtiFileSize))
	if err != nil {
		return nil, err
	}

	return doc.dti, nil
}

func (doc *DocumentIdx) CloseDti() error {
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

func (doc *DocumentIdx) Close() error {
	var err error = nil

	err = firstError(err, doc.CloseDti())
	err = firstError(err, doc.CloseDoi())
	if doc.docFile != nil {
		err = firstError(err, doc.docFile.Close())
	}
	doc.dti = nil
	doc.dtiFile = nil
	return err
}

func (doc *DocumentIdx) Clean() error {
	var err error = nil

	err = firstError(err, doc.Close())
	err = firstError(err, os.RemoveAll(fmt.Sprintf(docIdxPathFmt, doc.docID, doc.docVer)))

	return err
}

func CleanDocumentIndexes() error {
	return os.RemoveAll(docIdxBaseDir)
}

// Keep only the first error
func firstError(curErr, newErr error) error {
	if curErr == nil && newErr != nil {
		curErr = newErr
	}
	return curErr
}
