package api

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/overnest/strongsalt-crypto-go/pake/srp"
)

// todo RegisterOrganization removed from sdk

// RegisterOrganization creates an organization. The user who
// created the organization is automatically an administrator.
func RegisterOrganization(sdc client.StrongDocClient,
	orgName, orgAddr, orgEmail, adminName, adminPassword, adminEmail, source, sourceData string,
	kdfMeta []byte, userPubKey []byte, encUserPriKey []byte, orgPubKey []byte, encOrgPriKey []byte) (orgID, adminID string, err error) {

	srpSession, err := srp.NewFromVersion(1)
	if err != nil {
		return
	}
	srpVerifier, err := srpSession.Verifier([]byte(""), []byte(adminPassword))
	_, verifierString := srpVerifier.Encode()

	resp, err := sdc.GetGrpcClient().RegisterOrganization(context.Background(), &proto.RegisterOrganizationReq{
		OrgName:         orgName,
		OrgAddr:         orgAddr,
		OrgEmail:        orgEmail,
		UserName:        adminName,
		Password:        adminPassword,
		AdminEmail:      adminEmail,
		SharableOrgs:    nil,
		MultiLevelShare: false,
		Source:          source,
		SourceData:      sourceData,
		KdfMetadata:     base64.URLEncoding.EncodeToString(kdfMeta), //todo base64
		UserPubKey:      base64.URLEncoding.EncodeToString(userPubKey),
		EncUserPriKey:   base64.URLEncoding.EncodeToString(encUserPriKey),
		OrgPubKey:       base64.URLEncoding.EncodeToString(orgPubKey),
		EncOrgPriKey:    base64.URLEncoding.EncodeToString(encOrgPriKey),
		AdminLoginData: &proto.RegisterLoginData{
			LoginType:    proto.LoginType_SRP,
			LoginVersion: 1,
			SrpVerifier:  verifierString,
		},
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
func RemoveOrganization(sdc client.StrongDocClient, force bool) (success bool, err error) {
	resp, err := sdc.GetGrpcClient().RemoveOrganization(context.Background(), &proto.RemoveOrganizationReq{
		Force: force,
	})
	if err != nil {
		return
	}
	success = resp.GetSuccess()
	return
}

// InviteUser sends invitation email to potential user
//
// Requires administrator privileges.
func InviteUser(sdc client.StrongDocClient, email string, expireTime int64) (bool, error) {
	req := &proto.InviteUserReq{
		Email:      email,
		ExpireTime: expireTime,
	}
	res, err := sdc.InviteUser(context.Background(), req)
	if err != nil {
		return false, err
	}
	return res.Success, nil
}

// Invitation is the registration invitation
type Invitation struct {
	Email      string
	ExpireTime *timestamp.Timestamp
	CreateTime *timestamp.Timestamp
}

// ListInvitations lists active non-expired invitations
//
// Requires administrator privileges.
func ListInvitations(sdc client.StrongDocClient) ([]Invitation, error) {
	req := &proto.ListInvitationsReq{}
	res, err := sdc.ListInvitations(context.Background(), req)
	if err != nil {
		return nil, err
	}
	invitations, err := utils.ConvertStruct(res.Invitations, []Invitation{})
	if err != nil {
		return nil, err
	}
	return *(invitations.(*[]Invitation)), nil
}

// RevokeInvitation revoke invitation
//
// Requires administrator privileges.
func RevokeInvitation(sdc client.StrongDocClient, email string) (bool, bool, error) {
	req := &proto.RevokeInvitationReq{
		Email: email,
	}
	res, err := sdc.RevokeInvitation(context.Background(), req)
	if err != nil {
		return false, false, err
	}
	return res.Success, res.CodeAlreadyUsed, nil
}

// RegisterUser creates new user if it doesn't already exist. Trying to create
// a user with an existing username throws an error.
//
// Does not require Login
func RegisterWithInvitation(sdc client.StrongDocClient,
	invitationCode string, orgID string, userName string, userPassword string, userEmail string,
	kdfMetaBytes []byte, pubKeyBytes []byte, encPriKeyBytes []byte) (string, string, bool, error) {

	srpSession, err := srp.NewFromVersion(1)
	if err != nil {
		return "", "", false, err
	}
	srpVerifier, err := srpSession.Verifier([]byte(""), []byte(userPassword))
	_, verifierString := srpVerifier.Encode()

	req := &proto.RegisterUserReq{
		InvitationCode: invitationCode,
		OrgID:          orgID,
		Email:          userEmail,
		KdfMetadata:    base64.URLEncoding.EncodeToString(kdfMetaBytes),
		UserPubKey:     base64.URLEncoding.EncodeToString(pubKeyBytes),
		EncUserPriKey:  base64.URLEncoding.EncodeToString(encPriKeyBytes),
		UserName:       userName,
		Password:       userPassword,
		LoginData: &proto.RegisterLoginData{
			LoginType:    proto.LoginType_SRP,
			LoginVersion: 1,
			SrpVerifier:  verifierString,
		},
	}
	res, err := sdc.GetGrpcClient().RegisterUser(context.Background(), req)
	if err != nil {
		return "", "", false, err
	}
	return res.UserID, res.OrgID, res.Success, nil
}

// User is the user of the organization
type User struct {
	UserName string
	UserID   string
	IsAdmin  bool
}

// ListUsers lists the users of the organization
func ListUsers(sdc client.StrongDocClient) (users []User, err error) {
	req := &proto.ListUsersReq{}
	res, err := sdc.GetGrpcClient().ListUsers(context.Background(), req)
	if err != nil {
		return
	}

	usersi, err := utils.ConvertStruct(res.OrgUsers, []User{})
	if err != nil {
		return nil, err
	}

	return *(usersi.(*[]User)), nil
}

// RemoveUser removes the user from the organization. The users documents still exist,
// but belong to the organization, only accessible by the admin of
// their former organization.
//
// Requires administrator privileges.
func RemoveUser(sdc client.StrongDocClient, user string) (count int64, err error) {
	req := &proto.RemoveUserReq{
		UserID: user,
	}
	res, err := sdc.GetGrpcClient().RemoveUser(context.Background(), req)
	if err != nil {
		return
	}
	count = res.Count
	return
}

// PromoteUser promotes a regular user to administrator
// privilege level.
//
// Requires administrator privileges.
func PromoteUser(sdc client.StrongDocClient, userID string) (success bool, err error) {
	req := &proto.PromoteUserReq{
		UserID: userID,
	}
	res, err := sdc.GetGrpcClient().PromoteUser(context.Background(), req)
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
func DemoteUser(sdc client.StrongDocClient, userID string) (success bool, err error) {
	req := &proto.DemoteUserReq{
		UserID: userID,
	}
	res, err := sdc.GetGrpcClient().DemoteUser(context.Background(), req)
	if err != nil {
		return
	}
	success = res.Success
	return
}

// AddSharableOrg adds a sharable Organization.
func AddSharableOrg(sdc client.StrongDocClient, orgID string) (success bool, err error) {
	req := &proto.AddSharableOrgReq{
		NewOrgID: orgID,
	}
	res, err := sdc.GetGrpcClient().AddSharableOrg(context.Background(), req)
	if err != nil {
		return
	}

	success = res.Success
	return
}

// RemoveSharableOrg removes a sharable Organization.
func RemoveSharableOrg(sdc client.StrongDocClient, orgID string) (success bool, err error) {
	req := &proto.RemoveSharableOrgReq{
		RemoveOrgID: orgID,
	}
	res, err := sdc.GetGrpcClient().RemoveSharableOrg(context.Background(), req)
	if err != nil {
		return
	}

	success = res.Success
	return
}

// SetMultiLevelSharing sets MultiLevel Sharing.
func SetMultiLevelSharing(sdc client.StrongDocClient, enable bool) (success bool, err error) {
	req := &proto.SetMultiLevelSharingReq{
		Enable: enable,
	}
	res, err := sdc.GetGrpcClient().SetMultiLevelSharing(context.Background(), req)
	if err != nil {
		return
	}

	success = res.Success
	return
}

// AccountInfo is info on the organization account
type AccountInfo struct {
	// Account's orgID
	OrgID string
	// Account's subscription info
	Subscription *Subscription
	// List of all account's payments
	Payments []*Payment
	// The address of the organization
	OrgAddress string
	// The ability to "reshare" a document that was shared to him/her to another user
	MultiLevelShare bool
	// The list of sharable organization IDs.
	SharableOrgs []string
}

// Subscription is the subscript type of the organization
type Subscription struct {
	// Subscription type (AWS Marketplace, Credit Card, etc.)
	Type string
	// State of the subscription (Created, Subscribed, Unsubscribed, etc.)
	Status string
}

// Payment is the payment information for the organization
type Payment struct {
	// Timestamp of the payment billing transaction
	BilledAt *time.Time
	// Start of the payment period
	PeriodStart *time.Time
	// End of the payment period
	PeriodEnd *time.Time
	// Amount of  payment
	Amount float64
	// Payment status ("No Payment","Zero Payment","Payment Pending","Payment Success","Payment Failed")
	Status string
}

//GetAccountInfo obtain information about the account
func GetAccountInfo(sdc client.StrongDocClient) (*AccountInfo, error) {
	req := &proto.GetAccountInfoReq{}
	resp, err := sdc.GetGrpcClient().GetAccountInfo(context.Background(), req)
	if err != nil {
		return nil, err
	}

	account, err := utils.ConvertStruct(resp, &AccountInfo{})
	if err != nil {
		return nil, err
	}

	return account.(*AccountInfo), err
}

// UserInfo is info on the user account
type UserInfo struct {
	// The user's userID
	UserID string
	// The user's email
	Email string
	// The user's name
	UserName string
	// The user's orgID
	OrgID string
	// Whether the user is an admin
	IsAdmin bool
}

//GetUserInfo obtain information about logged user
func GetUserInfo(sdc client.StrongDocClient) (*UserInfo, error) {
	req := &proto.GetUserInfoReq{}
	resp, err := sdc.GetGrpcClient().GetUserInfo(context.Background(), req)
	if err != nil {
		return nil, err
	}

	user, err := utils.ConvertStruct(resp, &UserInfo{})
	if err != nil {
		return nil, err
	}

	return user.(*UserInfo), nil
}
