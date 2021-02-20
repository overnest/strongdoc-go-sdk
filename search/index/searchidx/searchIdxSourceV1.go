package searchidx

import (
	"fmt"
	"io"
	"sort"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docoffsetidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/doctermidx"
)

type SearchIdxSourceBlockV1 struct {
	DocID       string
	DocVer      uint64
	FilterTerms []string
	TermOffset  map[string][]uint64
}

// SearchIdxSourceV1 is the Search Index Source V1
type SearchIdxSourceV1 interface {
	GetDocID() string
	GetDocVer() uint64
	GetAddTerms() []string
	GetDelTerms() []string
	// Returns io.EOF error if there are no more blocks
	GetNextSourceBlock(filterTerms []string) (*SearchIdxSourceBlockV1, error)
	Reset() error
	Close() error
	fmt.Stringer
}

type searchIdxSourceV1 struct {
	doi      docoffsetidx.DocOffsetIdx
	dtiOld   doctermidx.DocTermIdx
	dtiNew   doctermidx.DocTermIdx
	addTerms []string
	delTerms []string
}

// SearchSourceCreateDoc opens a search source when a new document is created
func SearchSourceCreateDoc(doi docoffsetidx.DocOffsetIdx, dti doctermidx.DocTermIdx) (SearchIdxSourceV1, error) {
	return createSearchSourceV1(doi, nil, dti)
}

// SearchSourceUpdateDoc opens a search source when an existing document is updated
func SearchSourceUpdateDoc(doi docoffsetidx.DocOffsetIdx, dtiOld, dtiNew doctermidx.DocTermIdx) (SearchIdxSourceV1, error) {
	return createSearchSourceV1(doi, dtiOld, dtiNew)
}

// SearchSourceDeleteDoc opens a search source when an existing document is deleted
func SearchSourceDeleteDoc(doi docoffsetidx.DocOffsetIdx, dti doctermidx.DocTermIdx) (SearchIdxSourceV1, error) {
	return createSearchSourceV1(doi, dti, nil)
}

func createSearchSourceV1(doi docoffsetidx.DocOffsetIdx, dtiOld, dtiNew doctermidx.DocTermIdx) (SearchIdxSourceV1, error) {
	source := &searchIdxSourceV1{doi, dtiOld, dtiNew, make([]string, 0), make([]string, 0)}

	// If there is not DTI given, then we'll figure out all the terms to be added from the DOI
	if dtiOld == nil && dtiNew == nil {
		terms := make(map[string]bool)

		switch doi.GetDoiVersion() {
		case docoffsetidx.DOI_V1:
			doiv1 := doi.(*docoffsetidx.DocOffsetIdxV1)

			var err error = nil
			for err == nil {
				var blk *docoffsetidx.DocOffsetIdxBlkV1
				blk, err = doiv1.ReadNextBlock()
				if err != nil && err != io.EOF {
					return nil, err
				}
				if blk != nil {
					for term := range blk.TermLoc {
						terms[term] = true
					}
				}
			}
			err = doiv1.Reset()
			if err != nil {
				return nil, err
			}

			source.addTerms = make([]string, 0, len(terms))
			for term := range terms {
				source.addTerms = append(source.addTerms, term)
			}

			sort.Strings(source.addTerms)
		default:
			return nil, errors.Errorf("Document offset index version %v is not supported",
				doi.GetDoiVersion())
		}
	} else {
		addTerms, delTerms, err := doctermidx.DiffDocTermIdx(dtiOld, dtiNew)
		if err != nil {
			return nil, err
		}
		source.addTerms = addTerms
		source.delTerms = delTerms
	}

	return source, nil
}

// GetNextSourceBlock gets the next source block from DOI
func (sis *searchIdxSourceV1) GetNextSourceBlock(filterTerms []string) (*SearchIdxSourceBlockV1, error) {
	sisBlock := &SearchIdxSourceBlockV1{sis.GetDocID(), sis.GetDocVer(), filterTerms, make(map[string][]uint64)}

	switch sis.doi.GetDoiVersion() {
	case docoffsetidx.DOI_V1:
		doiv1, ok := sis.doi.(*docoffsetidx.DocOffsetIdxV1)
		if !ok {
			return nil, errors.Errorf("Document offset index is not version %v",
				sis.doi.GetDoiVersion())
		}

		blk, err := doiv1.ReadNextBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if filterTerms == nil || len(filterTerms) == 0 {
			sisBlock.TermOffset = blk.TermLoc
			return sisBlock, nil
		}

		filterTermMap := make(map[string]bool)
		for _, term := range filterTerms {
			filterTermMap[term] = true
		}

		for term, locs := range blk.TermLoc {
			if filterTermMap[term] {
				sisBlock.TermOffset[term] = locs
			}
		}

		return sisBlock, err
	default:
		return nil, errors.Errorf("Document offset index version %v is not supported",
			sis.doi.GetDoiVersion())
	}
}

func (sis *searchIdxSourceV1) GetDocID() string {
	return sis.doi.GetDocID()
}

func (sis *searchIdxSourceV1) GetDocVer() uint64 {
	return sis.doi.GetDocVersion()
}

func (sis *searchIdxSourceV1) GetAddTerms() []string {
	return sis.addTerms
}

func (sis *searchIdxSourceV1) GetDelTerms() []string {
	return sis.delTerms
}

func (sis *searchIdxSourceV1) Reset() error {
	switch sis.doi.GetDoiVersion() {
	case docoffsetidx.DOI_V1:
		doiv1, ok := sis.doi.(*docoffsetidx.DocOffsetIdxV1)
		if !ok {
			return errors.Errorf("Document offset index is not version %v",
				sis.doi.GetDoiVersion())
		}
		return doiv1.Reset()
	default:
		return errors.Errorf("Document offset index version %v is not supported",
			sis.doi.GetDoiVersion())
	}
}

func (sis *searchIdxSourceV1) Close() (err error) {
	var err1, err2, err3 error
	err = nil

	if sis.doi != nil {
		err1 = sis.doi.Close()
	}

	if sis.dtiOld != nil {
		err2 = sis.dtiOld.Close()
	}

	if sis.dtiNew != nil {
		err3 = sis.dtiNew.Close()
	}

	if err1 != nil {
		err = err1
		return
	}

	if err2 != nil {
		err = err2
		return
	}

	if err3 != nil {
		err = err3
		return
	}

	return
}

func (sis *searchIdxSourceV1) String() string {
	return fmt.Sprintf("%v_%v", sis.GetDocID(), sis.GetDocVer())
}

func (sisb *SearchIdxSourceBlockV1) String() string {
	return fmt.Sprintf("%v_%v", sisb.DocID, sisb.DocVer)
}
