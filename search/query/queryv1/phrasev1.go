package queryv1

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"

	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

type PhraseSearchV1 struct {
	sdc       client.StrongDocClient
	owner     common.SearchIdxOwner
	terms     []string
	origTerms []string
	termKey   *sscrypto.StrongSaltKey
	indexKey  *sscrypto.StrongSaltKey
	reader    searchidx.StiReader
	termData  map[string]*termDataV1 // term -> termDataV1
}

type termDataV1 struct {
	term       string
	eof        bool
	docOffsets map[string]*docOffsetsV1 // docID -> docOffsetsV1
}

type docOffsetsV1 struct {
	docID   string
	docVer  uint64
	offsets []uint64
}

type PhraseSearchResultV1 struct {
	Terms  []string
	Docs   []*PhraseSearchDocV1
	DocMap map[string]*PhraseSearchDocV1 // docID -> PhraseSearchDocV1
}

type PhraseSearchDocV1 struct {
	DocID         string
	DocVer        uint64
	PhraseOffsets []uint64 // Offset of the first term in the phrase
}

func OpenPhraseSearchV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, terms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, searchIndexVer uint32) (*PhraseSearchV1, error) {

	analyzer, err := tokenizer.OpenBleveAnalyzer()
	if err != nil {
		return nil, err
	}

	searchTerms := make([]string, len(terms))
	for i, term := range terms {
		tokens := analyzer.Analyze([]byte(term))
		if len(tokens) != 1 {
			return nil, errors.Errorf("The search term %v fails analysis:%v", term, tokens)
		}
		searchTerms[i] = string(tokens[0].Term)
	}

	reader, err := searchidx.OpenSearchTermIndex(sdc, owner, searchTerms, termKey, indexKey, searchIndexVer)
	if err != nil {
		return nil, err
	}

	phraseSearch := &PhraseSearchV1{
		sdc:       sdc,
		owner:     owner,
		terms:     searchTerms,
		origTerms: terms,
		termKey:   termKey,
		indexKey:  indexKey,
		reader:    reader,
		termData:  make(map[string]*termDataV1), // term -> termDataV1
	}

	return phraseSearch, nil
}

func (search *PhraseSearchV1) GetNextResult() (*PhraseSearchResultV1, error) {
	if len(search.terms) == 0 {
		return nil, io.EOF
	}

	stiData, err := search.reader.ReadNextData()
	//fmt.Println("ReadNextData", stiData, err)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}

	err = search.mergeStiData(stiData)
	//fmt.Println("mergeStiData", err)

	if err != nil {
		return nil, err
	}

	result := &PhraseSearchResultV1{
		Terms:  search.origTerms,
		Docs:   make([]*PhraseSearchDocV1, 0, 100),
		DocMap: make(map[string]*PhraseSearchDocV1), // docID -> PhraseSearchDocV1
	}

	isEOF := false
	for _, term := range search.terms {
		termData := search.termData[term]
		if termData != nil {
			// If even one term is EOF, there can't be any more matches after this
			isEOF = isEOF || termData.eof
		}
	}

	if isEOF {
		err = io.EOF
	}

	term := search.terms[0]
	termData := search.termData[term]
	if termData == nil {
		return result, err
	}

	for docID, docOffsets := range termData.docOffsets {
		if docOffsets == nil {
			continue
		}

		// If there is only 1 term, then just use all offsets
		if len(search.terms) == 1 {
			if len(docOffsets.offsets) > 0 {
				searchDoc := &PhraseSearchDocV1{
					DocID:         docOffsets.docID,
					DocVer:        docOffsets.docVer,
					PhraseOffsets: docOffsets.offsets,
				}

				result.Docs = append(result.Docs, searchDoc)
				result.DocMap[searchDoc.DocID] = searchDoc
			}
			continue
		}

		matchOffsets := search.phraseSearch(search.terms[1:], docID, docOffsets.offsets)
		for i := range matchOffsets {
			matchOffsets[i]--
		}
		docOffsets.offsets = docOffsets.offsets[len(docOffsets.offsets):]

		if len(matchOffsets) > 0 {
			searchDoc := &PhraseSearchDocV1{
				DocID:         docOffsets.docID,
				DocVer:        docOffsets.docVer,
				PhraseOffsets: matchOffsets,
			}

			result.Docs = append(result.Docs, searchDoc)
			result.DocMap[searchDoc.DocID] = searchDoc
		}
	}

	return result, err
}

func (search *PhraseSearchV1) mergeStiData(stiData searchidx.StiData) error {
	if stiData != nil {
		for _, term := range search.terms {
			termData := search.termData[term]
			if termData == nil {
				termData = &termDataV1{
					term:       term,
					eof:        false,
					docOffsets: make(map[string]*docOffsetsV1), // docID -> docOffsetsV1
				}
				search.termData[term] = termData
			}
			termData.eof = stiData.IsTermEOF(term)

			stiDocOffsets := stiData.GetTermDocOffsets(term)
			if stiDocOffsets == nil {
				continue
			}

			for _, docID := range stiDocOffsets.GetDocIDs() {
				docVer := stiDocOffsets.GetDocVer(docID)
				stiOffsets := stiDocOffsets.GetOffsets(docID)
				docOffsets := termData.docOffsets[docID]
				if docOffsets == nil {
					docOffsets = &docOffsetsV1{
						docID:   docID,
						docVer:  docVer,
						offsets: make([]uint64, 0, len(stiOffsets)),
					}
					termData.docOffsets[docID] = docOffsets
				}

				docOffsets.offsets = append(docOffsets.offsets, stiOffsets...)
			}
		} // for _, term := range search.terms
	} // if stiData != nil

	return nil
}

func (search *PhraseSearchV1) phraseSearch(terms []string, docID string, prevOffsets []uint64) (matchOffsets []uint64) {
	if len(terms) == 0 || len(prevOffsets) == 0 {
		return make([]uint64, 0)
	}

	term := terms[0]
	termData := search.termData[term]
	if termData == nil {
		return make([]uint64, 0)
	}
	docOffsets := termData.docOffsets[docID]
	if docOffsets == nil {
		return make([]uint64, 0)
	}

	nextOffsets := make([]uint64, 0, len(prevOffsets))
	for _, prevOffset := range prevOffsets {
		for len(docOffsets.offsets) > 0 {
			offset := docOffsets.offsets[0]
			if offset > prevOffset {
				if offset == prevOffset+1 {
					nextOffsets = append(nextOffsets, offset)
				}
				break
			}
			docOffsets.offsets = docOffsets.offsets[1:]
		}
	}

	if len(terms) > 1 { // Need to recurse
		matchOffsets = search.phraseSearch(terms[1:], docID, nextOffsets)
		for i := range matchOffsets {
			matchOffsets[i]--
		}
	} else { // This is the last term. No need to recurse
		matchOffsets = nextOffsets
	}

	return
}

func (search *PhraseSearchV1) Reset() error {
	return search.reader.Reset()
}

func (search *PhraseSearchV1) Close() error {
	return search.reader.Close()
}

func (result *PhraseSearchResultV1) GetMatchDocs() []*PhraseSearchDocV1 {
	return result.Docs
}

func (result *PhraseSearchResultV1) GetMatchDocMap() map[string]*PhraseSearchDocV1 {
	return result.DocMap
}

func (result *PhraseSearchResultV1) Merge(newResult *PhraseSearchResultV1) *PhraseSearchResultV1 {
	if newResult == nil {
		return result
	}

	if result == nil {
		return newResult
	}

	for newDocID, newDoc := range newResult.DocMap {
		doc := result.DocMap[newDocID]
		if doc == nil { // New Document doesn't already exist
			result.Docs = append(result.Docs, newDoc)
			result.DocMap[newDocID] = newDoc
		} else {
			doc.PhraseOffsets = append(doc.PhraseOffsets, newDoc.PhraseOffsets...)
		}
	}

	return result
}

func (doc *PhraseSearchDocV1) GetDocID() string {
	return doc.DocID
}

func (doc *PhraseSearchDocV1) GetDocVer() uint64 {
	return doc.DocVer
}

func (doc *PhraseSearchDocV1) GetPhraseOffsets() []uint64 {
	return doc.PhraseOffsets
}
