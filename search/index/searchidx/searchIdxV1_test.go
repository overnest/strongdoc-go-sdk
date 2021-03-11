package searchidx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-errors/errors"
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

func TestSearchIdxWriterV1(t *testing.T) {
	numSources := 10
	owner := CreateSearchIdxOwner(SI_OWNER_USR, "owner1")

	docKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	sources := make([]SearchTermIdxSourceV1, 0, numSources)
	docs, err := InitDocuments(numSources, false)
	assert.NilError(t, err)
	defer CleanDocumentIndexes()

	for _, doc := range docs {
		assert.NilError(t, doc.CreateDoi(docKey))
		assert.NilError(t, doc.CreateDti(docKey))
		doi, err := doc.OpenDoi(docKey)
		assert.NilError(t, err)
		defer doc.CloseDoi()
		dti, err := doc.OpenDti(docKey)
		assert.NilError(t, err)
		defer doc.CloseDti()

		source, err := SearchTermIdxSourceCreateDoc(doi, dti)
		assert.NilError(t, err)
		sources = append(sources, source)
	}

	siw, err := CreateSearchIdxWriterV1(owner, termKey, indexKey, sources)
	assert.NilError(t, err)

	_, err = siw.ProcessAllTerms()
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
	}
	assert.NilError(t, err)

}
