package api

import (
	"context"
	"fmt"
	"log"

	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
)

// Login logs the user in, returning a Bearer Token.
// This token must henceforth be sent with all requests
// in the same session.
func Login(userID, password, orgID string) (token string, err error) {
	noAuthConn, err := client.ConnectToServerNoAuth()
	if err != nil {
		log.Fatalf("Can not obtain no auth connection %s", err)
		return
	}
	defer noAuthConn.Close()

	noauthClient := proto.NewStrongDocServiceClient(noAuthConn)
	res, err := noauthClient.Login(context.Background(), &proto.LoginRequest{
		UserID: userID, Password: password, OrgID: orgID})
	if err != nil {
		err = fmt.Errorf("Login err: [%v]\n", err)
		return
	}

	token = res.Token
	return token, nil
}

// Logout retires the Bearer token in use, ending the session.
func Logout(token string) (status string, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	res, err := authClient.Logout(context.Background(), &proto.LogoutRequest{})
	if err != nil {
		return
	}
	status = res.Status

	return
}
