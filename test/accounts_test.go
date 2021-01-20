package test

import (
	"fmt"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"gotest.tools/assert"
)

const (
	TestInvitationCode       = "abcdef" //hard-coded on server side for testing
	TestInvitationExpireTime = 10
	TestSource               = "Test Active"
	TestSourceData           = ""
)

type testUser struct {
	// user specified
	Name           string
	Email          string
	Password       string
	PasswordKeyPwd string

	// returned from server
	OrgID  string
	UserID string
}

type testOrg struct {
	// user specified
	Name       string
	Email      string
	Address    string
	Source     string
	SourceData string

	// returned from server
	OrgID string
}

// initialize test data
func initData(numOfOrgs int, numOfUsersPerOrg int) ([]*testOrg, [][]*testUser) {
	orgs := make([]*testOrg, numOfOrgs)
	orgUsers := make([][]*testUser, numOfOrgs*numOfUsersPerOrg)

	for i := 0; i < numOfOrgs; i++ {
		org := &testOrg{}
		org.Name = fmt.Sprintf("testOrgName_%v", i+1)
		org.Email = fmt.Sprintf("testOrgName_%v@example.com", i+1)
		org.Address = fmt.Sprintf("testOrgAddress_%v", i+1)
		org.Source = TestSource
		org.SourceData = TestSourceData
		orgs[i] = org
		usersInOrg := make([]*testUser, numOfUsersPerOrg)
		for j := 0; j < numOfUsersPerOrg; j++ {
			user := &testUser{}
			user.Name = fmt.Sprintf("testUserName_org%v_user%v", i+1, j+1)
			user.Email = fmt.Sprintf("testUserEmail_org%v_user%v@ss.com", i+1, j+1)
			user.PasswordKeyPwd = fmt.Sprintf("testUserPass_org%v_user%v", i+1, j+1)
			user.Password = fmt.Sprintf("testUserPass_org%v_user%v", i+1, j+1)
			usersInOrg[j] = user
		}
		orgUsers[i] = usersInOrg
	}
	return orgs, orgUsers
}

// todo: registrationOrg is not supported in sdk, remove later
// register for an org and admin
func registerOrgAndAdmin(sdc client.StrongDocClient, orgData *testOrg, userData *testUser) error {
	orgID, userID, err := api.RegisterOrganization(sdc, orgData.Name, orgData.Address,
		orgData.Email, userData.Name, userData.Password, userData.Email, orgData.Source, orgData.SourceData)
	if err != nil {
		return err
	}
	if orgID != orgData.Name {
		return fmt.Errorf("return wrong orgID")
	}
	if userID == "" {
		return fmt.Errorf("return wrong userID")
	}
	orgData.OrgID = orgID
	userData.UserID = userID
	userData.OrgID = orgID
	return nil
}

func TestLogin(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org and admin
	orgs, orgUsers := initData(1, 1)
	orgData := orgs[0]
	userData := orgUsers[0][0]
	err = registerOrgAndAdmin(sdc, orgData, userData)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{orgData.OrgID})

	// login with wrong password
	err = api.Login(sdc, userData.UserID, "wrongPassword", orgData.OrgID)
	//assert.ErrorContains(t, err, "Password does not match")
	assert.Assert(t, err != nil)

	// login with wrong userID
	err = api.Login(sdc, "wrongUserID", userData.Password, orgData.OrgID)
	//assert.ErrorContains(t, err, "Not a valid form")
	assert.Assert(t, err != nil)

	// login succeed
	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	//assert.Check(t, token != "", "empty token")
	_, err = api.Logout(sdc)
	assert.NilError(t, err)

	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	//assert.Check(t, token != "", "empty token")
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
}

func TestBusyLogin(t *testing.T) {
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	orgs, orgUsers := initData(1, 1)
	orgData := orgs[0]
	userData := orgUsers[0][0]
	err = registerOrgAndAdmin(sdc, orgData, userData)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{orgData.OrgID})

	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
	// duplicate logout
	_, err = api.Logout(sdc)
	assert.ErrorContains(t, err, " The JWT user is logged out")
	// do something needs login
	_, err = api.ListInvitations(sdc)
	assert.ErrorContains(t, err, " The JWT user is logged out")
}

func TestInviteUser(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org and admin
	orgs, orgUsers := initData(1, 2)
	orgData := orgs[0]
	userData := orgUsers[0][0]
	err = registerOrgAndAdmin(sdc, orgData, userData)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{orgData.OrgID})

	// admin login
	err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	//assert.Check(t, token != "", "empty token")

	// admin invite user
	newUser := orgUsers[0][1]

	// invite with wrong email format
	succ, err := api.InviteUser(sdc, "wrongEmail", TestInvitationExpireTime)
	assert.Check(t, !succ)
	assert.ErrorContains(t, err, "Invalid email format")
	// succeed
	succ, err = api.InviteUser(sdc, newUser.Email, TestInvitationExpireTime)
	assert.NilError(t, err)
	assert.Equal(t, true, succ)
	// already exists invitation
	succ, err = api.InviteUser(sdc, newUser.Email, TestInvitationExpireTime)
	assert.Check(t, !succ)
	assert.ErrorContains(t, err, "already exists active invitation")
	// already exists user
	succ, err = api.InviteUser(sdc, userData.Email, TestInvitationExpireTime)
	assert.Check(t, !succ)
	assert.ErrorContains(t, err, "already belongs to an org")

	// 	admin log out
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
}

func TestListInvitations(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org and admin
	nUsers := 10
	orgs, orgUsers := initData(1, nUsers)
	orgData := orgs[0]
	adminData := orgUsers[0][0]
	err = registerOrgAndAdmin(sdc, orgData, adminData)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{orgData.OrgID})

	// admin login
	err = api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
	assert.NilError(t, err)
	//assert.Check(t, token != "", "empty token")

	// invite users
	for i := 1; i < nUsers; i++ {
		newUser := orgUsers[0][i]
		succ, err := api.InviteUser(sdc, newUser.Email, TestInvitationExpireTime)
		assert.NilError(t, err)
		assert.Equal(t, true, succ)
	}

	// list invitations
	invitations, err := api.ListInvitations(sdc)
	assert.NilError(t, err)
	assert.Check(t, len(invitations)+1 == nUsers)

	// 	admin log out
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
}

func TestRevokeInvitation(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org and admin
	nUsers := 10
	orgs, orgUsers := initData(1, nUsers)
	orgData := orgs[0]
	adminData := orgUsers[0][0]
	err = registerOrgAndAdmin(sdc, orgData, adminData)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{orgData.OrgID})

	// admin login
	err = api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
	assert.NilError(t, err)
	//assert.Check(t, token != "", "empty token")

	// invite users
	for i := 1; i < nUsers; i++ {
		newUser := orgUsers[0][i]
		succ, err := api.InviteUser(sdc, newUser.Email, TestInvitationExpireTime)
		assert.NilError(t, err)
		assert.Equal(t, true, succ)
	}

	// list invitations
	invitations, err := api.ListInvitations(sdc)
	assert.NilError(t, err)
	assert.Check(t, len(invitations)+1 == nUsers) // len(invitations) = 9

	// revoke one invitation
	succ, codeAlreadyUsed, err := api.RevokeInvitation(sdc, invitations[0].Email)
	assert.Check(t, succ)
	assert.Check(t, !codeAlreadyUsed)
	assert.NilError(t, err)

	// list invitations
	invitations, err = api.ListInvitations(sdc)
	assert.NilError(t, err)
	assert.Check(t, len(invitations)+2 == nUsers) // len(invitations) = 8

	// cannot revoke alreadyUsed invitation
	newUser := orgUsers[0][2]
	_, _, succ, err = api.RegisterUser(sdc, TestInvitationCode, adminData.OrgID, newUser.Name, newUser.Password, newUser.Email)
	assert.NilError(t, err)
	assert.Check(t, succ)
	succ, codeAlreadyUsed, err = api.RevokeInvitation(sdc, newUser.Email)
	assert.Check(t, err, codeAlreadyUsed)
	assert.NilError(t, err)
	assert.Check(t, !succ)

	invitations, err = api.ListInvitations(sdc)
	assert.NilError(t, err)
	assert.Check(t, len(invitations)+3 == nUsers) // len(invitations) = 7

	// admin log out
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
}

func TestRegisterWithInvitation(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org and admin
	orgs, orgUsers := initData(1, 2)
	org := orgs[0]
	admin := orgUsers[0][0]
	newUser := orgUsers[0][1]
	err = registerOrgAndAdmin(sdc, org, admin)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{org.OrgID})

	// adminUser invite user
	err = api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
	assert.NilError(t, err)
	//assert.Check(t, token != "")
	succ, err := api.InviteUser(sdc, newUser.Email, TestInvitationExpireTime)
	assert.NilError(t, err)
	assert.Check(t, succ, "failed with InvitateUser, succ = false")
	_, err = api.Logout(sdc)

	// register with wrong invitation code
	_, _, succ, err = api.RegisterUser(sdc, "wrongCode", admin.OrgID, newUser.Name, newUser.Password, newUser.Email)
	assert.ErrorContains(t, err, " Invitation Code not Match")
	assert.Check(t, !succ)
	// register with wrong orgID
	_, _, succ, err = api.RegisterUser(sdc, TestInvitationCode, "wrongOrgID", newUser.Name, newUser.Password, newUser.Email)
	assert.ErrorContains(t, err, "No valid Invitation")
	assert.Check(t, !succ)
	// succeed
	_, _, succ, err = api.RegisterUser(sdc, TestInvitationCode, admin.OrgID, newUser.Name, newUser.Password, newUser.Email)
	assert.NilError(t, err)
	assert.Check(t, succ)
	// used invitation code
	_, _, succ, err = api.RegisterUser(sdc, TestInvitationCode, admin.OrgID, newUser.Name, newUser.Password, newUser.Email)
	assert.ErrorContains(t, err, "already belongs to an org")

}

/*func TestPromoteDemote(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org, admin and a normal user
	orgs, orgUsers := initData(1, 2)
	org := orgs[0]
	admin := orgUsers[0][0]
	normalUser := orgUsers[0][1]
	err = registerOrgAndAdmin(sdc, org, admin)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{org.OrgID})
	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
	succ, err := api.InviteUser(sdc, normalUser.Email, TestInvitationExpireTime)
	assert.NilError(t, err)
	api.Logout(sdc)
	userID, _, succ, err := api.RegisterUser(sdc, TestInvitationCode, admin.OrgID, normalUser.Name, normalUser.Password, normalUser.Email)
	assert.NilError(t, err)
	assert.Check(t, succ)
	normalUser.UserID = userID

	// admin login
	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)

	// promote
	succ, err = api.PromoteUser(sdc, normalUser.UserID)
	assert.NilError(t, err)
	assert.Equal(t, succ, true)
	// demote
	succ, err = api.DemoteUser(sdc, normalUser.Email)
	assert.Equal(t, succ, true)
	assert.NilError(t, err)
	// admin logout
	api.Logout(sdc)
}*/

func TestChangePassword(t *testing.T) {
	// init client
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)

	// register org, admin and a normal user
	orgs, orgUsers := initData(1, 1)
	org := orgs[0]
	admin := orgUsers[0][0]
	err = registerOrgAndAdmin(sdc, org, admin)
	assert.NilError(t, err)
	defer HardRemoveOrgs([]string{org.OrgID})
	api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)

	newPassword := "This is a password."

	err = api.ChangePassword(sdc, "Incorrect Password", newPassword)
	assert.Assert(t, err != nil)

	err = api.ChangePassword(sdc, admin.Password, newPassword)
	assert.NilError(t, err)

	_, err = api.Logout(sdc)
	assert.NilError(t, err)

	err = api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
	assert.Assert(t, err != nil)

	err = api.Login(sdc, admin.UserID, newPassword, admin.OrgID)
	assert.NilError(t, err)

	api.Logout(sdc)
}
