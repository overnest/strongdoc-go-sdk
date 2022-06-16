package apidef

const (
	DOC_IDX_V1      = int64(1)
	SER_IDX_V1      = int64(1)
	DOC_IDX_VER_CUR = DOC_IDX_V1
	SER_IDX_VER_CUR = SER_IDX_V1

	DOC_IDX_OFFSET      = DocIdxType("docidx_offset")
	DOC_IDX_TERM        = DocIdxType("docidx_term")
	SEARCH_IDX_TERM_OFF = SearchIdxType("searchidx_term_off")
	SEARCH_IDX_SORT_DOC = SearchIdxType("searchidx_sort_doc")
)

var (
	doc_idx_types    = []DocIdxType{DOC_IDX_OFFSET, DOC_IDX_TERM}
	search_idx_types = []SearchIdxType{SEARCH_IDX_TERM_OFF, SEARCH_IDX_SORT_DOC}
)

type DocIdxType string
type SearchIdxType string

func (t DocIdxType) String() string {
	return string(t)
}

func (t SearchIdxType) String() string {
	return string(t)
}

type SearchIdxV1 struct {
	TermID   string
	UpdateID string
	Type     SearchIdxType
}
