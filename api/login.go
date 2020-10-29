package api

import (
	"context"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

// Login logs the user in, returning a Bearer Token.
// This token must henceforth be sent with all Reqs
// in the same session.
func Login(sdc client.StrongDocClient, userID, password, orgID string) (token string, err error) {
	return sdc.Login(userID, password, orgID)
}

// Logout retires the Bearer token in use, ending the session.
func Logout(sdc client.StrongDocClient) (status string, err error) {
	res, err := sdc.GetGrpcClient().Logout(context.Background(), &proto.LogoutReq{})
	if err != nil {
		return
	}
	status = res.Status
	return
}
