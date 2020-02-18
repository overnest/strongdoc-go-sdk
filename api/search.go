package api

import (
	"context"
	"log"

	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
)

// Search searches the the terms in the uploaded or encrypted documents
func Search(token, query string) ([]*proto.DocumentResult, error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return nil, err
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	result, err := authClient.Search(context.Background(),
		&proto.SearchRequest{Query: query})
	return result.GetHits(), nil
}
