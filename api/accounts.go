package api

import (
	"context"
	"time"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// RegisterOrganization creates an organization. The user who
// created the organization is automatically an administrator.
func RegisterOrganization(orgName, orgAddr, orgEmail, adminName, adminPassword,
	adminEmail, source, sourceData string) (orgID, adminID string, err error) {

	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}
	resp, err := sdc.RegisterOrganization(context.Background(), &proto.RegisterOrganizationReq{
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
func RemoveOrganization(force bool) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	resp, err := sdc.RemoveOrganization(context.Background(), &proto.RemoveOrganizationReq{
		Force: force,
	})
	if err != nil {
		return
	}
	success = resp.GetSuccess()
	return
}

// RegisterUser creates new user if it doesn't already exist. Trying to create
// a user with an existing username throws an error.
//
// Requires administrator privileges.
func RegisterUser(user, pass, email string, admin bool) (userID string, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.RegisterUserReq{
		UserName: user,
		Password: pass,
		Email:    email,
		Admin:    admin,
	}
	res, err := sdc.RegisterUser(context.Background(), req)
	if err != nil {
		return
	}
	userID = res.UserID
	return
}

// User is the user of the organization
type User struct {
	UserName string
	UserID   string
	IsAdmin  bool
}

// ListUsers lists the users of the organization
func ListUsers() (users []User, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}
	req := &proto.ListUsersReq{}
	res, err := sdc.ListUsers(context.Background(), req)
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
func RemoveUser(user string) (count int64, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.RemoveUserReq{
		UserID: user,
	}
	res, err := sdc.RemoveUser(context.Background(), req)
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
func PromoteUser(userID string) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.PromoteUserReq{
		UserID: userID,
	}
	res, err := sdc.PromoteUser(context.Background(), req)
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
func DemoteUser(userID string) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.DemoteUserReq{
		UserID: userID,
	}
	res, err := sdc.DemoteUser(context.Background(), req)
	if err != nil {
		return
	}
	success = res.Success
	return
}

// AddSharableOrg adds a sharable Organization.
func AddSharableOrg(orgID string) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.AddSharableOrgReq{
		NewOrgID: orgID,
	}
	res, err := sdc.AddSharableOrg(context.Background(), req)
	if err != nil {
		return
	}

	success = res.Success
	return
}

// RemoveSharableOrg removes a sharable Organization.
func RemoveSharableOrg(orgID string) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.RemoveSharableOrgReq{
		RemoveOrgID: orgID,
	}
	res, err := sdc.RemoveSharableOrg(context.Background(), req)
	if err != nil {
		return
	}

	success = res.Success
	return
}

// SetMultiLevelSharing sets MultiLevel Sharing.
func SetMultiLevelSharing(enable bool) (success bool, err error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return
	}

	req := &proto.SetMultiLevelSharingReq{
		Enable: enable,
	}
	res, err := sdc.SetMultiLevelSharing(context.Background(), req)
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
func GetAccountInfo() (*AccountInfo, error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	req := &proto.GetAccountInfoReq{}
	resp, err := sdc.GetAccountInfo(context.Background(), req)
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
func GetUserInfo() (*UserInfo, error) {
	sdc, err := client.GetStrongDocClient()
	if err != nil {
		return nil, err
	}

	req := &proto.GetUserInfoReq{}
	resp, err := sdc.GetUserInfo(context.Background(), req)
	if err != nil {
		return nil, err
	}

	user, err := utils.ConvertStruct(resp, &UserInfo{})
	if err != nil {
		return nil, err
	}

	return user.(*UserInfo), nil
}
