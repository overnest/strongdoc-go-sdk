package api

import (
	"context"
	"log"

	"github.com/strongdoc/client/go/client"
	"github.com/strongdoc/client/go/proto"
)

// Login logs the user in, returning a Bearer Token.
// This token must henceforth be sent with all requests
// in the same session.
func Login(userID, password, orgID string) (string, error) {
	noAuthConn, err := client.ConnectToServerNoAuth()
	if err != nil {
		log.Fatalf("Can not obtain no auth connection %s", err)
		return "", err
	}
	defer noAuthConn.Close()

	noauthClient := proto.NewStrongDocServiceClient(noAuthConn)
	res, err := noauthClient.Login(context.Background(), &proto.LoginRequest{
		UserID: userID, Password: password, OrgID: orgID})
	if err != nil {
		return "", err
	}

	return res.Token, nil
}

// Logout retires the Bearer token in use, ending the session.
func Logout(token string) (err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return err
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	_, err = authClient.Logout(context.Background(), &proto.LogoutRequest{})
	if err != nil {
		return err
	}

	return nil
}
