package searchidx

import (
	"fmt"
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
	validateSibSize(t, stib, err)
	err = stib.AddDocOffset(docID1, 1, 2)
	validateSibSize(t, stib, err)
	err = stib.AddDocOffsets(docID1, 1, []uint64{100, 1000, 10000})
	validateSibSize(t, stib, err)

	stib.DocRemove(docID1)
	validateSibSize(t, stib, nil)

	// Replace older version
	err = stib.AddDocOffsets(docID1, 1, []uint64{100, 1000, 10000})
	validateSibSize(t, stib, err)
	err = stib.AddDocOffsets(docID1, 2, []uint64{200, 2000, 20000})
	validateSibSize(t, stib, err)

	err = stib.AddDocOffsets(docID2, 1, []uint64{300, 3000, 30000})
	validateSibSize(t, stib, err)
	err = stib.AddDocOffsets(docID3, 1, []uint64{400, 4000, 40000})
	validateSibSize(t, stib, err)

	stib.DocRemove(docID2)
	validateSibSize(t, stib, nil)
	stib.DocRemove(docID3)
	validateSibSize(t, stib, nil)
	stib.DocRemove(docID1)
	validateSibSize(t, stib, nil)

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
			validateSibSize(t, stib, err)
		} else {
			//fmt.Println("full")
			validateSibSize(t, stib, nil)
			docIdx = rand.Intn(len(docIDs))
			stib.DocRemove(docIDs[docIdx])
			validateSibSize(t, stib, nil)
		}

		i += count
	}
}

func validateSibSize(t *testing.T, stib *SearchTermIdxBlkV1, err error) {
	assert.NilError(t, err)

	b, err := stib.Serialize()
	assert.NilError(t, err)
	//fmt.Println("len", len(b), "pred", stib.predictedJSONSize, "max", stib.maxDataSize)
	assert.Equal(t, uint64(len(b)), stib.predictedJSONSize)
	assert.Assert(t, stib.predictedJSONSize <= stib.maxDataSize)
}

func TestRemoveString(t *testing.T) {
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
