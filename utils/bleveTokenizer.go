package utils

import (
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/analyzer/web"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/token/porter"
	"github.com/blevesearch/bleve/analysis/tokenizer/letter"
	bleveRegex "github.com/blevesearch/bleve/analysis/tokenizer/regexp"
	"github.com/blevesearch/bleve/analysis/tokenizer/single"
	bleveUnicode "github.com/blevesearch/bleve/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/analysis/tokenizer/whitespace"
	"github.com/blevesearch/bleve/registry"
)

//////////////////////////////////////////////////////////////////
//
//               BleveAnalyzer
//
//////////////////////////////////////////////////////////////////
// tokenizer type
type Tokenizer_Type struct {
	name   string
	config map[string]interface{}
}

var SINGLE = Tokenizer_Type{single.Name, nil}
var REGEX = Tokenizer_Type{bleveRegex.Name,
	map[string]interface{}{
		"type":   bleveRegex.Name,
		"regexp": `[0-9a-zA-Z]*`,
	}}
var LETTER = Tokenizer_Type{letter.Name, nil}
var WHITESPACE = Tokenizer_Type{whitespace.Name, nil}
var UNICODE = Tokenizer_Type{bleveUnicode.Name, nil}
var WEB = Tokenizer_Type{web.Name, nil}

// token filter type
type Token_Filter_Type struct {
	name string
}

var LOWERCASE = Token_Filter_Type{lowercase.Name}
var STEMMER = Token_Filter_Type{porter.Name}

func OpenBleveAnalyzer() (*analysis.Analyzer, error) {
	return openBleveAnalyzer(REGEX, LOWERCASE, STEMMER)
}

func openBleveAnalyzer(tokenizerType Tokenizer_Type, filterTypes ...Token_Filter_Type) (*analysis.Analyzer, error) {
	cache := registry.NewCache()

	var tokenizer analysis.Tokenizer
	var err error
	if tokenizerType.config == nil {
		tokenizer, err = cache.TokenizerNamed(tokenizerType.name)
	} else {
		tokenizer, err = cache.DefineTokenizer(tokenizerType.name, tokenizerType.config)
	}
	if err != nil {
		return nil, err
	}

	var tokenFilters []analysis.TokenFilter
	for _, filterType := range filterTypes {
		filter, err := cache.TokenFilterNamed(filterType.name)
		if err != nil {
			return nil, err
		}
		tokenFilters = append(tokenFilters, filter)
	}

	return &analysis.Analyzer{
		Tokenizer:    tokenizer,
		TokenFilters: tokenFilters,
	}, nil
}

//////////////////////////////////////////////////////////////////
//
//               BleveTokenizer
//
//////////////////////////////////////////////////////////////////
type BleveTokenizer interface {
	NextToken() (string, uint64, error)
	Reset() error
	Close() error
}

type bleveTokenizer struct {
	storage     Storage
	wordCounter uint64
	analyzer    *analysis.Analyzer
	savedTokens analysis.TokenStream
}

// OpenBleveTokenizer opens a tokenizer
func OpenBleveTokenizer(source Source) (BleveTokenizer, error) {
	// init storage
	storage, err := openStorage(source)
	if err != nil {
		return nil, err
	}

	// init analyzer
	analyzer, err := OpenBleveAnalyzer()
	if err != nil {
		return nil, err
	}

	return &bleveTokenizer{
		storage:     storage,
		analyzer:    analyzer,
		wordCounter: 0,
	}, nil
}

// Close closes the tokenizer. It does not close the underlying source
func (token *bleveTokenizer) Close() error {
	return token.storage.close()
}

func (token *bleveTokenizer) Reset() error {
	err := token.storage.reset()
	if err != nil {
		return err
	}
	token.wordCounter = 0
	return nil
}

// bleve tokens
// return token, counter, error
func (token *bleveTokenizer) NextToken() (string, uint64, error) {
	for len(token.savedTokens) == 0 {
		input, _, err := token.storage.nextRawToken()
		if err != nil {
			return "", 0, err
		}
		newTokens := token.analyzer.Analyze(input)
		token.savedTokens = append(token.savedTokens, newTokens...)
	}
	first := string(token.savedTokens[0].Term)
	token.savedTokens = token.savedTokens[1:]
	counter := token.wordCounter
	token.wordCounter++
	return first, counter, nil
}
