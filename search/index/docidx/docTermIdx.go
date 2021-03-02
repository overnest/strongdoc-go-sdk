package docidx

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/go-errors/errors"
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

// CreateDocTermIdx creates a new document term index for writing
func CreateDocTermIdx(docID string, docVer uint64, key *sscrypto.StrongSaltKey,
	source DocTermSourceV1, store interface{}, initOffset int64) (DocTermIdx, error) {

	return CreateDocTermIdxV1(docID, docVer, key, source, store, initOffset)
}

// OpenDocTermIdx opens a document term index for reading
func OpenDocTermIdx(key *sscrypto.StrongSaltKey, store interface{}, initOffset uint64, endOffset uint64) (DocTermIdx, error) {
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

	version, err := DeserializeDtiVersion(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetDtiVersion() {
	case DTI_V1:
		// Parse plaintext header body
		plainHdrBody := &DtiPlainHdrBodyV1{}
		plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
		if err != nil {
			return nil, errors.New(err)
		}
		return openDocTermIdxV1(key, plainHdrBody, reader, initOffset, endOffset, initOffset+uint64(parsed))
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
	case DTI_V1:
		dtiv1, ok := dti.(*DocTermIdxV1)
		if !ok {
			return nil, errors.Errorf("Document term index version is not %v",
				dti.GetDtiVersion())
		}
		blk, berr := dtiv1.ReadNextBlock()
		if berr != nil && berr != io.EOF {
			return nil, berr
		}
		err = berr
		if blk != nil && blk.totalTerms > 0 {
			terms = blk.Terms
		}
	default:
		return nil, errors.Errorf("Document term index version %v is not supported",
			dti.GetDtiVersion())
	}

	return
}
