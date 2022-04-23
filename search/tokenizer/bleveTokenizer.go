package tokenizer

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
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

//////////////////////////////////////////////////////////////////
//
//                          BleveAnalyzer
//
//////////////////////////////////////////////////////////////////
type bTokenizerType struct {
	name   string
	config map[string]interface{}
}

var bttSingle = bTokenizerType{single.Name, nil}
var bttRegex = bTokenizerType{bleveRegex.Name,
	map[string]interface{}{
		"type":   bleveRegex.Name,
		"regexp": `[0-9a-zA-Z]*`,
	}}
var bttLetter = bTokenizerType{letter.Name, nil}
var bttWhiteSpace = bTokenizerType{whitespace.Name, nil}
var bttUnicode = bTokenizerType{bleveUnicode.Name, nil}
var bttWeb = bTokenizerType{web.Name, nil}

type bTokenFilterType struct {
	name string
}

var btftLowercase = bTokenFilterType{lowercase.Name}
var btftStemmer = bTokenFilterType{porter.Name}

func OpenBleveAnalyzer(tokenizerType TokenizerType) (Analyzer, error) {
	var analyzer *analysis.Analyzer = nil
	var err error

	switch tokenizerType {
	case TKZER_BLEVE:
		analyzer, err = openBleveAnalyzerStem()
		if err != nil {
			return nil, err
		}
	case TKZER_BLEVE_NO_STM:
		analyzer, err = openBleveAnalyzerNoStem()
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("Tokenizer type %v unsupported", tokenizerType)
	}

	return &bAnalyzer{
		analyzer:      analyzer,
		tokenizerType: tokenizerType,
	}, nil
}

func openBleveAnalyzerStem() (*analysis.Analyzer, error) {
	return openBleveAnalyzer(bttRegex, btftLowercase, btftStemmer)
}

func openBleveAnalyzerNoStem() (*analysis.Analyzer, error) {
	return openBleveAnalyzer(bttRegex, btftLowercase)
}

func openBleveAnalyzer(tokenizerType bTokenizerType, filterTypes ...bTokenFilterType) (*analysis.Analyzer, error) {
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

type bAnalyzer struct {
	analyzer      *analysis.Analyzer
	tokenizerType TokenizerType
}

func (ana *bAnalyzer) Analyze(input string) []string {
	bTokens := ana.analyzer.Analyze([]byte(input))

	tokens := make([]string, len(bTokens))
	for i, btoken := range bTokens {
		tokens[i] = string(btoken.Term)
	}

	return tokens
}

func (ana *bAnalyzer) Type() TokenizerType {
	return ana.tokenizerType
}

//////////////////////////////////////////////////////////////////
//
//                         BleveTokenizer
//
//////////////////////////////////////////////////////////////////
type BleveTokenizer interface {
	NextToken() (string, uint64, error)
	Type() TokenizerType
	Reset() error
	Close() error
}

type bleveTokenizer struct {
	storage     Storage
	wordCounter uint64
	analyzer    *analysis.Analyzer
	savedTokens analysis.TokenStream
	tkzerType   TokenizerType
}

// OpenBleveTokenizer opens a tokenizer
func OpenBleveTokenizer(source utils.Source) (BleveTokenizer, error) {
	return openBleveTokenizer(TKZER_BLEVE, source)
}

// OpenBleveTokenizerNoStem opens a tokenizer without stemmer
func OpenBleveTokenizerNoStem(source utils.Source) (BleveTokenizer, error) {
	return openBleveTokenizer(TKZER_BLEVE_NO_STM, source)
}

func openBleveTokenizer(tokenizerType TokenizerType, source utils.Source) (BleveTokenizer, error) {
	// init storage
	storage, err := openStorage(source)
	if err != nil {
		return nil, err
	}

	var analyzer *analysis.Analyzer = nil

	switch tokenizerType {
	case TKZER_BLEVE:
		analyzer, err = openBleveAnalyzerStem()
		if err != nil {
			return nil, err
		}
	case TKZER_BLEVE_NO_STM:
		analyzer, err = openBleveAnalyzerNoStem()
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("Tokenizer type %v unsupported", tokenizerType)
	}

	return &bleveTokenizer{
		storage:     storage,
		analyzer:    analyzer,
		wordCounter: 0,
		tkzerType:   tokenizerType,
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

func (token *bleveTokenizer) Type() TokenizerType {
	return token.tkzerType
}
