package test

import (
	"fmt"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/stretchr/testify/require"
)

func TestRegistration(t *testing.T) {
	assert := require.New(t)

	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NoError(err)

	user, succ, err := api.RegisterUser(sdc, "UserName", "UserPasswd", "email@user.com")
	if err != nil {
		gerr := api.FromError(err)
		fmt.Println(gerr)
	}

	assert.NoError(err, err)
	assert.True(succ)
	fmt.Println(user)
}

// func TestLogin(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 1)
// 	orgData := orgs[0]
// 	userData := orgUsers[0][0]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, userData)
// 	assert.NilError(t, err)

// 	// login with wrong password
// 	err = api.Login(sdc, userData.UserID, "wrongPassword", orgData.OrgID)
// 	//assert.ErrorContains(t, err, "Password does not match")
// 	assert.Assert(t, err != nil)

// 	// login with wrong userID
// 	err = api.Login(sdc, "wrongUserID", userData.Password, orgData.OrgID)
// 	//assert.ErrorContains(t, err, "Not a valid form")
// 	assert.Assert(t, err != nil)

// 	// login succeed
// 	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
// 	assert.NilError(t, err)
// 	//assert.Check(t, token != "", "empty token")
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)

// 	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
// 	assert.NilError(t, err)
// 	//assert.Check(t, token != "", "empty token")
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)
// }

// func TestBusyLogin(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 1)

// 	orgData := orgs[0]
// 	userData := orgUsers[0][0]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, userData)
// 	assert.NilError(t, err)

// 	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
// 	assert.NilError(t, err)
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)
// 	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
// 	assert.NilError(t, err)
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)
// 	// duplicate logout
// 	_, err = api.Logout(sdc)
// 	assert.ErrorContains(t, err, " The JWT user is logged out")
// 	// do something needs login
// 	_, err = api.ListInvitations(sdc)
// 	assert.ErrorContains(t, err, " The JWT user is logged out")
// }

// func TestInviteUser(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 2)

// 	orgData := orgs[0]
// 	userData := orgUsers[0][0]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, userData)
// 	assert.NilError(t, err)

// 	// admin login
// 	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
// 	assert.NilError(t, err)
// 	//assert.Check(t, token != "", "empty token")

// 	// admin invite user
// 	newUser := orgUsers[0][1]

// 	// invite with wrong email format
// 	succ, err := api.InviteUser(sdc, "wrongEmail", testUtils.TestInvitationExpireTime)
// 	assert.Check(t, !succ)
// 	assert.ErrorContains(t, err, "Invalid email format")
// 	// succeed
// 	succ, err = api.InviteUser(sdc, newUser.Email, testUtils.TestInvitationExpireTime)
// 	assert.NilError(t, err)
// 	assert.Equal(t, true, succ)
// 	// already exists invitation
// 	succ, err = api.InviteUser(sdc, newUser.Email, testUtils.TestInvitationExpireTime)
// 	assert.Check(t, !succ)
// 	assert.ErrorContains(t, err, "already exists active invitation")
// 	// already exists user
// 	succ, err = api.InviteUser(sdc, userData.Email, testUtils.TestInvitationExpireTime)
// 	assert.Check(t, !succ)
// 	assert.ErrorContains(t, err, "already belongs to an org")

// 	// 	admin log out
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)
// }

// func TestListInvitations(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 10)

// 	orgData := orgs[0]
// 	adminData := orgUsers[0][0]
// 	users := orgUsers[0]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, adminData)
// 	assert.NilError(t, err)

// 	// admin login
// 	err = api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
// 	assert.NilError(t, err)

// 	// invite users
// 	for i := 1; i < len(users); i++ {
// 		newUser := orgUsers[0][i]
// 		succ, err := api.InviteUser(sdc, newUser.Email, testUtils.TestInvitationExpireTime)
// 		assert.NilError(t, err)
// 		assert.Equal(t, true, succ)
// 	}

// 	// list invitations
// 	invitations, err := api.ListInvitations(sdc)
// 	assert.NilError(t, err)
// 	assert.Check(t, len(invitations)+1 == len(users))

// 	// 	admin log out
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)
// }

// func TestRevokeInvitation(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 10)

// 	orgData := orgs[0]
// 	adminData := orgUsers[0][0]
// 	users := orgUsers[0]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, adminData)
// 	assert.NilError(t, err)

// 	// admin login
// 	err = api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
// 	assert.NilError(t, err)

// 	// invite users
// 	for i := 1; i < len(users); i++ {
// 		newUser := orgUsers[0][i]
// 		succ, err := api.InviteUser(sdc, newUser.Email, testUtils.TestInvitationExpireTime)
// 		assert.NilError(t, err)
// 		assert.Equal(t, true, succ)
// 	}

// 	// list invitations
// 	invitations, err := api.ListInvitations(sdc)
// 	assert.NilError(t, err)
// 	assert.Check(t, len(invitations)+1 == len(users)) // len(invitations) = 9

// 	// revoke one invitation
// 	succ, codeAlreadyUsed, err := api.RevokeInvitation(sdc, invitations[0].Email)
// 	assert.Check(t, succ)
// 	assert.Check(t, !codeAlreadyUsed)
// 	assert.NilError(t, err)

// 	// list invitations
// 	invitations, err = api.ListInvitations(sdc)
// 	assert.NilError(t, err)
// 	assert.Check(t, len(invitations)+2 == len(users)) // len(invitations) = 8

// 	// cannot revoke alreadyUsed invitation
// 	newUser := orgUsers[0][2]
// 	_, _, succ, err = api.RegisterUser(sdc, testUtils.TestInvitationCode, adminData.OrgID, newUser.Name, newUser.Password, newUser.Email)
// 	assert.NilError(t, err)
// 	assert.Check(t, succ)
// 	succ, codeAlreadyUsed, err = api.RevokeInvitation(sdc, newUser.Email)
// 	assert.Check(t, err, codeAlreadyUsed)
// 	assert.NilError(t, err)
// 	assert.Check(t, !succ)

// 	invitations, err = api.ListInvitations(sdc)
// 	assert.NilError(t, err)
// 	assert.Check(t, len(invitations)+3 == len(users)) // len(invitations) = 7

// 	// admin log out
// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)
// }

// func TestRegisterWithInvitation(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 2)

// 	orgData := orgs[0]
// 	admin := orgUsers[0][0]
// 	newUser := orgUsers[0][1]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, admin)
// 	assert.NilError(t, err)

// 	// admin login
// 	err = api.Login(sdc, admin.UserID, admin.Password, orgData.OrgID)
// 	assert.NilError(t, err)

// 	// adminUser invite user
// 	err = api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
// 	assert.NilError(t, err)
// 	//assert.Check(t, token != "")
// 	succ, err := api.InviteUser(sdc, newUser.Email, testUtils.TestInvitationExpireTime)
// 	assert.NilError(t, err)
// 	assert.Check(t, succ, "failed with InvitateUser, succ = false")
// 	_, err = api.Logout(sdc)

// 	// register with wrong invitation code
// 	_, _, succ, err = api.RegisterUser(sdc, "wrongCode", admin.OrgID, newUser.Name, newUser.Password, newUser.Email)
// 	assert.ErrorContains(t, err, " Invitation Code not Match")
// 	assert.Check(t, !succ)
// 	// register with wrong orgID
// 	_, _, succ, err = api.RegisterUser(sdc, testUtils.TestInvitationCode, "wrongOrgID", newUser.Name, newUser.Password, newUser.Email)
// 	assert.ErrorContains(t, err, "No valid Invitation")
// 	assert.Check(t, !succ)
// 	// succeed
// 	_, _, succ, err = api.RegisterUser(sdc, testUtils.TestInvitationCode, admin.OrgID, newUser.Name, newUser.Password, newUser.Email)
// 	assert.NilError(t, err)
// 	assert.Check(t, succ)
// 	// used invitation code
// 	_, _, succ, err = api.RegisterUser(sdc, testUtils.TestInvitationCode, admin.OrgID, newUser.Name, newUser.Password, newUser.Email)
// 	assert.ErrorContains(t, err, "already belongs to an org")

// }

// func TestPromoteDemote(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 2)

// 	orgData := orgs[0]
// 	admin := orgUsers[0][0]
// 	normalUser := orgUsers[0][1]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, admin)
// 	assert.NilError(t, err)

// 	// register a normal user
// 	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
// 	succ, err := api.InviteUser(sdc, normalUser.Email, testUtils.TestInvitationExpireTime)
// 	assert.NilError(t, err)
// 	api.Logout(sdc)
// 	userID, _, succ, err := api.RegisterUser(sdc, testUtils.TestInvitationCode, admin.OrgID, normalUser.Name, normalUser.Password, normalUser.Email)
// 	assert.NilError(t, err)
// 	assert.Check(t, succ)
// 	normalUser.UserID = userID

// 	// admin login
// 	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)

// 	docBytes := []byte("This is a document.")
// 	docID, err := api.UploadDocumentStream(sdc, "testDoc", bytes.NewBuffer(docBytes))
// 	assert.NilError(t, err)

// 	// promote
// 	succ, err = api.PromoteUser(sdc, normalUser.UserID)
// 	assert.NilError(t, err)
// 	assert.Equal(t, succ, true)

// 	// logout first admin
// 	api.Logout(sdc)

// 	// login promoted user
// 	err = api.Login(sdc, normalUser.Email, normalUser.Password, admin.OrgID)
// 	assert.NilError(t, err)

// 	docReader, err := api.DownloadDocumentStream(sdc, docID)
// 	assert.NilError(t, err)

// 	downloadedBytes, err := ioutil.ReadAll(docReader)
// 	assert.Assert(t, bytes.Equal(docBytes, downloadedBytes), "Plaintext doesn't match")

// 	// logout promoted user
// 	api.Logout(sdc)

// 	// login first admin
// 	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)

// 	// demote
// 	succ, err = api.DemoteUser(sdc, normalUser.Email)
// 	assert.NilError(t, err)
// 	assert.Equal(t, succ, true)

// 	// admin logout
// 	api.Logout(sdc)

// 	// login demoted user
// 	api.Login(sdc, normalUser.Email, normalUser.Password, admin.OrgID)

// 	_, err = api.DownloadDocumentStream(sdc, docID)
// 	assert.Assert(t, err != nil)
// }

// func TestChangePassword(t *testing.T) {
// 	sdc, orgs, orgUsers := testUtils.PrevTest(t, 1, 2)

// 	orgData := orgs[0]
// 	admin := orgUsers[0][0]

// 	// register org and admin
// 	err := testUtils.RegisterOrgAndAdmin(sdc, orgData, admin)
// 	assert.NilError(t, err)

// 	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)

// 	newPassword := "This is a password."

// 	err = api.ChangePassword(sdc, "Incorrect Password", newPassword)
// 	assert.Assert(t, err != nil)

// 	err = api.ChangePassword(sdc, admin.Password, newPassword)
// 	assert.NilError(t, err)

// 	_, err = api.Logout(sdc)
// 	assert.NilError(t, err)

// 	err = api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
// 	assert.Assert(t, err != nil)

// 	err = api.Login(sdc, admin.UserID, newPassword, admin.OrgID)
// 	assert.NilError(t, err)

// 	api.Logout(sdc)
// }
