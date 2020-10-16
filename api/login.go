package api

import (
	"context"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
)

// Login logs the user in, returning a Bearer Token.
// This token must henceforth be sent with all Reqs
// in the same session.
func Login(userID, password, orgID, keyPassword string) (token string, err error) {
	sdm, err := client.GetStrongDocManager()
	if err != nil {
		return "", err
	}

	return sdm.Login(userID, password, orgID, keyPassword)
}

// Logout retires the Bearer token in use, ending the session.
func Logout() (status string, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}
	res, err := sdc.Logout(context.Background(), &proto.LogoutReq{})
	if err != nil {
		return
	}
	status = res.Status
	return
}
