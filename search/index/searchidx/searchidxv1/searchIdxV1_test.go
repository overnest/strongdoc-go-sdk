package searchidxv1

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
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
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	// ================================ Generate updateID ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	termHmac, err := createTermHmac(term, termKey)
	assert.NilError(t, err)

	for i := 0; i < idCount; i++ {
		writer, updateID, err := common.OpenSearchTermIndexWriter(sdc, owner, termHmac)
		updateIDs[i] = updateID
		assert.NilError(t, err)
		err = writer.Close()
		assert.NilError(t, err)
	}

	time.Sleep(time.Second * 10)

	defer common.RemoveSearchIndex(sdc, owner, termHmac)

	resultIDs, err := GetUpdateIdsHmacV1(sdc, owner, termHmac)
	assert.NilError(t, err)

	assert.DeepEqual(t, updateIDs, resultIDs)
}

//  TODO only available for localTest
func TestSearchIdxWriterV1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := prevTest(t)
	numSources := 2
	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")

	// ================================ Generate Search Index ================================
	docKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	sources := make([]SearchTermIdxSourceV1, 0, numSources)
	docs, err := docidx.InitTestDocuments(numSources, false)
	assert.NilError(t, err)

	for _, doc := range docs {
		assert.NilError(t, doc.CreateDoiAndDti(sdc, docKey))
		defer doc.RemoveAllVersionsIndexes(sdc)
		doi, err := doc.OpenDoi(sdc, docKey)
		assert.NilError(t, err)
		dti, err := doc.OpenDti(sdc, docKey)
		assert.NilError(t, err)
		source, err := SearchTermIdxSourceCreateDoc(doi, dti)
		assert.NilError(t, err)
		sources = append(sources, source)
		defer doi.Close()
		defer dti.Close()
	}

	siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, sources)
	assert.NilError(t, err)

	_, err = siw.ProcessAllTerms(sdc)
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
	}

	assert.NilError(t, err)
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

func generateTermHmacAndRemoveSearchIndex(sdc client.StrongDocClient, owner common.SearchIdxOwner, term string, termKey *sscrypto.StrongSaltKey) error {
	hamcTerm, err := createTermHmac(term, termKey)
	if err != nil {
		return err
	}
	return common.RemoveSearchIndex(sdc, owner, hamcTerm)
}
