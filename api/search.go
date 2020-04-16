package api

import (
	"context"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// DocumentResult contains the document search result
type DocumentResult struct {
	// The document ID that contains the query terms.
	DocID string
	// The score of the search result.
	Score float64
}

// Search searches for the queries in the uploaded and encrypted documents.
// The list of document IDs and scores are included in the result.
func Search(query string) ([]*DocumentResult, error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	result, err := sdc.Search(context.Background(),
		&proto.SearchReq{Query: query})
	if err != nil {
		return nil, err
	}

	hits, err := utils.ConvertStruct(result.GetHits(), []*DocumentResult{})
	if err != nil {
		return nil, err
	}

	return *hits.(*[]*DocumentResult), nil
}
