package docidx

import (
	//"encoding/json"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"
	"io"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"

	"github.com/go-errors/errors"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

//////////////////////////////////////////////////////////////////
//
//                   Document Offset Index
//
//////////////////////////////////////////////////////////////////

// CreateDocOffsetIdx creates a new document offset index for writing
func CreateDocOffsetIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	initOffset int64) (*docidxv1.DocOffsetIdxV1, error) {
	return docidxv1.CreateDocOffsetIdxV1(sdc, docID, docVer, key, initOffset)
}

// Create document offset index and writes output
func CreateAndSaveDocOffsetIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourceData utils.Storage) error {
	return CreateAndSaveDocOffsetIdxWithOffset(sdc, docID, docVer, key, sourceData, 0)
}

func CreateAndSaveDocOffsetIdxWithOffset(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourceData utils.Storage, initOffset int64) error {
	tokenizer, err := utils.OpenFileTokenizer(sourceData)
	if err != err {
		return err
	}

	doi, err := CreateDocOffsetIdx(sdc, docID, docVer, key, initOffset)
	if err != nil {
		return err
	}

	for token, _, wordCounter, err := tokenizer.NextToken(); err != io.EOF; token, _, wordCounter, err = tokenizer.NextToken() {
		if err != nil && err != io.EOF {
			return err
		}
		addErr := doi.AddTermOffset(token, wordCounter)
		if addErr != nil {
			return addErr
		}
	}

	return doi.Close()
}

func OpenDocOffsetIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey) (common.DocOffsetIdx, error) {
	return OpenDocOffsetIdxWithOffset(sdc, docID, docVer, key, 0)
}

// OpenDocOffsetIdx opens a document offset index for reading
func OpenDocOffsetIdxWithOffset(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey, initOffset int64) (common.DocOffsetIdx, error) {
	reader, err := common.OpenDocOffsetIdxReader(sdc, docID, docVer)
	if err != nil {
		return nil, err
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
