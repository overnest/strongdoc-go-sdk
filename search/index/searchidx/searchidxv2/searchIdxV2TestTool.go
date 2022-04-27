package searchidxv2

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
)

const (
	TEST_BUCKET_COUNT uint32 = common.STI_TERM_BUCKET_COUNT
)

// create doc index and search indexV2
func TestCreateDocIndexAndSearchIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner,
	docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	oldDocs []*docidx.TestDocumentIdxV1,
	newDocs []*docidx.TestDocumentIdxV1) {

	sources := make([]SearchTermIdxSourceV2, 0, len(newDocs))
	for i, newDoc := range newDocs {
		// create doi & dti
		assert.NilError(t, newDoc.CreateDoiAndDti(sdc, docKey))
		newDoi, err := newDoc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)
		defer newDoi.Close()
		newDti, err := newDoc.OpenDti(sdc, docKey)
		assert.NilError(t, err)
		defer newDti.Close()

		// create SearchTermIdxSourceV2
		if oldDocs == nil {
			source, err := SearchTermIdxSourceCreateDoc(newDoi, newDti)
			assert.NilError(t, err)
			sources = append(sources, source)
		} else {
			oldDoc := oldDocs[i]
			oldDti, err := oldDoc.OpenDti(sdc, docKey)
			assert.NilError(t, err)
			defer oldDti.Close()

			source, err := SearchTermIdxSourceUpdateDoc(newDoi, oldDti, newDti)
			assert.NilError(t, err)
			sources = append(sources, source)
		}
	}

	// create search index writer
	siw, err := CreateSearchIdxWriterV2(owner, termKey, indexKey, sources)
	assert.NilError(t, err)

	// process all terms
	termErr, err := siw.ProcessAllTerms(sdc, nil)
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
	}
	assert.NilError(t, err)

	for term, err := range termErr {
		if err != nil {
			fmt.Println(term, err.(*errors.Error).ErrorStack())
		}
		assert.NilError(t, err)
	}
}

// validate created search index
func TestValidateSearchIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, docKey, termKey, indexKey *sscrypto.StrongSaltKey,
	newDocs []*docidx.TestDocumentIdxV1) {

	analyzer := GetSearchTermIdxAnalyzer()

	doiInfo := getDoiInfo(t, sdc, docKey, newDocs)
	for _, doiTerm := range doiInfo.doiTerms {
		analzdTerm := analyzer.Analyze(doiTerm)[0]

		stiDocLocMap := make(map[*docidx.TestDocumentIdxV1][]uint64) // doc -> locations
		for doc, doiAllTermLocMap := range doiInfo.doiAllTermLocMap {
			if locs, exist := doiAllTermLocMap[doiTerm]; exist {
				stiDocLocMap[doc] = locs
			}
		}
		ssdiDocLocMap := make(map[*docidx.TestDocumentIdxV1][]uint64) // doc -> locations
		for doc, doiAnalTermLocMap := range doiInfo.doiAnalTermLocMap {
			if locs, exist := doiAnalTermLocMap[analzdTerm]; exist {
				ssdiDocLocMap[doc] = locs
			}
		}
		// TestValidateSearchTermIdxV2(t, sdc, owner, termKey, indexKey, doiTerm, stiDocLocMap)
		// TestValidateSortedDocIdxV2(t, sdc, owner, termKey, indexKey, doiTerm, ssdiDocLocMap)
	}

	for term, docs := range doiInfo.doiDelTermMap {
		TestValidateDeletedSearchTermIdxV2(t, sdc, owner, termKey, indexKey, term, docs)
		TestValidateDeletedSortedDocIdxV2(t, sdc, owner, termKey, indexKey, term, docs)
	}
}

// validate search term index
func TestValidateSearchTermIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	doiTerm string, doiDocLocMap map[*docidx.TestDocumentIdxV1][]uint64) {

	termID, err := GetSearchTermID(doiTerm, termKey, TEST_BUCKET_COUNT)
	assert.NilError(t, err)

	updateID, err := GetLatestUpdateIDV2(sdc, owner, termID)
	assert.NilError(t, err)

	// open term index
	sti, err := OpenSearchTermIdxV2(sdc, owner, termID, termKey, indexKey, updateID)
	assert.NilError(t, err)

	stiAnalyzer, err := tokenizer.OpenAnalyzer(sti.TokenizerType)
	assert.NilError(t, err)

	stiAnalzdTerm := stiAnalyzer.Analyze(doiTerm)[0]

	for err == nil {
		var blk *SearchTermIdxBlkV2
		blk, err = sti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if blk == nil {
			continue
		}

		if _, exists := blk.TermDocVerOffset[doiTerm]; !exists {
			continue
		}
		//fmt.Println("check the block")

		for doiDoc, doiLocs := range doiDocLocMap {
			verOff := blk.TermDocVerOffset[doiTerm][doiDoc.DocID]
			if verOff == nil {
				continue
			}

			//fmt.Println("check docID", doc.DocID)
			//fmt.Println("locs", locs)
			//fmt.Println("verOff.Offsets", verOff.Offsets)

			assert.Equal(t, doiDoc.DocVer, verOff.Version)
			if len(doiLocs) < len(verOff.Offsets) {
				fmt.Println("PROBLEM: validate sti(doiTerm/analzdTerm): ", doiTerm, stiAnalzdTerm, termID, doiDoc.DocID)
				fmt.Println("doiLocs: ", doiLocs)
				fmt.Println("verOff.Offsets: ", verOff.Offsets)
			}
			assert.Assert(t, len(doiLocs) >= len(verOff.Offsets))
			if !reflect.DeepEqual(verOff.Offsets, doiLocs[:len(verOff.Offsets)]) {
				fmt.Println("PROBLEM: validate sti(doiTerm/analzdTerm): ", doiTerm, stiAnalzdTerm, termID, doiDoc.DocID)
			}
			assert.DeepEqual(t, verOff.Offsets, doiLocs[:len(verOff.Offsets)])
			doiDocLocMap[doiDoc] = doiLocs[len(verOff.Offsets):]
		}
	}

	err = sti.Close()
	assert.NilError(t, err)
}

// validate deleted search term index
func TestValidateDeletedSearchTermIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, delDocs []*docidx.TestDocumentIdxV1) {

	termID, err := GetSearchTermID(term, termKey, TEST_BUCKET_COUNT)
	assert.NilError(t, err)

	updateID, err := GetLatestUpdateIDV2(sdc, owner, termID)
	assert.NilError(t, err)

	sti, err := OpenSearchTermIdxV2(sdc, owner, termID, termKey, indexKey, updateID)
	assert.NilError(t, err)

	for err == nil {
		var blk *SearchTermIdxBlkV2
		blk, err = sti.ReadNextBlock()
		if err != nil {
			assert.Equal(t, err, io.EOF)
		}

		if blk == nil {
			continue
		}

		if _, exists := blk.TermDocVerOffset[term]; !exists {
			continue
		}

		for _, delDoc := range delDocs {
			assert.Assert(t, blk.TermDocVerOffset[term][delDoc.DocID] == nil)
		}
	}

	err = sti.Close()
	assert.NilError(t, err)
}

// validate sorted doc index
func TestValidateSortedDocIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	doiTerm string, doiDocLocMap map[*docidx.TestDocumentIdxV1][]uint64) {

	termID, err := GetSearchTermID(doiTerm, termKey, TEST_BUCKET_COUNT)
	assert.NilError(t, err)

	updateID, err := GetLatestUpdateIDV2(sdc, owner, termID)
	assert.NilError(t, err)

	ssdi, err := OpenSearchSortDocIdxV2(sdc, owner, termID, termKey, indexKey, updateID)
	assert.NilError(t, err)

	ssdiAnalyzer, err := tokenizer.OpenAnalyzer(ssdi.TokenizerType)
	assert.NilError(t, err)

	ssdiAnalzdTerm := ssdiAnalyzer.Analyze(doiTerm)[0]

	doiDocIDs := make([]string, 0, len(doiDocLocMap))
	for doiDoc := range doiDocLocMap {
		doiDocIDs = append(doiDocIDs, doiDoc.DocID)
	}

	ssdiDocIdVerMap, err := ssdi.FindDocIDs(doiTerm, doiDocIDs)
	assert.NilError(t, err)

	for doiDoc := range doiDocLocMap {
		ssdiDocIDVer := ssdiDocIdVerMap[doiDoc.DocID]
		if ssdiDocIDVer == nil {
			fmt.Println("PROBLEM: validate ssdi(doiTerm/analzdTerm/doiDoc.DocID):", doiTerm, ssdiAnalzdTerm, doiDoc.DocID)
		}

		assert.Assert(t, ssdiDocIDVer != nil, "Did not find the proper document")
		if ssdiDocIDVer != nil {
			assert.Equal(t, doiDoc.DocVer, ssdiDocIDVer.DocVer)
		}
	}

	err = ssdi.Close()
	assert.NilError(t, err)
}

// validate deleted sorted doc index
func TestValidateDeletedSortedDocIdxV2(t *testing.T, sdc client.StrongDocClient,
	owner common.SearchIdxOwner, termKey, indexKey *sscrypto.StrongSaltKey,
	term string, delDocs []*docidx.TestDocumentIdxV1) {

	termID, err := GetSearchTermID(term, termKey, TEST_BUCKET_COUNT)
	assert.NilError(t, err)

	updateIDs, err := GetUpdateIdsV2(sdc, owner, termID)
	assert.NilError(t, err)

	ssdi, err := OpenSearchSortDocIdxV2(sdc, owner, termID, termKey, indexKey, updateIDs[0])
	if err != nil {
		assert.Equal(t, err, os.ErrNotExist)
		return
	}

	docIDs := make([]string, 0, len(delDocs))
	for _, doc := range delDocs {
		docIDs = append(docIDs, doc.DocID)
	}

	docIdVerMap, err := ssdi.FindDocIDs(term, docIDs)
	assert.NilError(t, err)

	for _, doc := range delDocs {
		assert.Assert(t, docIdVerMap[doc.DocID] == nil)
	}

	err = ssdi.Close()
	assert.NilError(t, err)
}

type doiInfo struct {
	doiTerms          []string                                          // list of terms
	doiTermMap        map[string]bool                                   // term -> bool
	doiTermLocMap     map[*docidx.TestDocumentIdxV1]map[string][]uint64 // doc -> (term -> []locations)
	doiDelTermMap     map[string][]*docidx.TestDocumentIdxV1            // term -> []doc
	doiAllTermLocMap  map[*docidx.TestDocumentIdxV1]map[string][]uint64 // doc -> (allTerm -> []location)
	doiAnalTermLocMap map[*docidx.TestDocumentIdxV1]map[string][]uint64 // doc -> (analzdTerm -> []location)
}

func getDoiInfo(t *testing.T, sdc client.StrongDocClient, docKey *sscrypto.StrongSaltKey,
	newDocs []*docidx.TestDocumentIdxV1) *doiInfo {

	analyzer := GetSearchTermIdxAnalyzer()

	doiInfo := &doiInfo{
		doiTerms:          nil,                                                     // list of terms
		doiTermMap:        make(map[string]bool),                                   // term -> bool
		doiTermLocMap:     make(map[*docidx.TestDocumentIdxV1]map[string][]uint64), // doc -> (term -> []locations)
		doiDelTermMap:     make(map[string][]*docidx.TestDocumentIdxV1),            // term -> []doc
		doiAllTermLocMap:  make(map[*docidx.TestDocumentIdxV1]map[string][]uint64), // doc -> (allTerm -> []location)
		doiAnalTermLocMap: make(map[*docidx.TestDocumentIdxV1]map[string][]uint64), // doc -> (analzdTerm -> []location)
	}

	// open document offset index
	for _, doc := range newDocs {
		doi, err := doc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)

		source, err := SearchTermIdxSourceCreateDoc(doi, nil)
		assert.NilError(t, err)

		sourceTermLoc := make(map[string][]uint64)
		sourceTerms := source.GetAllTerms()
		for {
			sourceBlock, err := source.GetNextSourceBlock(nil)
			if err != nil {
				assert.Assert(t, err == io.EOF)
			}

			if sourceBlock != nil {
				for term, offset := range sourceBlock.TermOffset {
					sourceTermLoc[term] = append(sourceTermLoc[term], offset...)
				}
			}

			if err == io.EOF {
				break
			}
		}

		doiInfo.doiTermLocMap[doc] = sourceTermLoc
		for _, sourceTerm := range sourceTerms {
			doiInfo.doiTermMap[sourceTerm] = true
		}

		analTermLoc := make(map[string][]uint64)
		for term, locations := range sourceTermLoc {
			analzdTerm := analyzer.Analyze(term)[0]
			analTermLoc[analzdTerm] = append(analTermLoc[analzdTerm], locations...)
		}
		for _, locations := range analTermLoc {
			sort.Slice(locations, func(i, j int) bool { return locations[i] < locations[j] })
		}
		doiInfo.doiAnalTermLocMap[doc] = analTermLoc

		allTermLoc := make(map[string][]uint64)
		for term := range sourceTermLoc {
			analzdTerm := analyzer.Analyze(term)[0]
			allTermLoc[analzdTerm] = analTermLoc[analzdTerm]
			allTermLoc[term] = analTermLoc[analzdTerm]
		}
		doiInfo.doiAllTermLocMap[doc] = allTermLoc

		err = source.Close()
		assert.NilError(t, err)

		for term := range doc.DeletedTerms {
			doiInfo.doiDelTermMap[term] = append(doiInfo.doiDelTermMap[term], doc)
		}
	}

	doiInfo.doiTerms = make([]string, 0, len(doiInfo.doiTermMap))
	for term := range doiInfo.doiTermMap {
		doiInfo.doiTerms = append(doiInfo.doiTerms, term)
	}
	sort.Strings(doiInfo.doiTerms)

	return doiInfo
}
