package docidx

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"io"
)

//////////////////////////////////////////////////////////////////
//
//                Document Offset Index Version
//
//////////////////////////////////////////////////////////////////

// DocOffsetIdx store document offset index version
type DocOffsetIdx interface {
	GetDoiVersion() uint32
	GetDocID() string
	GetDocVersion() uint64
	Close() error
}

// DoiVersionS is structure used to store document offset index version
type DoiVersionS struct {
	DoiVer uint32
}

// GetDoiVersion retrieves the document offset index version number
func (h *DoiVersionS) GetDoiVersion() uint32 {
	return h.DoiVer
}

// Deserialize deserializes the data into version number object
func (h *DoiVersionS) Deserialize(data []byte) (*DoiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeDoiVersion deserializes the data into version number object
func DeserializeDoiVersion(data []byte) (*DoiVersionS, error) {
	h := &DoiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                   Document Offset Index
//
//////////////////////////////////////////////////////////////////

// createDocOffsetIdx creates a new document offset index for writing
func createDocOffsetIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey, initOffset int64) (*DocOffsetIdxV1, error) {
	return createDocOffsetIdxV1(sdc, docID, docVer, key, initOffset)
}

// todo expose to user
// Create document offset index and writes output
func CreateAndSaveDocOffsetIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourceData interface{}) error {
	return CreateAndSaveDocOffsetIdxWithOffset(sdc, docID, docVer, key, sourceData, 0)
}

// todo expose to user
func CreateAndSaveDocOffsetIdxWithOffset(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourceData interface{}, initOffset int64) error {
	tokenizer, err := utils.OpenFileTokenizer(sourceData)
	if err != err {
		return err
	}

	doi, err := createDocOffsetIdx(sdc, docID, docVer, key, initOffset)
	if err != nil {
		return err
	}

	for token, pos, err := tokenizer.NextToken(); err != io.EOF; token, pos, err = tokenizer.NextToken() {
		addErr := doi.AddTermOffset(token, uint64(pos.Offset))
		if addErr != nil {
			return addErr
		}
	}

	return doi.Close()
}

// todo expose to user
// OpenDocOffsetIdx opens a document offset index for reading
func OpenDocOffsetIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey) (DocOffsetIdx, error) {
	return OpenDocOffsetIdxWithOffset(sdc, docID, docVer, key, 0)
}

// todo expose to user
// OpenDocOffsetIdx opens a document offset index for reading
func OpenDocOffsetIdxWithOffset(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey, initOffset int64) (DocOffsetIdx, error) {
	reader, err := openDocOffsetIdxReader(sdc, docID, docVer)
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

	version, err := DeserializeDoiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetDoiVersion() {
	case DOI_V1:
		// Parse plaintext header body
		plainHdrBody := &DoiPlainHdrBodyV1{}
		plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
		if err != nil {
			return nil, errors.New(err)
		}
		return openDocOffsetIdxV1(key, plainHdrBody, reader, initOffset+int64(parsed))
	default:
		return nil, errors.Errorf("Document offset index version %v is not supported",
			version.GetDoiVersion())
	}
}
