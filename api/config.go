package api

import (
	"context"
	"log"

	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
)

// GetConfiguration returns a string representing the configurations
// set.
func GetConfiguration(token string) (config string, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.GetConfigurationReq{}
	res, err := authClient.GetConfiguration(context.Background(), req)
	if err != nil {
		return
	}
	config = res.Configuration
	return
}