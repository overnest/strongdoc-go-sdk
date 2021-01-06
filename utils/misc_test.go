package utils

import (
	"testing"

	"gotest.tools/assert"
)

func TestBinarySearchU64(t *testing.T) {
	a1 := []uint64{}
	a2 := []uint64{1}
	a3 := []uint64{1, 2}
	a4 := []uint64{1, 2, 3}
	a5 := []uint64{1, 2, 3, 4, 5, 6}
	a6 := []uint64{1, 2, 3, 4, 5, 6, 7}

	// Test positive cases
	testBinarySearchU64Positive(t, a2)
	testBinarySearchU64Positive(t, a3)
	testBinarySearchU64Positive(t, a4)
	testBinarySearchU64Positive(t, a5)
	testBinarySearchU64Positive(t, a6)

	// Test negative cases
	assert.Equal(t, BinarySearchU64(a1, 1), -1)
	assert.Equal(t, BinarySearchU64(a2, 0), -1)
	assert.Equal(t, BinarySearchU64(a2, 2), -1)
	assert.Equal(t, BinarySearchU64(a3, 0), -1)
	assert.Equal(t, BinarySearchU64(a3, 3), -1)
	assert.Equal(t, BinarySearchU64(a3, 4), -1)
	assert.Equal(t, BinarySearchU64(a4, 0), -1)
	assert.Equal(t, BinarySearchU64(a4, 4), -1)
	assert.Equal(t, BinarySearchU64(a4, 5), -1)
	assert.Equal(t, BinarySearchU64(a5, 0), -1)
	assert.Equal(t, BinarySearchU64(a5, 7), -1)
	assert.Equal(t, BinarySearchU64(a5, 8), -1)
	assert.Equal(t, BinarySearchU64(a6, 0), -1)
	assert.Equal(t, BinarySearchU64(a6, 8), -1)
	assert.Equal(t, BinarySearchU64(a6, 9), -1)
}

func testBinarySearchU64Positive(t *testing.T, list []uint64) {
	for i := uint64(1); i <= uint64(len(list)); i++ {
		assert.Equal(t, BinarySearchU64(list, i), int(i-1))
	}
}
