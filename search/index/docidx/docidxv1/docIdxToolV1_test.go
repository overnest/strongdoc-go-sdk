package docidxv1

import (
	"io"
	"sort"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
)

func TestTools(t *testing.T) {
	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	indexes, err := InitTestDocuments(10, false)
	assert.NilError(t, err)

	// Create the indexes
	for _, idx := range indexes {
		assert.NilError(t, idx.CreateDoi(key))
		assert.NilError(t, idx.CreateDti(key))
	}
	defer CleanTestDocumentIndexes()

	// Validate the indexes
	for _, idx := range indexes {
		// Open the DOI
		doi, err := idx.OpenDoi(key)
		assert.NilError(t, err)
		defer idx.CloseDoi()

		termloc := make(map[string][]uint64)
		for err == nil {
			var blk *DocOffsetIdxBlkV1
			blk, err = doi.ReadNextBlock()
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}
			if blk != nil {
				for term, locs := range blk.TermLoc {
					termloc[term] = append(termloc[term], locs...)
				}
			}
		}

		i := uint64(0)
		tokenizer, err := utils.OpenFileTokenizer(idx.docFilePath)
		assert.NilError(t, err)
		defer tokenizer.Close()

		// Validate the DOI
		for token, _, err := tokenizer.NextToken(); err != io.EOF; token, _, err = tokenizer.NextToken() {
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}

			if token != "" {
				locs, exist := termloc[token]
				assert.Assert(t, exist)
				assert.Assert(t, len(locs) > 0)
				assert.Equal(t, i, locs[0])
				termloc[token] = locs[1:]
				i++
			}
		}

		// Open the DTI
		dti, err := idx.OpenDti(key)
		assert.NilError(t, err)
		defer idx.CloseDti()

		terms := make([]string, 0, len(termloc))
		for term := range termloc {
			terms = append(terms, term)
			delete(termloc, term)
		}
		sort.Strings(terms)

		// Validate the DTI
		for err == nil {
			var blk *DocTermIdxBlkV1
			blk, err = dti.ReadNextBlock()
			if err != nil {
				assert.Equal(t, err, io.EOF)
			}
			if blk != nil {
				for _, term := range blk.Terms {
					assert.Assert(t, len(terms) > 0)
					assert.Equal(t, term, terms[0])
					terms = terms[1:]
				}
			}
		}

		assert.Equal(t, len(terms), 0)
	} // for _, idx := range indexes
}
