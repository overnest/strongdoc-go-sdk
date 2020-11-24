package test

import (
	"fmt"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	cryptoKey "github.com/overnest/strongsalt-crypto-go"
	cryptoKdf "github.com/overnest/strongsalt-crypto-go/kdf"
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

	// client keys
	kdf             *cryptoKdf.StrongSaltKdf
	userPasswordKey *cryptoKey.StrongSaltKey
	userKey         *cryptoKey.StrongSaltKey
	serialKdf       []byte
	pubKey          []byte
	encPriKey       []byte

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

	// client keys
	orgKey    *cryptoKey.StrongSaltKey
	pubKey    []byte
	encPriKey []byte

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

// generate client-side keys for user
func generateUserClientKeys(user *testUser) error {
	// generate user key
	userKdf, err := cryptoKdf.New(cryptoKdf.Type_Pbkdf2, cryptoKey.Type_Secretbox)
	if err != nil {
		return err
	}
	kdfMetaBytes, err := userKdf.Serialize()
	if err != nil {
		return err
	}
	userPasswordKey, err := userKdf.GenerateKey([]byte(user.PasswordKeyPwd))
	if err != nil {
		return err
	}
	userKey, err := cryptoKey.GenerateKey(cryptoKey.Type_X25519)
	if err != nil {
		return err
	}
	userPublicKeyBytes, err := userKey.SerializePublic()
	if err != nil {
		return err
	}
	userFullKeyBytes, err := userKey.Serialize()
	if err != nil {
		return err
	}
	encUserPriKeyBytes, err := userPasswordKey.Encrypt(userFullKeyBytes)
	if err != nil {
		return err
	}

	user.kdf = userKdf
	user.userPasswordKey = userPasswordKey
	user.userKey = userKey
	user.serialKdf = kdfMetaBytes
	user.pubKey = userPublicKeyBytes
	user.encPriKey = encUserPriKeyBytes

	return nil
}

// generate client-side keys for org
func generateOrgAndUserClientKeys(org *testOrg, user *testUser) error {
	generateUserClientKeys(user)
	orgKey, err := cryptoKey.GenerateKey(cryptoKey.Type_X25519)
	if err != nil {
		return err
	}
	orgPublicKeyBytes, err := orgKey.SerializePublic()
	if err != nil {
		return err
	}
	orgFullKeyBytes, err := orgKey.Serialize()
	if err != nil {
		return err
	}
	encOrgPriKeyBytes, err := user.userKey.Encrypt(orgFullKeyBytes)
	if err != nil {
		return err
	}

	org.orgKey = orgKey
	org.pubKey = orgPublicKeyBytes
	org.encPriKey = encOrgPriKeyBytes

	return nil
}

// todo: registrationOrg is not supported in sdk, remove later
// register for an org and admin
func registerOrgAndAdmin(sdc client.StrongDocClient, orgData *testOrg, userData *testUser) error {
	if err := generateOrgAndUserClientKeys(orgData, userData); err != nil {
		return err
	}
	orgID, userID, err := api.RegisterOrganization(sdc, orgData.Name, orgData.Address,
		orgData.Email, userData.Name, userData.Password, userData.Email, orgData.Source, orgData.SourceData,
		userData.serialKdf, userData.pubKey, userData.encPriKey, orgData.pubKey, orgData.encPriKey)
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
	token, err := api.Login(sdc, userData.UserID, "wrongPassword", orgData.OrgID)
	//assert.ErrorContains(t, err, "Password does not match")
	assert.Assert(t, err != nil)

	// login with wrong userID
	token, err = api.Login(sdc, "wrongUserID", userData.Password, orgData.OrgID)
	//assert.ErrorContains(t, err, "Not a valid form")
	assert.Assert(t, err != nil)

	// login succeed
	token, err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	assert.Check(t, token != "", "empty token")
	_, err = api.Logout(sdc)
	assert.NilError(t, err)

	token, err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	assert.Check(t, token != "", "empty token")
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

	_, err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	_, err = api.Logout(sdc)
	assert.NilError(t, err)
	_, err = api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
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
	token, err := api.Login(sdc, userData.UserID, userData.Password, orgData.OrgID)
	assert.NilError(t, err)
	assert.Check(t, token != "", "empty token")

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
	token, err := api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
	assert.NilError(t, err)
	assert.Check(t, token != "", "empty token")

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
	token, err := api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
	assert.NilError(t, err)
	assert.Check(t, token != "", "empty token")

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
	err = generateUserClientKeys(newUser)
	assert.NilError(t, err)
	_, _, succ, err = api.RegisterWithInvitation(sdc, TestInvitationCode, adminData.OrgID, newUser.Name, newUser.Password, newUser.Email, newUser.serialKdf, newUser.pubKey, newUser.encPriKey)
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
	token, err := api.Login(sdc, admin.UserID, admin.Password, admin.OrgID)
	assert.NilError(t, err)
	assert.Check(t, token != "")
	succ, err := api.InviteUser(sdc, newUser.Email, TestInvitationExpireTime)
	assert.NilError(t, err)
	assert.Check(t, succ, "failed with InvitateUser, succ = false")
	_, err = api.Logout(sdc)

	// newUser generate client keys
	err = generateUserClientKeys(newUser)
	assert.NilError(t, err)

	// register with wrong invitation code
	_, _, succ, err = api.RegisterWithInvitation(sdc, "wrongCode", admin.OrgID, newUser.Name, newUser.Password, newUser.Email, newUser.serialKdf, newUser.pubKey, newUser.encPriKey)
	assert.ErrorContains(t, err, " Invitation Code not Match")
	assert.Check(t, !succ)
	// register with wrong orgID
	_, _, succ, err = api.RegisterWithInvitation(sdc, TestInvitationCode, "wrongOrgID", newUser.Name, newUser.Password, newUser.Email, newUser.serialKdf, newUser.pubKey, newUser.encPriKey)
	assert.ErrorContains(t, err, "No valid Invitation")
	assert.Check(t, !succ)
	// succeed
	_, _, succ, err = api.RegisterWithInvitation(sdc, TestInvitationCode, admin.OrgID, newUser.Name, newUser.Password, newUser.Email, newUser.serialKdf, newUser.pubKey, newUser.encPriKey)
	assert.NilError(t, err)
	assert.Check(t, succ)
	// used invitation code
	_, _, succ, err = api.RegisterWithInvitation(sdc, TestInvitationCode, admin.OrgID, newUser.Name, newUser.Password, newUser.Email, newUser.serialKdf, newUser.pubKey, newUser.encPriKey)
	assert.ErrorContains(t, err, "already belongs to an org")

}

func TestPromoteDemote(t *testing.T) {
	// set up and tear down
	sdc, registeredOrgs, registeredOrgUsers, orgids, err := testSetup(1, 2) // 1: number of orgs, 2: number of users per org
	assert.NilError(t, err)
	defer testTeardown(orgids)

	t.Run("promoteUser test", func(t *testing.T) {
		// admin login
		org := registeredOrgs[0]
		admin := registeredOrgUsers[0][0]
		_, err = api.Login(sdc, admin.UserID, admin.Password, org.OrgID)
		user := registeredOrgUsers[0][1]
		// promote
		succ, startOver, err := api.PromoteUser(sdc, user.UserID)
		assert.Equal(t, succ, true)
		assert.Equal(t, startOver, false)
		assert.NilError(t, err)
		// demote
		succ, err = api.DemoteUser(sdc, user.UserID)
		assert.Equal(t, succ, true)
		assert.NilError(t, err)
		// logout
		api.Logout(sdc)
	})
}
