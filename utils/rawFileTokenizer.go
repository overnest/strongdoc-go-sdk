package utils

import (
	"github.com/blevesearch/bleve/analysis"
	"io"
	"strings"
)

// RawFileTokenizer tokenizes a file
type RawFileTokenizer interface {
	NextRawToken() (rawToken string, space rune, err error)
	Analyze(rawToken string) (terms []string)
	IsPureTerm(rawToken string) (isPure bool, pureTerm string)
	GetAllPureTerms() ([]string, error)
	Close() (err error)
}

type rawFileTokenizer struct {
	storage  Storage
	analyzer *analysis.Analyzer
}

// OpenRawFileTokenizer opens a reader for tokenization
func OpenRawFileTokenizer(source Source) (RawFileTokenizer, error) {
	storage, err := openStorage(source)
	if err != nil {
		return nil, err
	}

	analyzer, err := openAnalyzer()

	return &rawFileTokenizer{
		storage:  storage,
		analyzer: analyzer,
	}, nil
}

func (token *rawFileTokenizer) Close() (err error) {
	return token.storage.close()
}

func (tokenizer *rawFileTokenizer) NextRawToken() (rawToken string, space rune, err error) {
	var rawTokenBytes []byte
	rawTokenBytes, space, err = tokenizer.storage.nextRawToken()
	if err != nil {
		return
	}
	rawToken = string(rawTokenBytes)
	return
}

// analyze rawToken
func (tokenizer *rawFileTokenizer) Analyze(rawToken string) (terms []string) {
	_, terms = tokenizer.getWords(rawToken)
	return
}

func (tokenizer *rawFileTokenizer) IsPureTerm(rawToken string) (isPure bool, pureWord string) {
	var words []string
	isPure, words = tokenizer.getWords(rawToken)
	if isPure {
		pureWord = words[0]
	}
	return
}

func (tokenizer *rawFileTokenizer) getWords(rawToken string) (isPure bool, terms []string) {
	isPure = false
	tokens := tokenizer.analyzer.Analyze([]byte(rawToken))
	for _, token := range tokens {
		terms = append(terms, (string)(token.Term))
	}
	if len(terms) == 1 && terms[0] == strings.ToLower(rawToken) {
		isPure = true
	}
	return
}

// get pure words from source
func (tokenizer *rawFileTokenizer) GetAllPureTerms() ([]string, error) {
	// term map
	pureTermsMap := make(map[string]bool)
	unpureTermsMap := make(map[string]bool)
	for rawToken, _, err := tokenizer.NextRawToken(); err != io.EOF; rawToken, _, err = tokenizer.NextRawToken() {
		if err != nil && err != io.EOF {
			return nil, err
		}
		isPure, words := tokenizer.getWords(rawToken)
		if isPure {
			pureTermsMap[words[0]] = true
		} else {
			for _, word := range words {
				unpureTermsMap[word] = true
			}
		}
	}
	var terms []string
	for term, _ := range pureTermsMap {
		if !unpureTermsMap[term] {
			terms = append(terms, term)
		}
	}
	return terms, nil
}
