package api

import (
	"github.com/overnest/strongdoc-go-sdk/client"
)

// Login logs the user in, returning a Bearer Token.
// This token must henceforth be sent with all Reqs
// in the same session.
func Login(sdc client.StrongDocClient, userID, password, orgID string) (err error) {
	// return sdc.Login(userID, password, orgID)
	return nil
}

// Logout retires the Bearer token in use, ending the session.
func Logout(sdc client.StrongDocClient) (status string, err error) {
	/*
		res, err := sdc.GetGrpcClient().Logout(context.Background(), &proto.LogoutReq{})
		if err != nil {
			return
		}
		status = res.Status
		return
	*/
	// return sdc.Logout()
	return "", nil
}
