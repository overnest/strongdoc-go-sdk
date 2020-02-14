package api

import (
	"context"
	"log"

	"github.com/strongdoc/client/go/client"
	"github.com/strongdoc/client/go/proto"
)

// RegisterOrganization creates an organization
func RegisterOrganization(orgName, orgAddr, adminName, adminPassword, adminEmail string) (orgID, adminID string, err error) {
	noAuthConn, err := client.ConnectToServerNoAuth()
	if err != nil {
		log.Fatalf("Can not obtain no auth connection %s", err)
		return
	}
	defer noAuthConn.Close()

	noauthClient := proto.NewStrongDocServiceClient(noAuthConn)
	resp, err := noauthClient.RegisterOrganization(context.Background(), &proto.RegisterOrganizationRequest{
		OrgName:         orgName,
		OrgAddr:         orgAddr,
		UserName:        adminName,
		Password:        adminPassword,
		Email:           adminEmail,
		SharableOrgs:    nil,
		MultiLevelShare: false,
	})
	if err != nil {
		return
	}

	return resp.GetOrgID(), resp.GetUserID(), nil
}

// RemoveOrganization removes an organization
func RemoveOrganization(token string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	resp, err := authClient.RemoveOrganization(context.Background(), &proto.RemoveOrganizationRequest{
		Force: true,
	})
	if err != nil {
		return
	}

	success = resp.GetSuccess()
	return
}
