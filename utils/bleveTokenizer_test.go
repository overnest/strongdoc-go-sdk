package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
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

var sampleTests = map[string](map[string][]string){
	"": {
		SINGLE.name:     []string{""},
		LETTER.name:     nil,
		WHITESPACE.name: nil,
		REGEX.name:      nil,
		UNICODE.name:    nil,
		WEB.name:        nil,
	},
	"123 abc 4c5d": {
		SINGLE.name:     []string{"123 abc 4c5d"},
		LETTER.name:     []string{"abc", "c", "d"},
		WHITESPACE.name: []string{"123", "abc", "4c5d"},
		REGEX.name:      []string{"123", "abc", "4c5d"},
		UNICODE.name:    []string{"123", "abc", "4c5d"},
		WEB.name:        []string{"123", "abc", "4c5d"},
	},
	" Hello World.": {
		SINGLE.name:     []string{" hello world."},
		LETTER.name:     []string{"hello", "world"},
		WHITESPACE.name: []string{"hello", "world."},
		REGEX.name:      []string{"hello", "world"},
		UNICODE.name:    []string{"hello", "world"},
		WEB.name:        []string{"hello", "world"},
	},
	"Dominique@mcdiabetes.com": {
		SINGLE.name:     []string{"dominique@mcdiabetes.com"},
		LETTER.name:     []string{"dominique", "mcdiabetes", "com"},
		WHITESPACE.name: []string{"dominique@mcdiabetes.com"},
		REGEX.name:      []string{"dominique", "mcdiabetes", "com"},
		UNICODE.name:    []string{"dominique", "mcdiabetes.com"},
		WEB.name:        []string{"d", "ominique@mcdiabetes.com"},
	},
	"That http://blevesearch.com": {
		SINGLE.name:     []string{"that http://blevesearch.com"},
		LETTER.name:     []string{"that", "http", "blevesearch", "com"},
		WHITESPACE.name: []string{"that", "http://blevesearch.com"},
		REGEX.name:      []string{"that", "http", "blevesearch", "com"},
		UNICODE.name:    []string{"that", "http", "blevesearch.com"},
		WEB.name:        []string{"that", "http://blevesearch.com"},
	},
	"Hello info@blevesearch.com": {
		SINGLE.name:     []string{"hello info@blevesearch.com"},
		LETTER.name:     []string{"hello", "info", "blevesearch", "com"},
		WHITESPACE.name: []string{"hello", "info@blevesearch.com"},
		REGEX.name:      []string{"hello", "info", "blevesearch", "com"},
		UNICODE.name:    []string{"hello", "info", "blevesearch.com"},
		WEB.name:        []string{"hello", "info@blevesearch.com"},
	},
}

func TestBleveAnalyzer(t *testing.T) {
	// Single Token: the entire input bytes as a single token
	testBleveAnalyzer(t, SINGLE, LOWERCASE)

	// Letter Token: simply identifies tokens as sequences of Unicode runes that are part of the Letter category
	testBleveAnalyzer(t, LETTER, LOWERCASE)

	// Whitespace: identifies tokens as sequences of Unicode runes that are NOT part of the Space category
	testBleveAnalyzer(t, WHITESPACE, LOWERCASE)

	// Regular Expression: tokenize input using a configurable regular expression
	testBleveAnalyzer(t, REGEX, LOWERCASE)

	// Unicode Tokenizer: uses the segment library to perform Unicode Text Segmentation on word boundaries
	// follow: http://www.unicode.org/reports/tr29/#Word_Boundary_Rules
	testBleveAnalyzer(t, UNICODE, LOWERCASE)

	// Web tokenizer: better support for email, url and web content (regex implementation)
	testBleveAnalyzer(t, WEB, LOWERCASE)

	// ICU tokenizer: uses the ICU library to tokenize the input
	// better support for some Asian languages by using a dictionary-based approach to identify words

}

func testBleveAnalyzer(t *testing.T, tokenizerType Tokenizer_Type, filter_type ...Token_Filter_Type) {
	analyzer, err := openBleveAnalyzer(tokenizerType, filter_type...)
	assert.NilError(t, err)
	for input, outputs := range sampleTests {
		expected := outputs[tokenizerType.name]
		var actual []string
		for _, term := range analyzer.Analyze([]byte(input)) {
			actual = append(actual, string(term.Term))
		}
		assert.DeepEqual(t, expected, actual)
	}
}

func TestBleveTokenizer(t *testing.T) {
	parsedTokens := int(100)
	testFileName := "./testDocuments/enwik8.txt.gz"

	path, err := FetchFileLoc(testFileName)
	assert.NilError(t, err)

	file, err := OpenLocalFile(path)
	assert.NilError(t, err)

	tokenizer, err := OpenBleveTokenizer(file)
	assert.NilError(t, err)

	tokens1 := make([]string, 0, parsedTokens)

	for i := 0; i < parsedTokens; i++ {
		token, counter, err := tokenizer.NextToken()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		assert.Equal(t, uint64(i), counter)
		tokens1 = append(tokens1, token)
	}

	err = tokenizer.Reset()
	assert.NilError(t, err)

	tokens2 := make([]string, 0, parsedTokens)
	for i := 0; i < parsedTokens; i++ {
		token, counter, err := tokenizer.NextToken()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		assert.Equal(t, uint64(i), counter)
		tokens2 = append(tokens2, token)
	}

	assert.DeepEqual(t, tokens1, tokens2)

	err = tokenizer.Close()
	assert.NilError(t, err)

	err = file.Close()
	assert.NilError(t, err)
}

func TestBlevePaulTokenizer(t *testing.T) {
	files, err := ioutil.ReadDir(GetInitialTestDocumentDir())
	assert.NilError(t, err)

	chans := make([]chan map[string]bool, len(files))
	for i, file := range files {
		if file.IsDir() {
			continue
		}

		chans[i] = make(chan map[string]bool)
		go func(f os.FileInfo, c chan map[string]bool) {
			defer close(c)
			path := path.Join(GetInitialTestDocumentDir(), f.Name())

			file, err := OpenLocalFile(path)
			assert.NilError(t, err)

			tokenizer, err := OpenBleveTokenizer(file)
			assert.NilError(t, err)

			unique := make(map[string]bool)
			for {
				token, _, err := tokenizer.NextToken()
				if err != nil {
					assert.Equal(t, err, io.EOF)
					break
				}
				unique[token] = true
			}

			err = tokenizer.Close()
			assert.NilError(t, err)

			err = file.Close()
			assert.NilError(t, err)

			c <- unique
		}(file, chans[i])
	}

	termmap := make(map[string]bool)
	for _, c := range chans {
		unique := <-c
		for k, v := range unique {
			termmap[k] = v
		}
	}

	terms := make([]string, 0, len(termmap))
	for k := range termmap {
		terms = append(terms, k)
	}

	sort.Strings(terms)
	fmt.Println("Term count:", len(terms))
	// fmt.Println(terms)
}
