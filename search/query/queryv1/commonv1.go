package queryv1

import (
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/tokenizer"
)

// Use the same analyzer on the query terms
func AnalyzeTerms(terms []string) ([]string, map[string]string, error) {
	analyzer, err := tokenizer.OpenBleveAnalyzer(tokenizer.TKZER_BLEVE)
	if err != nil {
		return nil, nil, err
	}

	analyzedTerms := make([]string, len(terms))
	analyzedTermsMap := make(map[string]string)
	for i, term := range terms {
		tokens := analyzer.Analyze(term)
		if len(tokens) != 1 {
			return analyzedTerms, analyzedTermsMap, errors.Errorf("HashedTerm analysis error %v:%v", term, tokens)
		}
		token := tokens[0]
		analyzedTerms[i] = token
		analyzedTermsMap[token] = term
	}

	return analyzedTerms, analyzedTermsMap, nil
}
