package queryv1

import (
	"io"
	"sort"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

type TermSearchV1 struct {
	sdc               client.StrongDocClient
	owner             common.SearchIdxOwner
	terms             []string
	origTerms         []string
	searchToOrigTerms map[string]string
	termKey           *sscrypto.StrongSaltKey
	indexKey          *sscrypto.StrongSaltKey
	reader            searchidx.SsdiReader
}

type TermSearchDocVerV1 struct {
	DocID  string
	DocVer uint64
}

type TermSearchResultV1 struct {
	Terms      []string
	TermDocIDs map[string][]string          // term -> []docID(Sorted)
	TermDocVer map[string]map[string]uint64 // term -> (docID -> docVer)
}

func OpenTermSearchV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, terms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, ssdiVer uint32) (*TermSearchV1, error) {

	analyzer, err := tokenizer.OpenBleveAnalyzer()
	if err != nil {
		return nil, err
	}

	searchTerms := make([]string, len(terms))
	searchToOrigTerms := make(map[string]string)
	for i, term := range terms {
		tokens := analyzer.Analyze([]byte(term))
		if len(tokens) != 1 {
			return nil, errors.Errorf("The search term %v fails analysis:%v", term, tokens)
		}
		searchTerms[i] = string(tokens[0].Term)
		searchToOrigTerms[searchTerms[i]] = term
	}

	reader, err := searchidx.OpenSearchSortedDocIndex(sdc, owner, searchTerms, termKey, indexKey, ssdiVer)
	if err != nil {
		return nil, err
	}

	termSearch := &TermSearchV1{
		sdc:               sdc,
		owner:             owner,
		terms:             searchTerms,
		origTerms:         terms,
		searchToOrigTerms: searchToOrigTerms,
		termKey:           termKey,
		indexKey:          indexKey,
		reader:            reader,
	}

	return termSearch, nil
}

func (search *TermSearchV1) GetNextResult() (*TermSearchResultV1, error) {
	if len(search.terms) == 0 {
		return nil, io.EOF
	}

	ssdiData, err := search.reader.ReadNextData()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}

	result := &TermSearchResultV1{
		Terms:      search.origTerms,
		TermDocIDs: make(map[string][]string),          // term -> []docID(Sorted)
		TermDocVer: make(map[string]map[string]uint64), // term -> (docID -> docVer)
	}

	if ssdiData != nil {
		for term, docs := range ssdiData.GetTermDocs() {
			origTerm := search.searchToOrigTerms[term]
			docIDs := make([]string, len(docs))
			result.TermDocIDs[origTerm] = docIDs
			docVer := make(map[string]uint64)
			result.TermDocVer[origTerm] = docVer

			for i, doc := range docs {
				docIDs[i] = doc.GetDocID()
				docVer[doc.GetDocID()] = doc.GetDocVer()
			}
		}
	}

	return result, err
}

func (search *TermSearchV1) Reset() error {
	return search.reader.Reset()
}

func (search *TermSearchV1) Close() error {
	return search.reader.Close()
}

func (result *TermSearchResultV1) Merge(newResult *TermSearchResultV1) *TermSearchResultV1 {
	if newResult == nil {
		return result
	}

	if result == nil {
		return newResult
	}

	for term, newDocVer := range newResult.TermDocVer {
		oldDocIDs := result.TermDocIDs[term]
		oldDocVer := result.TermDocVer[term]

		if len(oldDocVer) == 0 || len(oldDocIDs) == 0 {
			result.TermDocVer[term] = newDocVer
			result.TermDocIDs[term] = newResult.TermDocIDs[term]
		} else {
			for docID, newVer := range newDocVer {
				if oldVer, exist := oldDocVer[docID]; !exist {
					oldDocIDs = append(oldDocIDs, docID)
					oldDocVer[docID] = newVer
				} else {
					if oldVer != newVer {
						// TODO: Shouldn't happen
					}
				}
			}
			sort.Strings(oldDocIDs)
			result.TermDocIDs[term] = oldDocIDs
		}
	}

	return result
}

func (docver *TermSearchDocVerV1) GetDocID() string {
	return docver.DocID
}

func (docver *TermSearchDocVerV1) GetDocVer() uint64 {
	return docver.DocVer
}
