package utils

import (
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/single"
	"gotest.tools/assert"
	"testing"

	"github.com/blevesearch/bleve/v2/analysis"
)

func TestBleveTokenizer(t *testing.T) {
	input := []byte("Hello World")

	// single
	singleTokenizerOutput := analysis.TokenStream{
		{
			Start:    0,
			End:      11,
			Term:     []byte("Hello World"),
			Position: 1,
			Type:     analysis.AlphaNumeric,
		},
	}

	singleTokenizer := single.NewSingleTokenTokenizer()
	assert.DeepEqual(t, singleTokenizer.Tokenize(input), singleTokenizerOutput)

}
