package docidxv1

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

//////////////////////////////////////////////////////////////////
//
//                   Document Offset Index
//
//////////////////////////////////////////////////////////////////

// CreateDocOffsetIdx creates a new document offset index for writing
func CreateDocOffsetIdx(docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	store interface{}, initOffset int64) (*docidxv1.DocOffsetIdxV1, error) {

	return docidxv1.CreateDocOffsetIdxV1(docID, docVer, key, store, initOffset)
}

// OpenDocOffsetIdx opens a document offset index for reading
func OpenDocOffsetIdx(key *sscrypto.StrongSaltKey, store interface{}, initOffset int64) (common.DocOffsetIdx, error) {
	reader, ok := store.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}

	plainHdr, parsed, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, errors.New(err)
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, errors.New(err)
	}

	version, err := common.DeserializeDoiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetDoiVersion() {
	case common.DOI_V1:
		// Parse plaintext header body
		plainHdrBody, err := docidxv1.DoiPlainHdrBodyV1Deserialize(plainHdrBodyData)
		if err != nil {
			return nil, errors.New(err)
		}
		return docidxv1.OpenDocOffsetIdxPrivV1(key, plainHdrBody, reader, initOffset+int64(parsed))
	default:
		return nil, errors.Errorf("Document offset index version %v is not supported",
			version.GetDoiVersion())
	}
}
