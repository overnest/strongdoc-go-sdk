package tokenizer

import (
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

type TokenizerType int

// DO NOT MODIFY. ONLY APPEND
// This value is written into index files. Never modify. Only append
const (
	TKZER_NONE = iota
	TKZER_BLEVE
	TKZER_BLEVE_NO_STM
)

type Analyzer interface {
	Analyze(input string) []string
	Type() TokenizerType
}

type Tokenizer interface {
	NextToken() (string, uint64, error)
	Type() TokenizerType
	Reset() error
	Close() error
}

func OpenAnalyzer(tokenizerType TokenizerType) (Analyzer, error) {
	switch tokenizerType {
	case TKZER_BLEVE, TKZER_BLEVE_NO_STM:
		return OpenBleveAnalyzer(tokenizerType)
	default:
		return nil, errors.Errorf("Tokenizer type %v unsupported", tokenizerType)
	}
}

func OpenTokenizer(tokenizerType TokenizerType, source utils.Source) (Tokenizer, error) {
	switch tokenizerType {
	case TKZER_BLEVE, TKZER_BLEVE_NO_STM:
		return openBleveTokenizer(tokenizerType, source)
	default:
		return nil, errors.Errorf("Tokenizer type %v unsupported", tokenizerType)
	}
}