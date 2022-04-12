package query

// TODO: parse query language and execute

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	scom "github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/searchidxv2"
	"github.com/overnest/strongdoc-go-sdk/search/query/queryv1"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

type Credential struct {
	owner    scom.SearchIdxOwner
	docKey   *sscrypto.StrongSaltKey
	termKey  *sscrypto.StrongSaltKey
	indexKey *sscrypto.StrongSaltKey
}

// DocumentResult contains the document search result
type DocumentResult struct {
	// The document ID that contains the query terms.
	DocID string
	// The score of the search result.
	Score float64
}

func OpenCredentials() *Credential {
	owner := scom.CreateSearchIdxOwner(utils.OwnerUser, "owner1")
	keys, _ := scom.TestGetKeys()
	docKey, termKey, indexKey := keys[scom.TestDocKeyID], keys[scom.TestTermKeyID],
		keys[scom.TestIndexKeyID]
	return &Credential{
		owner, docKey, termKey, indexKey,
	}
}

func UploadDocument(cred *Credential, fileNames []string) ([]string, error) {
	documents := make([]*docidx.TestDocumentIdxV1, 0, len(fileNames))

	for _, filename := range fileNames {
		doc := &docidx.TestDocumentIdxV1{
			DocFileName:  filename,
			DocFilePath:  filename,
			DocID:        filename,
			DocVer:       1,
			AddedTerms:   make(map[string]bool),
			DeletedTerms: make(map[string]bool),
		}
		documents = append(documents, doc)
	}

	err := createDocIndexAndSearchIdxV2(cred.owner, cred.docKey, cred.termKey, cred.indexKey, nil, documents)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(documents))
	for i, doc := range documents {
		result[i] = doc.DocID
	}

	return result, nil
}

// create doc index and search indexV2
func createDocIndexAndSearchIdxV2(
	owner scom.SearchIdxOwner,
	docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	oldDocs []*docidx.TestDocumentIdxV1,
	newDocs []*docidx.TestDocumentIdxV1) error {

	sources := make([]searchidxv2.SearchTermIdxSourceV2, 0, len(newDocs))
	for i, newDoc := range newDocs {
		// create doi & dti
		newDoc.CreateDoiAndDti(nil, docKey)

		newDoi, err := newDoc.OpenDoi(nil, docKey)
		if err != nil {
			return err
		}
		defer newDoi.Close()

		newDti, err := newDoc.OpenDti(nil, docKey)
		if err != nil {
			return err
		}
		defer newDti.Close()

		// create SearchTermIdxSourceV2
		if oldDocs == nil {
			source, err := searchidxv2.SearchTermIdxSourceCreateDoc(newDoi, newDti)
			if err != nil {
				return err
			}
			sources = append(sources, source)
		} else {
			oldDoc := oldDocs[i]
			oldDti, err := oldDoc.OpenDti(nil, docKey)
			if err != nil {
				return err
			}
			defer oldDti.Close()

			source, err := searchidxv2.SearchTermIdxSourceUpdateDoc(newDoi, oldDti, newDti)
			if err != nil {
				return err
			}
			sources = append(sources, source)
		}
	}

	// create search index writer
	siw, err := searchidxv2.CreateSearchIdxWriterV2(owner, termKey, indexKey, sources)
	if err != nil {
		return err
	}

	// process all terms
	termErr, err := siw.ProcessAllTerms(nil, nil)
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
		return err
	}

	for term, err := range termErr {
		if err != nil {
			fmt.Println(term, err.(*errors.Error).ErrorStack())
		}
	}

	return nil
}

func Search(cred *Credential, query string) ([]*DocumentResult, error) {
	terms := strings.Fields(query)

	reader, err := queryv1.OpenPhraseSearchV1(nil, cred.owner, terms,
		cred.termKey, cred.indexKey, scom.STI_V2)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var searchResult *queryv1.PhraseSearchResultV1 = nil
	for err == nil {
		var newResult *queryv1.PhraseSearchResultV1 = nil
		newResult, err = reader.GetNextResult()
		//fmt.Println("newResult=", newResult, "err=", err)
		if err != nil && err != io.EOF {
			return nil, err
		}

		searchResult = searchResult.Merge(newResult)
	}

	result := make([]*DocumentResult, len(searchResult.Docs))
	for i, doc := range searchResult.Docs {
		result[i] = &DocumentResult{doc.GetDocID(), 0}
	}

	return result, nil
}

func Cleanup(cred *Credential) error {
	docidx.CleanupTestDocumentsTmpFiles()
	scom.RemoveSearchIndex(nil, cred.owner)
	return nil
}
