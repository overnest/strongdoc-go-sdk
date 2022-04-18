package tokenizer

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/utils"
	"gotest.tools/assert"
)

func TestRegex(t *testing.T) {
	regexGoodMap := map[*regexp.Regexp][]string{
		utils.MimeGzip: {
			"application/x-gzip",
			"application/gzip",
		},
		utils.MimeText: {
			"text/plain",
		},
	}

	regexGoodBad := map[*regexp.Regexp][]string{
		utils.MimeGzip: {
			"application/x-gzipp",
			"application/x-gzipp",
			"application/gzipabc",
			"applications/gzip",
		},
		utils.MimeText: {
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
		bttSingle.name:     []string{""},
		bttLetter.name:     nil,
		bttWhiteSpace.name: nil,
		bttRegex.name:      nil,
		bttUnicode.name:    nil,
		bttWeb.name:        nil,
	},
	"123 abc 4c5d": {
		bttSingle.name:     []string{"123 abc 4c5d"},
		bttLetter.name:     []string{"abc", "c", "d"},
		bttWhiteSpace.name: []string{"123", "abc", "4c5d"},
		bttRegex.name:      []string{"123", "abc", "4c5d"},
		bttUnicode.name:    []string{"123", "abc", "4c5d"},
		bttWeb.name:        []string{"123", "abc", "4c5d"},
	},
	" Hello World.": {
		bttSingle.name:     []string{" hello world."},
		bttLetter.name:     []string{"hello", "world"},
		bttWhiteSpace.name: []string{"hello", "world."},
		bttRegex.name:      []string{"hello", "world"},
		bttUnicode.name:    []string{"hello", "world"},
		bttWeb.name:        []string{"hello", "world"},
	},
	"Dominique@mcdiabetes.com": {
		bttSingle.name:     []string{"dominique@mcdiabetes.com"},
		bttLetter.name:     []string{"dominique", "mcdiabetes", "com"},
		bttWhiteSpace.name: []string{"dominique@mcdiabetes.com"},
		bttRegex.name:      []string{"dominique", "mcdiabetes", "com"},
		bttUnicode.name:    []string{"dominique", "mcdiabetes.com"},
		bttWeb.name:        []string{"d", "ominique@mcdiabetes.com"},
	},
	"That http://blevesearch.com": {
		bttSingle.name:     []string{"that http://blevesearch.com"},
		bttLetter.name:     []string{"that", "http", "blevesearch", "com"},
		bttWhiteSpace.name: []string{"that", "http://blevesearch.com"},
		bttRegex.name:      []string{"that", "http", "blevesearch", "com"},
		bttUnicode.name:    []string{"that", "http", "blevesearch.com"},
		bttWeb.name:        []string{"that", "http://blevesearch.com"},
	},
	"Hello info@blevesearch.com": {
		bttSingle.name:     []string{"hello info@blevesearch.com"},
		bttLetter.name:     []string{"hello", "info", "blevesearch", "com"},
		bttWhiteSpace.name: []string{"hello", "info@blevesearch.com"},
		bttRegex.name:      []string{"hello", "info", "blevesearch", "com"},
		bttUnicode.name:    []string{"hello", "info", "blevesearch.com"},
		bttWeb.name:        []string{"hello", "info@blevesearch.com"},
	},
}

func TestBleveAnalyzer(t *testing.T) {
	// Single Token: the entire input bytes as a single token
	testBleveAnalyzer(t, bttSingle, btftLowercase)

	// Letter Token: simply identifies tokens as sequences of Unicode runes that are part of the Letter category
	testBleveAnalyzer(t, bttLetter, btftLowercase)

	// Whitespace: identifies tokens as sequences of Unicode runes that are NOT part of the Space category
	testBleveAnalyzer(t, bttWhiteSpace, btftLowercase)

	// Regular Expression: tokenize input using a configurable regular expression
	testBleveAnalyzer(t, bttRegex, btftLowercase)

	// Unicode Tokenizer: uses the segment library to perform Unicode Text Segmentation on word boundaries
	// follow: http://www.unicode.org/reports/tr29/#Word_Boundary_Rules
	testBleveAnalyzer(t, bttUnicode, btftLowercase)

	// Web tokenizer: better support for email, url and web content (regex implementation)
	testBleveAnalyzer(t, bttWeb, btftLowercase)

	// ICU tokenizer: uses the ICU library to tokenize the input
	// better support for some Asian languages by using a dictionary-based approach to identify words

}

func testBleveAnalyzer(t *testing.T, tokenizerType bTokenizerType, filter_type ...bTokenFilterType) {
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

	path, err := utils.FetchFileLoc(testFileName)
	assert.NilError(t, err)

	file, err := utils.OpenLocalFile(path)
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
	files, err := ioutil.ReadDir(utils.GetInitialTestDocumentDir())
	assert.NilError(t, err)

	chans := make([]chan map[string]bool, len(files))
	for i, file := range files {
		if file.IsDir() {
			continue
		}

		chans[i] = make(chan map[string]bool)
		go func(f os.FileInfo, c chan map[string]bool) {
			defer close(c)
			path := path.Join(utils.GetInitialTestDocumentDir(), f.Name())

			file, err := utils.OpenLocalFile(path)
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
