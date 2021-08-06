package queryv1

import (
	"io"
	"sort"
	"strings"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	qcommon "github.com/overnest/strongdoc-go-sdk/search/query/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

type QueryCompV1 interface {
	IsComplete() bool
	SetComplete(complete bool)
	GetOp() qcommon.QueryOp
	GetTerms() []string
	GetResult() QueryResultV1
	SetResult(result QueryResultV1)
	GetChildComps() []QueryCompV1
	GetChildResults() []QueryResultV1
}

type queryCompV1 struct {
	complete bool
	op       qcommon.QueryOp
	terms    []string
	children []QueryCompV1
	result   QueryResultV1
}

func (qo *queryCompV1) IsComplete() bool {
	if !qo.complete && len(qo.GetChildComps()) > 0 &&
		(qo.op == qcommon.QueryAnd || qo.op == qcommon.QueryOr) {

		allComplete := true
		for _, child := range qo.children {
			allComplete = (allComplete || child.IsComplete())
		}
		qo.complete = allComplete
	}
	return qo.complete
}

func (qo *queryCompV1) SetComplete(complete bool) {
	qo.complete = complete
}

func (qo *queryCompV1) GetOp() qcommon.QueryOp {
	return qo.op
}

func (qo *queryCompV1) GetTerms() []string {
	return qo.terms
}

func (qo *queryCompV1) GetResult() QueryResultV1 {
	return qo.result
}

func (qo *queryCompV1) SetResult(result QueryResultV1) {
	qo.result = result
	if qo.result != nil {
		qo.SetComplete(true)
	}
}

func (qo *queryCompV1) GetChildComps() []QueryCompV1 {
	return qo.children
}

func (qo *queryCompV1) GetChildResults() []QueryResultV1 {
	if qo.IsComplete() && len(qo.GetChildComps()) > 0 &&
		(qo.op == qcommon.QueryAnd || qo.op == qcommon.QueryOr) {

		results := make([]QueryResultV1, 0, len(qo.children))
		for _, child := range qo.GetChildComps() {
			results = append(results, child.GetResult())
		}
		return results
	}
	return nil
}

func NewQueryCompErrV1(op qcommon.QueryOp, terms []string, children []QueryCompV1) (QueryCompV1, error) {
	switch op {
	case qcommon.Phrase, qcommon.TermAnd, qcommon.TermOr:
		if len(terms) == 0 {
			return nil, errors.Errorf("The operation %v requires at least 1 term", op)
		}
		return &queryCompV1{
			complete: false,
			op:       op,
			terms:    terms,
			children: nil,
			result:   nil,
		}, nil
	case qcommon.QueryAnd, qcommon.QueryOr:
		if len(children) == 0 {
			return nil, errors.Errorf("The operation %v requires at least 1 child query component", op)
		}
		return &queryCompV1{
			complete: false,
			op:       op,
			terms:    nil,
			children: children,
			result:   nil,
		}, nil
	default:
		return nil, errors.Errorf("Unsupported operation %v", op)
	}
}

func NewQueryCompV1(op qcommon.QueryOp, terms []string, children []QueryCompV1) QueryCompV1 {
	qc, err := NewQueryCompErrV1(op, terms, children)
	if err != nil {
		return nil
	}
	return qc
}

type QueryDocVerV1 interface {
	GetDocID() string
	GetDocVer() uint64
}

type queryDocVerV1 struct {
	docID  string
	docVer uint64
}

func (qdv *queryDocVerV1) GetDocID() string {
	return qdv.docID
}

func (qdv *queryDocVerV1) GetDocVer() uint64 {
	return qdv.docVer
}

type QueryResultV1 interface {
	GetDocVers() []QueryDocVerV1 // List sorted by DocID
	SetDocVers(docVers []QueryDocVerV1)
	Clone() QueryResultV1
}

type queryResultV1 struct {
	docVers []QueryDocVerV1
}

func (qrv *queryResultV1) GetDocVers() []QueryDocVerV1 {
	return qrv.docVers
}

func (qrv *queryResultV1) SetDocVers(docVers []QueryDocVerV1) {
	qrv.docVers = docVers
}

func (qrv *queryResultV1) Clone() QueryResultV1 {
	result := &queryResultV1{
		docVers: make([]QueryDocVerV1, len(qrv.docVers)),
	}
	for i, docVer := range qrv.docVers {
		result.docVers[i] = docVer
	}
	return result
}

func QueryTermsAndV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, terms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, searchIndexVer uint32) (QueryResultV1, error) {

	queryResult := &queryResultV1{
		docVers: []QueryDocVerV1{},
	}

	if len(terms) == 0 {
		return queryResult, nil
	}

	reader, err := OpenTermSearchV1(sdc, owner, terms, termKey, indexKey, searchIndexVer)
	if err != nil {
		return queryResult, err
	}
	defer reader.Close()

	var termResult *TermSearchResultV1 = nil
	for err == nil {
		var newResult *TermSearchResultV1 = nil
		newResult, err = reader.GetNextResult()
		if err != nil && err != io.EOF {
			return nil, err
		}
		termResult = termResult.Merge(newResult)
	}

	if termResult != nil {
		queryResult.docVers = queryTermsAndV1(terms, "", termResult)
	}

	return queryResult, nil
}

func queryTermsAndV1(terms []string, prevDocID string, termResult *TermSearchResultV1) []QueryDocVerV1 {
	result := []QueryDocVerV1{}
	if len(terms) == 0 {
		return result
	}

	term := terms[0]
	docIDs := termResult.TermDocIDs[term]
	for len(docIDs) > 0 {
		docID := docIDs[0]
		docVer := termResult.TermDocVer[term][docID]

		if len(prevDocID) > 0 {
			comp := strings.Compare(docID, prevDocID)
			if comp == 0 {
				if len(terms) == 1 { // last term
					result = append(result, &queryDocVerV1{docID, docVer})
				} else {
					result = append(result, queryTermsAndV1(terms[1:], docID, termResult)...)
				}
			} else if comp > 0 {
				break
			}
		} else {
			if len(terms) == 1 {
				result = append(result, &queryDocVerV1{docID, docVer})
			} else {
				result = append(result, queryTermsAndV1(terms[1:], docID, termResult)...)
			}
		}

		docIDs = docIDs[1:]
		termResult.TermDocIDs[term] = docIDs
	}

	return result
}

func QueryTermsOrV1(sdc client.StrongDocClient, owner common.SearchIdxOwner, terms []string,
	termKey, indexKey *sscrypto.StrongSaltKey, searchIndexVer uint32) (QueryResultV1, error) {

	queryResult := &queryResultV1{
		docVers: []QueryDocVerV1{},
	}

	if len(terms) == 0 {
		return queryResult, nil
	}

	reader, err := OpenTermSearchV1(sdc, owner, terms, termKey, indexKey, searchIndexVer)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var termResult *TermSearchResultV1 = nil
	for err == nil {
		var newResult *TermSearchResultV1 = nil
		newResult, err = reader.GetNextResult()
		if err != nil && err != io.EOF {
			return nil, err
		}
		termResult = termResult.Merge(newResult)
	}

	if termResult != nil {
		docVers := make([]QueryDocVerV1, 0, 1000)
		docIDMap := make(map[string]bool)
		for term, docIDs := range termResult.TermDocIDs {
			docVerMap := termResult.TermDocVer[term]
			for _, docID := range docIDs {
				if !docIDMap[docID] {
					docIDMap[docID] = true
					docVers = append(docVers, &queryDocVerV1{docID, docVerMap[docID]})
				}
			}
		}
		sort.Slice(docVers, func(i int, j int) bool {
			return (strings.Compare(docVers[i].GetDocID(), docVers[j].GetDocID()) < 0)
		})
		queryResult.docVers = docVers
	}

	return queryResult, nil
}

func QueryResultsAndV1(results []QueryResultV1) (QueryResultV1, error) {
	queryResult := &queryResultV1{
		docVers: []QueryDocVerV1{},
	}

	if len(results) == 0 {
		return queryResult, nil
	}

	cloneResults := make([]QueryResultV1, len(results))
	for i, result := range results {
		cloneResults[i] = result.Clone()
	}

	queryResult.docVers = queryResultsAndV1(cloneResults, nil)
	return queryResult, nil
}

func queryResultsAndV1(results []QueryResultV1, prevDocVer QueryDocVerV1) []QueryDocVerV1 {
	queryResult := []QueryDocVerV1{}
	if len(results) == 0 {
		return queryResult
	}

	result := results[0]
	docVers := result.GetDocVers()
	for len(docVers) > 0 {
		docVer := docVers[0]

		if prevDocVer != nil {
			comp := strings.Compare(prevDocVer.GetDocID(), docVer.GetDocID())
			if comp == 0 {
				if len(results) == 1 { // last result
					queryResult = append(queryResult, docVer)
				} else {
					queryResult = append(queryResult, queryResultsAndV1(results[1:], docVer)...)
				}
			} else if comp > 0 {
				break
			}
		} else {
			queryResult = append(queryResult, queryResultsAndV1(results[1:], docVer)...)
		}

		docVers = docVers[1:]
		result.SetDocVers(docVers)
	}

	return queryResult
}

func QueryResultsOr(results []QueryResultV1) (QueryResultV1, error) {
	queryResult := &queryResultV1{
		docVers: []QueryDocVerV1{},
	}

	if len(results) == 0 {
		return queryResult, nil
	}

	resultDocVer := make([]QueryDocVerV1, 0, 1000)
	docIDMap := make(map[string]bool)
	for _, result := range results {
		docVers := result.GetDocVers()
		for _, docVer := range docVers {
			if !docIDMap[docVer.GetDocID()] {
				docIDMap[docVer.GetDocID()] = true
				resultDocVer = append(resultDocVer, docVer)
			}
		}
	}

	sort.Slice(resultDocVer, func(i int, j int) bool {
		return (strings.Compare(resultDocVer[i].GetDocID(), resultDocVer[j].GetDocID()) < 0)
	})
	queryResult.docVers = resultDocVer

	return queryResult, nil
}
