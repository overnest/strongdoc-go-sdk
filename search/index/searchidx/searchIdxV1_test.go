package searchidx

import (
	"os"
	"testing"
	"time"

	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

func TestSearchTermUpdateIDsV1(t *testing.T) {
	idCount := 10
	updateIDs := make([]string, idCount)
	term := "myTerm"
	owner := CreateSearchIdxOwner(SI_OWNER_USR, "owner1")

	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	termHmac, err := createTermHmac(term, termKey)
	assert.NilError(t, err)

	for i := 0; i < idCount; i++ {
		updateIDs[i] = newUpdateIDV1()
		path := GetSearchIdxPathV1(GetSearchIdxPathPrefix(), owner, termHmac, updateIDs[i])
		err = os.MkdirAll(path, 0770)
		assert.NilError(t, err)
		time.Sleep(time.Millisecond * 100)
	}

	defer os.RemoveAll(GetSearchIdxPathPrefix())

	resultIDs, err := GetUpdateIdsHmacV1(owner, termHmac)
	assert.NilError(t, err)

	assert.DeepEqual(t, updateIDs, resultIDs)
}
