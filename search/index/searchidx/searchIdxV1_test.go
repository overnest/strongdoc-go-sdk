package searchidx

import (
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestSearchTermUpdateIDsV1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	idCount := 10
	updateIDs := make([]string, idCount)
	term := "myTerm"
	owner := CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	// ================================ Generate updateID ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	termHmac, err := createTermHmac(term, termKey)
	assert.NilError(t, err)

	for i := 0; i < idCount; i++ {
		writer, updateID, err := openSearchTermIndexWriter(sdc, owner, termHmac)
		updateIDs[i] = updateID
		assert.NilError(t, err)
		err = writer.Close()
		assert.NilError(t, err)
	}

	time.Sleep(time.Second * 10)

	defer removeSearchIndex(sdc, owner, termHmac)

	resultIDs, err := GetUpdateIdsHmacV1(sdc, owner, termHmac)
	assert.NilError(t, err)

	assert.DeepEqual(t, updateIDs, resultIDs)
}

func prevTest(t *testing.T) client.StrongDocClient {
	if utils.TestLocal {
		return nil
	}
	// register org and admin
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	// login
	user := users[0][0]
	err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
	assert.NilError(t, err)
	return sdc
}
