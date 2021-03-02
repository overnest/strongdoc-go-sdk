package searchidx

import (
	"fmt"
	"os"
	"testing"
	"time"

	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
//
//	data source	==FileTokenizer==> tokenized data ==doi Generator==> offset index ==dti Generator==> term index ==> search index
//
//  step1: tokenize data using FileTokenizer
//	step2: generate Document Offset Index(doi) from tokenized data
//	step3: generate Document Term Index(dti) from tokenized data or doi
//  step4: generate Org/User Search Index(si) from  doi and dti(optional)
//
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

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
		fmt.Println("path=", path)

		err = os.MkdirAll(path, 0770)
		assert.NilError(t, err)
		time.Sleep(time.Millisecond * 100)
	}

	defer os.RemoveAll(GetSearchIdxPathPrefix())

	resultIDs, err := GetUpdateIdsHmacV1(owner, termHmac)
	assert.NilError(t, err)

	assert.DeepEqual(t, updateIDs, resultIDs)
}
