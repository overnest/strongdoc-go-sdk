package api

import (
	"context"
	"log"

	"github.com/overnest/strongdoc-go/client"
	"github.com/overnest/strongdoc-go/proto"
)

// RegisterOrganization creates an organization. The user who
// created the organization is automatically an administrator.
func RegisterOrganization(orgName, orgAddr, adminName, adminPassword, adminEmail string) (orgID, adminID string, err error) {
	noAuthConn, err := client.ConnectToServerNoAuth()
	if err != nil {
		log.Fatalf("Can not obtain no auth connection %s", err)
		return
	}
	defer noAuthConn.Close()

	noAuthClient := proto.NewStrongDocServiceClient(noAuthConn)
	resp, err := noAuthClient.RegisterOrganization(context.Background(), &proto.RegisterOrganizationRequest{
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

// RemoveOrganization removes an organization, and all of its
// users, documents, and other data that it owns.
//
// Requires administrator privileges.
func RemoveOrganization(token string, force bool) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	resp, err := authClient.RemoveOrganization(context.Background(), &proto.RemoveOrganizationRequest{
		Force: force,
	})
	if err != nil {
		return
	}
	success = resp.GetSuccess()
	return
}

// PromoteUser promotes a regular user to administrator
// privilege level.
//
// Requires administrator privileges.
func PromoteUser(token, userId string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.PromoteUserRequest{
		UserID:               userId,
	}
	res, err := authClient.PromoteUser(context.Background(), req)
	if err != nil {
		return
	}
	success = res.Success
	return
}

// DemoteUser demotes an administrator to regular user level.
// privilege level.
//
// Requires administrator privileges.
func DemoteUser(token, userId string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.DemoteUserRequest{
		UserID:               userId,
	}
	res, err := authClient.DemoteUser(context.Background(), req)
	if err != nil {
		return
	}
	success = res.Success
	return
}

// Creates new user if it doesn't already exist. Trying to create
// a user with an existing username throws an error.
//
// Requires administrator privileges.
func RegisterUser(token, user, pass, email string, admin bool) (userId string, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.RegisterUserRequest{
		UserName:             user,
		Password:             pass,
		Email:                email,
		Admin:                admin,
	}
	res, err := authClient.RegisterUser(context.Background(), req)
	if err != nil {
		return
	}
	userId = res.UserID
	return
}

// Removes the user from the organization. The users documents still exist,
// but belong to the organization, only accessible by the admin of
// their former organization.
//
// Requires administrator privileges.
func RemoveUser(token, user string) (count int64, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.RemoveUserRequest{
		UserID:             user,
	}
	res, err := authClient.RemoveUser(context.Background(), req)
	if err != nil {
		return
	}
	count = res.Count
	return
}

type User struct {
	UserName string
	UserID string
	IsAdmin bool
}

func ListUsers(token string) (users []User, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()

	authClient := proto.NewStrongDocServiceClient(authConn)
	req := &proto.ListUsersRequest{}
	res, err := authClient.ListUsers(context.Background(), req)
	if err != nil {
		return
	}
	users = make([]User, 0)
	for _, protoUser := range res.OrgUsers {
		user := User{
			protoUser.UserName,
			protoUser.UserID,
			protoUser.IsAdmin,
		}
		users = append(users, user)
	}
	return
}

// AddSharableOrg adds a sharable Organization.
func AddSharableOrg(token, orgID string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.AddSharableOrgRequest{
		NewOrgID: orgID,
	}
	res, err := authClient.AddSharableOrg(context.Background(), req)

	success = res.Success
	return
}

// RemoveSharableOrg removes a sharable Organization.
func RemoveSharableOrg(token, orgID string) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.RemoveSharableOrgRequest{
		RemoveOrgID: orgID,
	}
	res, err := authClient.RemoveSharableOrg(context.Background(), req)

	success = res.Success
	return
}

// SetMultiLevelSharing sets MultiLevel Sharing.
func SetMultiLevelSharing(token string, enable bool) (success bool, err error) {
	authConn, err := client.ConnectToServerWithAuth(token)
	if err != nil {
		log.Fatalf("Can not obtain auth connection %s", err)
		return
	}
	defer authConn.Close()
	authClient := proto.NewStrongDocServiceClient(authConn)

	req := &proto.SetMultiLevelSharingRequest{
		Enable: enable,
	}
	res, err := authClient.SetMultiLevelSharing(context.Background(), req)

	success = res.Success
	return
}

