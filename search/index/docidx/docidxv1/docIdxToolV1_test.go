package docidxv1

import (
	"io"
	"math/rand"
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

func TestCreateModifiedDoc(t *testing.T) {
	documents := 30
	versions := 3

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)

	docs, err := InitTestDocuments(documents, false)
	assert.NilError(t, err)
	defer CleanTestDocumentIndexes()

	docVers := make([][]*TestDocumentIdxV1, len(docs))
	for i, doc := range docs {
		docVers[i] = make([]*TestDocumentIdxV1, versions+1)
		docVers[i][0] = doc
	}

	for v := 0; v < versions; v++ {
		for _, oldDocs := range docVers {
			oldDoc := oldDocs[v]
			newDoc := testCreateModifiedDoc(t, oldDoc, key, v == 0)
			oldDocs[v+1] = newDoc
		}
	}
}

func testCreateModifiedDoc(t *testing.T, oldDoc *TestDocumentIdxV1, key *sscrypto.StrongSaltKey, createOldIdx bool) *TestDocumentIdxV1 {
	addedTerms, deletedTerms := rand.Intn(99)+1, rand.Intn(99)+1

	newDoc, err := oldDoc.CreateModifiedDoc(addedTerms, deletedTerms)
	assert.NilError(t, err)
	defer oldDoc.Close()
	defer newDoc.Close()

	assert.Equal(t, len(newDoc.AddedTerms), addedTerms)
	assert.Equal(t, len(newDoc.DeletedTerms), deletedTerms)

	if createOldIdx {
		err = oldDoc.CreateDoi(key)
		assert.NilError(t, err)
		err = oldDoc.CreateDti(key)
		assert.NilError(t, err)
	}
	err = newDoc.CreateDoi(key)
	assert.NilError(t, err)
	err = newDoc.CreateDti(key)
	assert.NilError(t, err)

	oldDti, err := oldDoc.OpenDti(key)
	assert.NilError(t, err)
	newDti, err := newDoc.OpenDti(key)
	assert.NilError(t, err)

	oldTermList, oldTermMap, err := oldDti.ReadAllTerms()
	assert.NilError(t, err)
	newTermList, newTermMap, err := newDti.ReadAllTerms()
	assert.NilError(t, err)

	dtiDeleted := make([]string, 0, len(oldTermList))
	for _, term := range oldTermList {
		if !newTermMap[term] {
			dtiDeleted = append(dtiDeleted, term)
		}
	}

	dtiAdded := make([]string, 0, len(oldTermList))
	for _, term := range newTermList {
		if !oldTermMap[term] {
			dtiAdded = append(dtiAdded, term)
		}
	}

	docAdded := make([]string, 0, len(newDoc.AddedTerms))
	for term := range newDoc.AddedTerms {
		docAdded = append(docAdded, term)
	}
	sort.Strings(docAdded)

	docDeleted := make([]string, 0, len(newDoc.DeletedTerms))
	for term := range newDoc.DeletedTerms {
		docDeleted = append(docDeleted, term)
	}
	sort.Strings(docDeleted)

	assert.DeepEqual(t, dtiAdded, docAdded)
	assert.DeepEqual(t, dtiDeleted, docDeleted)

	return newDoc
}
