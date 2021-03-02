package searchidx

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"math/rand"
	"os"
	"testing"

	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

func TestSearchTermIdxV1(t *testing.T) {
	// ================================ Prev Test ================================
	testClient := prevTest(t)

	//var owners = []SearchIdxOwner{
	//	CreateSearchIdxOwner(SI_OWNER_USR, "owner1"),
	//	CreateSearchIdxOwner(SI_OWNER_USR, "owner2"),
	//	CreateSearchIdxOwner(SI_OWNER_USR, "owner3")}

	owner1 := CreateSearchIdxOwner(SI_OWNER_USR, "owner1")
	//var terms = []string{"term1", "term2", "term3"}

	term1 := "term1"

	docIDs := 20
	maxOffsets := 30

	// ================================ Generate search term index ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	writer, err := testUtils.OpenSearchIdxOffsetWriter(testLocal, testClient, owner1.GetOwnerType2(), owner1.GetOwnerID(), term1)
	sti, err := CreateSearchTermIdxV1_2(owner1, term1, termKey, indexKey, nil, nil, writer)
	assert.NilError(t, err)
	defer os.RemoveAll(GetSearchIdxPathPrefix())

	var block *SearchTermIdxBlkV1 = CreateSearchTermIdxBlkV1(sti.GetMaxBlockDataSize())
	for i := uint64(0); i < 10000000; {
		docID := fmt.Sprintf("DocID_%v", rand.Intn(docIDs))
		offsetCount := uint64(rand.Intn(maxOffsets-1) + 1)
		offsets := make([]uint64, offsetCount)
		for j := uint64(0); j < offsetCount; j++ {
			offsets[j] = i + j
		}
		i += offsetCount

		err = block.AddDocOffsets(docID, 1, offsets)
		if err != nil {
			err = sti.WriteBlock(block)
			assert.NilError(t, err)
			block = CreateSearchTermIdxBlkV1(sti.GetMaxBlockDataSize())

			err = block.AddDocOffsets(docID, 1, offsets)
			assert.NilError(t, err)
		}
	}

	if !block.IsEmpty() {
		err = sti.WriteBlock(block)
		assert.NilError(t, err)
	}

	err = sti.Close()
	assert.NilError(t, err)
}
