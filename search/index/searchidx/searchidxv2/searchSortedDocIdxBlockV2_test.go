package searchidxv2

import (
	"testing"
)

func TestSearchSortedDocIdxBlkV2(t *testing.T) {
	//blk := CreateSearchSortDocIdxBlkV2("term1", "", 1000)
	//assert.Check(t, !blk.IsFull())
	//
	//// add docVer
	//blk.AddTermDocVer("doc1", 1000)
	//blk.formatToBlockData()
	//expected, err := blocks.GetPredictedJSONSize(blk)
	//
	//assert.NilError(t, err)
	//assert.Check(t, blk.predictedJSONSize == uint64(expected))
	//
	//// add outdated version, ignore
	//before := blk.predictedJSONSize
	//blk.AddTermDocVer("doc1", 2)
	//blk.formatToBlockData()
	//expected, err = blocks.GetPredictedJSONSize(blk)
	//assert.Check(t, blk.predictedJSONSize == uint64(expected))
	//assert.Check(t, before == blk.predictedJSONSize)
}
