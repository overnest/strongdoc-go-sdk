package searchidxv1

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/overnest/strongsalt-common-go/blocks"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"io"
	"math/rand"
	"testing"

	"gotest.tools/assert"
)

func TestSearchTermIdxBlockV1(t *testing.T) {
	var err error
	docID1, docID2, docID3 := "DOCID1", "DOCID2", "DOCID3"
	docIDs := []string{docID1, docID2, docID3}

	stib := CreateSearchTermIdxBlkV1(1000)

	//
	// Test predicted size
	//
	err = stib.AddDocOffset(docID1, 1, 1)
	validateStibSize(t, stib, err)
	err = stib.AddDocOffset(docID1, 1, 2)
	validateStibSize(t, stib, err)
	err = stib.AddDocOffsets(docID1, 1, []uint64{100, 1000, 10000})
	validateStibSize(t, stib, err)

	stib.DocRemove(docID1)
	validateStibSize(t, stib, nil)

	// Replace older version
	err = stib.AddDocOffsets(docID1, 1, []uint64{100, 1000, 10000})
	validateStibSize(t, stib, err)
	err = stib.AddDocOffsets(docID1, 2, []uint64{200, 2000, 20000})
	validateStibSize(t, stib, err)

	err = stib.AddDocOffsets(docID2, 1, []uint64{300, 3000, 30000})
	validateStibSize(t, stib, err)
	err = stib.AddDocOffsets(docID3, 1, []uint64{400, 4000, 40000})
	validateStibSize(t, stib, err)

	stib.DocRemove(docID2)
	validateStibSize(t, stib, nil)
	stib.DocRemove(docID3)
	validateStibSize(t, stib, nil)
	stib.DocRemove(docID1)
	validateStibSize(t, stib, nil)

	//
	// Test maximum size
	//
	for i := 0; i < 10000; {
		count := rand.Intn(9) + 1
		offsets := make([]uint64, count)
		for j := 0; j < count; j++ {
			offsets[j] = uint64(i + j)
		}

		docIdx := rand.Intn(len(docIDs))
		err = stib.AddDocOffsets(docIDs[docIdx], 1, offsets)
		if err == nil {
			validateStibSize(t, stib, err)
		} else {
			//fmt.Println("full")
			validateStibSize(t, stib, nil)
			docIdx = rand.Intn(len(docIDs))
			stib.DocRemove(docIDs[docIdx])
			validateStibSize(t, stib, nil)
		}

		i += count
	}
}

func validateStibSize(t *testing.T, stib *SearchTermIdxBlkV1, err error) {
	assert.NilError(t, err)

	predictSize, err := blocks.GetPredictedJSONSize(stib)
	assert.NilError(t, err)
	//fmt.Println("predictSize", predictSize, "pred", stib.predictedJSONSize, "max", stib.maxDataSize)
	assert.Equal(t, uint64(predictSize), stib.predictedJSONSize)
	assert.Assert(t, stib.predictedJSONSize <= stib.maxDataSize)
}

func TestSearchTermIdxBatchRemoveString(t *testing.T) {
	size := int(1000)
	arr := make([]string, size)
	smap := make(map[string]bool)
	for i := 0; i < size; i++ {
		arr[i] = fmt.Sprintf("%v", i)
		smap[arr[i]] = true
	}

	for len(arr) > 0 {
		for i := 0; i < len(arr); i++ {
			remove := rand.Intn(2)
			if remove == 1 {
				// fmt.Println("remove", i, arr[i], arr)

				delete(smap, arr[i])
				arr = removeString(arr, i)
				i--

				// Validate array
				assert.Equal(t, len(smap), len(arr))
				for _, str := range arr {
					_, ok := smap[str]
					assert.Assert(t, ok)
				}
			}
		}
	}
}

func TestSearchTermIdxSimpleV1(t *testing.T) {
	// ================================ Prev Test ================================
	sdc := common.PrevTest(t)

	owner := common.CreateSearchIdxOwner(utils.OwnerUser, "owner1")
	term := "term1"
	maxDocID := 20
	maxOffsetCount := 30
	defer common.RemoveSearchIndex(sdc, owner)

	// ================================ Generate doc search index (termIdx) ================================
	termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
	assert.NilError(t, err)
	indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	//
	// Create New STI and write a lot of blocks with random info
	//
	sti, writtenBlocks := createSearchTermIdxSimpleV1(t, sdc, owner, term,
		termKey, indexKey, maxDocID, maxOffsetCount)

	//
	// Open STI and make sure the blocks match
	//
	updateIDs, err := GetUpdateIdsV1(sdc, owner, term, termKey)
	assert.NilError(t, err)

	sti, err = OpenSearchTermIdxV1(sdc, owner, term, termKey, indexKey, updateIDs[0])
	assert.NilError(t, err)

	var block *SearchTermIdxBlkV1
	for i := 0; i < 2; i++ {
		var blocks int
		for blocks = 0; err != io.EOF; blocks++ {
			block, err = sti.ReadNextBlock()
			if err != nil {
				assert.Assert(t, err == io.EOF)
			}
			if block == nil {
				break
			}

			assert.DeepEqual(t, writtenBlocks[blocks].DocVerOffset, block.DocVerOffset)
		}

		assert.Equal(t, len(writtenBlocks), blocks)
		err = sti.Reset()
		assert.NilError(t, err)
	}

	err = sti.Close()
	assert.NilError(t, err)
}

func createSearchTermIdxSimpleV1(t *testing.T,
	sdc client.StrongDocClient,
	owner common.SearchIdxOwner, term string,
	termKey, indexKey *sscrypto.StrongSaltKey,
	maxDocID, maxOffsetCount int) (*SearchTermIdxV1, []*SearchTermIdxBlkV1) {

	//
	// Create New STI and write a lot of blocks with random info
	//
	writtenBlocks := make([]*SearchTermIdxBlkV1, 0, 100)
	sti, err := CreateSearchTermIdxV1(sdc, owner, term, termKey, indexKey, nil, nil)
	assert.NilError(t, err)

	var block *SearchTermIdxBlkV1 = CreateSearchTermIdxBlkV1(sti.GetMaxBlockDataSize())
	for i := uint64(0); i < 10000000; {
		docID := fmt.Sprintf("DocID_%v", rand.Intn(maxDocID))
		offsetCount := uint64(rand.Intn(maxOffsetCount-1) + 1)
		offsets := make([]uint64, offsetCount)
		for j := uint64(0); j < offsetCount; j++ {
			offsets[j] = i + j
		}
		i += offsetCount

		err = block.AddDocOffsets(docID, 1, offsets)
		if err != nil {
			err = sti.WriteNextBlock(block)
			assert.NilError(t, err)
			writtenBlocks = append(writtenBlocks, block)
			block = CreateSearchTermIdxBlkV1(sti.GetMaxBlockDataSize())

			err = block.AddDocOffsets(docID, 1, offsets)
			assert.NilError(t, err)
		}
	}

	if !block.IsEmpty() {
		err = sti.WriteNextBlock(block)
		assert.NilError(t, err)
		writtenBlocks = append(writtenBlocks, block)
	}

	err = sti.Close()
	assert.NilError(t, err)

	return sti, writtenBlocks
}
