package searchidxv2

import (
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"io"
)

//////////////////////////////////////////////////////////////////
//
//                          Search Term Index
//
//////////////////////////////////////////////////////////////////

// SearchTermIdxV1 is a structure for search term index V1
type SearchTermIdxV2 struct {
	common.StiVersionS
	Term       string
	Owner      common.SearchIdxOwner
	OldSti     common.SearchTermIdx
	DelDocs    *DeletedDocsV1
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	IndexNonce []byte
	InitOffset uint64
	termID     string
	updateID   string
	writer     io.WriteCloser
	reader     io.ReadCloser
	bwriter    ssblocks.BlockListWriterV1
	breader    ssblocks.BlockListReaderV1
	storeSize  uint64
}
