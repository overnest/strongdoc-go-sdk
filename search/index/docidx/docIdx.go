package docidx

import (
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
)

// create and save document offset index and term index
func CreateAndSaveDocIndexes(sdc client.StrongDocClient, docID string, docVer uint64,
	key *sscrypto.StrongSaltKey, sourceData utils.Source) error {

	err := CreateAndSaveDocOffsetIdx(sdc, docID, docVer, key, sourceData)
	if err != nil {
		return err
	}

	return CreateAndSaveDocTermIdxFromDOI(sdc, docID, docVer, key)
}
