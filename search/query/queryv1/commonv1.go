package queryv1

import (
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// Use the same analyzer on the query terms
func AnalyzeTerms(terms []string) ([]string, map[string]string, error) {
	analyzer, err := utils.OpenBleveAnalyzer()
	if err != nil {
		return nil, nil, err
	}

	analyzedTerms := make([]string, len(terms))
	analyzedTermsMap := make(map[string]string)
	for i, term := range terms {
		tokens := analyzer.Analyze([]byte(term))
		if len(tokens) != 1 {
			return analyzedTerms, analyzedTermsMap, errors.Errorf("Term analysis error %v:%v", term, tokens)
		}
		token := tokens[0].String()
		analyzedTerms[i] = token
		analyzedTermsMap[token] = term
	}

	return analyzedTerms, analyzedTermsMap, nil
}
