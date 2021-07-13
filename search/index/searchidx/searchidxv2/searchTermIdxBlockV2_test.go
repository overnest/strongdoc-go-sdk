package searchidxv2

import (
	"github.com/overnest/strongsalt-common-go/blocks"
	"gotest.tools/assert"
	"testing"
)

func TestSearchTermIdxBlkV2(t *testing.T) {
	var maxSize uint64 = 1000 // 1000 bytes
	blk := CreateSearchTermIdxBlkV2(maxSize)
	assert.Check(t, blk.IsEmpty())

	blk.AddDocOffsets("term1", "doc1", 1, []uint64{1, 2, 3, 4, 5})
	expected, err := blocks.GetPredictedJSONSize(blk)
	assert.NilError(t, err)
	assert.Check(t, blk.predictedJSONSize == uint64(expected))

	blk.AddDocOffsets("term2", "doc2", 1, []uint64{1, 2, 10, 100})
	expected, err = blocks.GetPredictedJSONSize(blk)
	assert.NilError(t, err)
	assert.Check(t, blk.predictedJSONSize == uint64(expected))

	blk.AddDocOffsets("term3", "doc3", 1, []uint64{1, 10000})
	expected, err = blocks.GetPredictedJSONSize(blk)
	assert.NilError(t, err)
	assert.Check(t, blk.predictedJSONSize == uint64(expected))

	blk.AddDocOffsets("term3", "doc3", 200, []uint64{3, 500, 900})
	expected, err = blocks.GetPredictedJSONSize(blk)
	assert.NilError(t, err)
	assert.Check(t, blk.predictedJSONSize == uint64(expected))

}
