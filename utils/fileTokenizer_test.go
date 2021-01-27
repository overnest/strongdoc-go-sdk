package utils

import (
	"fmt"
	"io"
	"regexp"
	"testing"

	"gotest.tools/assert"
)

func TestRegex(t *testing.T) {
	regexGoodMap := map[*regexp.Regexp][]string{
		mimeGzip: {
			"application/x-gzip",
			"application/gzip",
		},
		mimeText: {
			"text/plain",
		},
	}

	regexGoodBad := map[*regexp.Regexp][]string{
		mimeGzip: {
			"application/x-gzipp",
			"application/x-gzipp",
			"application/gzipabc",
			"applications/gzip",
		},
		mimeText: {
			"text/plainn",
			"ttext/plainn",
			"text//plain",
		},
	}

	for regex, list := range regexGoodMap {
		for _, mime := range list {
			assert.Assert(t, regex.MatchString(mime),
				"Text %v should match the regex %v", mime, regex.String())
		}
	}

	for regex, list := range regexGoodBad {
		for _, mime := range list {
			assert.Assert(t, !regex.MatchString(mime),
				"Text %v should not match the regex %v", mime, regex.String())
		}
	}
}

func TestFileTokenizer(t *testing.T) {
	parsedTokens := int(100)
	testFileName := "./testDocuments/enwik8.txt.gz"
	//tempFileName := "/tmp/fileTokenTemp"

	path, err := FetchFileLoc(testFileName)
	assert.NilError(t, err)

	tokenizer, err := OpenFileTokenizer(path)
	assert.NilError(t, err)

	tokens1 := make([]string, 0, parsedTokens)

	for i := 0; i < parsedTokens; i++ {
		token, pos, err := tokenizer.NextToken()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		tokens1 = append(tokens1, token)
		fmt.Println("token", token, pos)
	}

	err = tokenizer.Reset()
	assert.NilError(t, err)

	tokens2 := make([]string, 0, parsedTokens)
	for i := 0; i < parsedTokens; i++ {
		token, pos, err := tokenizer.NextToken()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		tokens2 = append(tokens2, token)
		fmt.Println("token", token, pos)
	}

	assert.DeepEqual(t, tokens1, tokens2)

	err = tokenizer.Close()
	assert.NilError(t, err)
}
