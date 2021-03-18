package docidx

import (
	"encoding/json"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"io"
	"strings"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/docidxv1"

	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

//////////////////////////////////////////////////////////////////
//
//                Document Term Index Version
//
//////////////////////////////////////////////////////////////////

// DocTermIdx store document term index version
type DocTermIdx interface {
	GetDtiVersion() uint32
	GetDocID() string
	GetDocVersion() uint64
	Close() error
}

// DtiVersionS is structure used to store document term index version
type DtiVersionS struct {
	DtiVer uint32
}

// GetDtiVersion retrieves the document term index version number
func (h *DtiVersionS) GetDtiVersion() uint32 {
	return h.DtiVer
}

// Deserialize deserializes the data into version number object
func (h *DtiVersionS) Deserialize(data []byte) (*DtiVersionS, error) {
	err := json.Unmarshal(data, h)
	if err != nil {
		return nil, errors.New(err)
	}
	return h, nil
}

// DeserializeDtiVersion deserializes the data into version number object
func DeserializeDtiVersion(data []byte) (*DtiVersionS, error) {
	h := &DtiVersionS{}
	return h.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                   Document Term Index
//
//////////////////////////////////////////////////////////////////

// create term index from source data
func CreateAndSaveDocTermIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourcefile utils.Storage) error {
	return CreateAndSaveDocTermIdxWithOffset(sdc, docID, docVer, key, sourcefile, 0)
}

func CreateAndSaveDocTermIdxWithOffset(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourcefile utils.Storage, initOffset int64) error {
	return createAndSaveDocTermIdxV1(sdc, docID, docVer, key, sourcefile, initOffset)
}

func createAndSaveDocTermIdxV1(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	sourcefile utils.Storage, initOffset int64) error {
	source, err := docidxv1.OpenDocTermSourceTextFileV1(sourcefile)
	if err != nil {
		return err
	}
	return doCreateAndSaveTermIdxV1(sdc, docID, docVer, key, initOffset, source)
}

// create term index from doi
func CreateAndSaveDocTermIdxFromDOI(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey) error {
	return CreateAndSaveDocTermIdxFromDOIWithOffset(sdc, docID, docVer, key, 0)
}

func CreateAndSaveDocTermIdxFromDOIWithOffset(sdc client.StrongDocClient, docID string, docVer uint64,
	key *sscrypto.StrongSaltKey, initOffset int64) error {
	return createAndSaveDocTermIdxV1FromDOI(sdc, docID, docVer, key, initOffset)
}

func createAndSaveDocTermIdxV1FromDOI(sdc client.StrongDocClient, docID string, docVer uint64,
	key *sscrypto.StrongSaltKey, initOffset int64) error {
	doi, err := OpenDocOffsetIdxWithOffset(sdc, docID, docVer, key, initOffset)
	if err != nil {
		return err
	}
	defer doi.Close()

	source, err := docidxv1.OpenDocTermSourceDocOffsetV1(doi)
	if err != nil {
		return err
	}

	return doCreateAndSaveTermIdxV1(sdc, docID, docVer, key, initOffset, source)
}

func doCreateAndSaveTermIdxV1(sdc client.StrongDocClient, docID string, docVer uint64,
	key *sscrypto.StrongSaltKey, initOffset int64, source docidxv1.DocTermSourceV1) error {
	dtiWriter, err := common.OpenDocTermIdxWriter(sdc, docID, docVer)
	if err != nil {
		return err
	}
	// Create a new document term index for writing
	dti, err := docidxv1.CreateDocTermIdxV1(docID, docVer, key, source, dtiWriter, initOffset)
	if err != nil {
		return err
	}
	defer dti.Close()

	for err == nil {
		_, err = dti.WriteNextBlock()
		if err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

func OpenDocTermIdx(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey) (DocTermIdx, error) {
	size, err := common.GetDocTermIndexSize(sdc, docID, docVer)
	if err != nil {
		return nil, err
	}
	return OpenDocTermIdxWithOffset(sdc, docID, docVer, key, 0, size)
}

// OpenDocTermIdxWithOffset opens a document term index for reading from initOffset to endOffset
func OpenDocTermIdxWithOffset(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey, initOffset uint64, endOffset uint64) (DocTermIdx, error) {
	reader, err := common.OpenDocTermIdxReader(sdc, docID, docVer)
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

	version, err := DeserializeDtiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetDtiVersion() {
	case common.DTI_V1:
		// Parse plaintext header body
		plainHdrBody, err := docidxv1.DtiPlainHdrBodyV1Deserialize(plainHdrBodyData)
		if err != nil {
			return nil, errors.New(err)
		}
		return docidxv1.OpenDocTermIdxPrivV1(key, plainHdrBody, reader, initOffset, endOffset, initOffset+uint64(parsed))
	default:
		return nil, errors.Errorf("Document term index version %v is not supported",
			version.GetDtiVersion())
	}
}

// DiffDocTermIdx diffs the terms indexes
func DiffDocTermIdx(dtiOld DocTermIdx, dtiNew DocTermIdx) (added []string, deleted []string, err error) {
	var oldTerm *string = nil
	var newTerm *string = nil

	oldTermList, err := getNextTermList(dtiOld)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	if oldTermList != nil {
		oldTerm = &oldTermList[0]
	}

	newTermList, err := getNextTermList(dtiNew)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	if newTermList != nil {
		newTerm = &newTermList[0]
	}

	added = make([]string, 0, 1000)
	deleted = make([]string, 0, 1000)
	oldTermIdx := 1
	newTermIdx := 1

	for oldTerm != nil || newTerm != nil {
		updateOld := false
		updateNew := false

		if oldTerm == nil {
			added = append(added, *newTerm)
			updateNew = true
		} else if newTerm == nil {
			deleted = append(deleted, *oldTerm)
			updateOld = true
		} else {
			comp := strings.Compare(*oldTerm, *newTerm)
			if comp == 0 {
				updateOld = true
				updateNew = true
			} else if comp < 0 {
				deleted = append(deleted, *oldTerm)
				updateOld = true
			} else {
				added = append(added, *newTerm)
				updateNew = true
			}
		}

		if updateOld && oldTermList != nil {
			oldTerm = nil
			if oldTermIdx >= len(oldTermList) {
				oldTermList, err = getNextTermList(dtiOld)
				if err != nil && err != io.EOF {
					return nil, nil, err
				}
				oldTermIdx = 0
			}

			if oldTermList != nil {
				oldTerm = &oldTermList[oldTermIdx]
				oldTermIdx++
			}
		}

		if updateNew && newTermList != nil {
			newTerm = nil
			if newTermIdx >= len(newTermList) {
				newTermList, err = getNextTermList(dtiNew)
				if err != nil && err != io.EOF {
					return nil, nil, err
				}
				newTermIdx = 0
			}

			if newTermList != nil {
				newTerm = &newTermList[newTermIdx]
				newTermIdx++
			}
		}
	} // for oldterm != nil && newterm != nil

	if err == io.EOF {
		err = nil
	}
	return
}

func getNextTermList(dti DocTermIdx) (terms []string, err error) {
	terms = nil
	err = nil

	if dti == nil {
		err = io.EOF
		return
	}

	switch dti.GetDtiVersion() {
	case common.DTI_V1:
		dtiv1, ok := dti.(*docidxv1.DocTermIdxV1)
		if !ok {
			return nil, errors.Errorf("Document term index version is not %v",
				dti.GetDtiVersion())
		}
		blk, berr := dtiv1.ReadNextBlock()
		if berr != nil && berr != io.EOF {
			return nil, berr
		}
		err = berr
		if blk != nil && blk.GetTotalTerms() > 0 {
			terms = blk.Terms
		}
	default:
		return nil, errors.Errorf("Document term index version %v is not supported",
			dti.GetDtiVersion())
	}

	return
}

// GetAllTermList gets all the terms in the DTI
func GetAllTermList(dti DocTermIdx) (terms []string, err error) {
	terms = make([]string, 0)
	if dti == nil {
		return terms, nil
	}

	switch dti.GetDtiVersion() {
	case common.DTI_V1:
		dtiv1, ok := dti.(*docidxv1.DocTermIdxV1)
		if !ok {
			return nil, errors.Errorf("Document term index version is not %v",
				dti.GetDtiVersion())
		}

		err = dtiv1.Reset()
		if err != nil {
			return
		}

		for err == nil {
			var blk *docidxv1.DocTermIdxBlkV1
			blk, err = dtiv1.ReadNextBlock()
			if err != nil && err != io.EOF {
				return nil, err
			}
			if blk != nil && blk.GetTotalTerms() > 0 {
				if len(terms) == 0 {
					terms = blk.Terms
				} else {
					terms = append(terms, blk.Terms...)
				}
			}
		}

		return terms, nil
	default:
		return nil, errors.Errorf("Document term index version %v is not supported",
			dti.GetDtiVersion())
	}

	return
}
