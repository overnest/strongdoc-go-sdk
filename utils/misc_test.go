package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

func TestGrep(t *testing.T) {
	booksdir := "testDocuments/books"
	grepTerms := [][]string{
		{"almost", "no", "restrictions", "whatsoever"},
		{"almost", "no", "restrictions", "whatsoever", "doesnotexist"},
	}

	dirPath, err := FetchFileLoc(booksdir)
	assert.NilError(t, err)

	for _, terms := range grepTerms {
		docMap, err := Grep(booksdir, terms, false)
		assert.NilError(t, err)

		// Use shell command to verify
		cmd := fmt.Sprintf("zgrep -b -R \"%v\" %v | cut -d':' -f1-2", strings.Join(terms, " "), dirPath)
		out, err := exec.Command("bash", "-c", cmd).Output()
		assert.NilError(t, err)

		grepDocMap := make(map[string][]uint64)
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			docOffset := strings.Split(line, ":")
			if len(docOffset) != 2 {
				continue
			}

			offset, err := strconv.ParseUint(docOffset[1], 10, 64)
			assert.NilError(t, err)
			grepDocMap[docOffset[0]] = append(grepDocMap[docOffset[0]], offset)
		}

		assert.DeepEqual(t, docMap, grepDocMap)
	}
}
