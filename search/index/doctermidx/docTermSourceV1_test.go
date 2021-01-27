package doctermidx

import (
	"io"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/utils"
	"gotest.tools/assert"
)

func TestDocTermSourceTextFileV1(t *testing.T) {
	sourceFileName := "./testDocuments/enwik8.txt.gz"
	sourceFilePath, err := utils.FetchFileLoc(sourceFileName)
	assert.NilError(t, err)

	dts, err := OpenDocTermSourceTextFileV1(sourceFilePath)
	assert.NilError(t, err)

	terms := make([]string, 0, 1000000)

	for true {
		term, _, err := dts.GetNextTerm()
		if term != "" {
			terms = append(terms, term)
		}

		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
	}

	err = dts.Reset()
	assert.NilError(t, err)

	i := 0
	for true {
		term, _, err := dts.GetNextTerm()
		if term != "" {
			assert.Equal(t, terms[i], term)
			i++
		}

		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
	}

}
