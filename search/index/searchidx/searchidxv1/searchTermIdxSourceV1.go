package searchidxv1

import (
	"fmt"
	"io"
	"sort"

	"github.com/go-errors/errors"
	docidx "github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	didxcommon "github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	didxv1 "github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
)

type SearchTermIdxSourceBlockV1 struct {
	DocID       string
	DocVer      uint64
	FilterTerms []string
	TermOffset  map[string][]uint64
}

// SearchTermIdxSourceV1 is the Search HashedTerm Index Source V1
type SearchTermIdxSourceV1 interface {
	GetDocID() string
	GetDocVer() uint64
	GetAddTerms() []string
	GetDelTerms() []string
	// Returns io.EOF error if there are no more blocks
	GetNextSourceBlock(filterTerms []string) (*SearchTermIdxSourceBlockV1, error)
	Reset() error
	Close() error
	fmt.Stringer
}

type searchTermIdxSourceV1 struct {
	doiNew   didxcommon.DocOffsetIdx
	dtiOld   didxcommon.DocTermIdx
	dtiNew   didxcommon.DocTermIdx
	addTerms []string
	delTerms []string
}

// SearchTermIdxSourceCreateDoc opens a search source when a new document is created
func SearchTermIdxSourceCreateDoc(doiNew didxcommon.DocOffsetIdx, dtiNew didxcommon.DocTermIdx) (SearchTermIdxSourceV1, error) {
	return createSearchTermIdxSourceV1(doiNew, nil, dtiNew)
}

// SearchTermIdxSourceUpdateDoc opens a search source when an existing document is updated
func SearchTermIdxSourceUpdateDoc(doiNew didxcommon.DocOffsetIdx, dtiOld, dtiNew didxcommon.DocTermIdx) (SearchTermIdxSourceV1, error) {
	return createSearchTermIdxSourceV1(doiNew, dtiOld, dtiNew)
}

// SearchTermIdxSourceDeleteDoc opens a search source when an existing document is deleted
func SearchTermIdxSourceDeleteDoc(doiDel didxcommon.DocOffsetIdx, dtiDel didxcommon.DocTermIdx) (SearchTermIdxSourceV1, error) {
	return createSearchTermIdxSourceV1(doiDel, dtiDel, nil)
}

func createSearchTermIdxSourceV1(doiNew didxcommon.DocOffsetIdx, dtiOld, dtiNew didxcommon.DocTermIdx) (SearchTermIdxSourceV1, error) {
	source := &searchTermIdxSourceV1{doiNew, dtiOld, dtiNew,
		make([]string, 0), make([]string, 0)}

	// If there is no DTI given, then we'll figure out all the terms to be added from the DOI
	if dtiOld == nil && dtiNew == nil {
		terms := make(map[string]bool)

		switch doiNew.GetDoiVersion() {
		case didxcommon.DOI_V1:
			doiv1 := doiNew.(*didxv1.DocOffsetIdxV1)

			var err error = nil
			for err == nil {
				var blk *didxv1.DocOffsetIdxBlkV1
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
				doiNew.GetDoiVersion())
		}
	} else {
		_, delTerms, err := docidx.DiffDocTermIdx(dtiOld, dtiNew)
		if err != nil {
			return nil, err
		}
		source.addTerms, err = docidx.GetAllTermList(dtiNew)
		if err != nil {
			return nil, err
		}
		source.delTerms = delTerms
	}

	return source, nil
}

// GetNextSourceBlock gets the next source block from DOI
func (sis *searchTermIdxSourceV1) GetNextSourceBlock(filterTerms []string) (*SearchTermIdxSourceBlockV1, error) {
	sisBlock := &SearchTermIdxSourceBlockV1{sis.GetDocID(), sis.GetDocVer(),
		filterTerms, make(map[string][]uint64)}

	switch sis.doiNew.GetDoiVersion() {
	case didxcommon.DOI_V1:
		doiv1, ok := sis.doiNew.(*didxv1.DocOffsetIdxV1)
		if !ok {
			return nil, errors.Errorf("Document offset index is not version %v",
				sis.doiNew.GetDoiVersion())
		}

		blk, err := doiv1.ReadNextBlock()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if blk != nil {
			if len(filterTerms) == 0 {
				sisBlock.TermOffset = blk.TermLoc
				return sisBlock, nil
			}

			filterTermMap := make(map[string]bool)
			for _, term := range filterTerms {
				filterTermMap[term] = true
			}

			for term, locs := range blk.TermLoc {
				if filterTermMap[term] && locs != nil && len(locs) > 0 {
					sisBlock.TermOffset[term] = locs
				}
			}
		}

		return sisBlock, err
	default:
		return nil, errors.Errorf("Document offset index version %v is not supported",
			sis.doiNew.GetDoiVersion())
	}
}

func (sis *searchTermIdxSourceV1) GetDocID() string {
	return sis.doiNew.GetDocID()
}

func (sis *searchTermIdxSourceV1) GetDocVer() uint64 {
	return sis.doiNew.GetDocVersion()
}

func (sis *searchTermIdxSourceV1) GetAddTerms() []string {
	return sis.addTerms
}

func (sis *searchTermIdxSourceV1) GetDelTerms() []string {
	return sis.delTerms
}

func (sis *searchTermIdxSourceV1) Reset() error {
	switch sis.doiNew.GetDoiVersion() {
	case didxcommon.DOI_V1:
		doiv1, ok := sis.doiNew.(*didxv1.DocOffsetIdxV1)
		if !ok {
			return errors.Errorf("Document offset index is not version %v",
				sis.doiNew.GetDoiVersion())
		}
		return doiv1.Reset()
	default:
		return errors.Errorf("Document offset index version %v is not supported",
			sis.doiNew.GetDoiVersion())
	}
}

func (sis *searchTermIdxSourceV1) Close() (err error) {
	var err1, err2, err3 error
	err = nil

	if sis.doiNew != nil {
		err1 = sis.doiNew.Close()
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

func (sis *searchTermIdxSourceV1) String() string {
	return fmt.Sprintf("%v_%v", sis.GetDocID(), sis.GetDocVer())
}

func (sisb *SearchTermIdxSourceBlockV1) String() string {
	return fmt.Sprintf("%v_%v", sisb.DocID, sisb.DocVer)
}
