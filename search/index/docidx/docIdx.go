package docidx

import (
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

// create and save document offset index and term index
func CreateAndSaveDocIndexes(sdc client.StrongDocClient, docID string, docVer uint64, key *sscrypto.StrongSaltKey, sourceData utils.Storage) error {
	err := CreateAndSaveDocOffsetIdx(sdc, docID, docVer, key, sourceData)
	if err != nil {
		return err
	}

	_, err = sourceData.Seek(0, utils.SeekSet)
	if err != nil {
		return err
	}
	return CreateAndSaveDocTermIdx(sdc, docID, docVer, key, sourceData)
}
