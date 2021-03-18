package testUtils

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"gotest.tools/assert"
	"math/rand"
	"strings"
	"testing"
)

/**
register orgs and users
*/

const (
	TestInvitationCode       = "abcdef" //hard-coded on server side for testing
	TestInvitationExpireTime = 10
	TestSource               = "Test Active"
	TestSourceData           = ""
)

type TestUser struct {
	// user specified
	Name           string
	Email          string
	Password       string
	PasswordKeyPwd string

	// returned from server
	OrgID  string
	UserID string
}

type TestOrg struct {
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
func initData(numOfOrgs int, numOfUsersPerOrg int) ([]*TestOrg, [][]*TestUser) {
	orgs := make([]*TestOrg, numOfOrgs)
	orgUsers := make([][]*TestUser, numOfOrgs*numOfUsersPerOrg)

	for i := 0; i < numOfOrgs; i++ {
		org := &TestOrg{}
		org.Name = fmt.Sprintf("testOrgName_%v", i+1)
		org.Email = fmt.Sprintf("testOrgName_%v@example.com", i+1)
		org.Address = fmt.Sprintf("testOrgAddress_%v", i+1)
		org.Source = TestSource
		org.SourceData = TestSourceData
		orgs[i] = org
		usersInOrg := make([]*TestUser, numOfUsersPerOrg)
		for j := 0; j < numOfUsersPerOrg; j++ {
			user := &TestUser{}
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
func RegisterOrgAndAdmin(sdc client.StrongDocClient, orgData *TestOrg, userData *TestUser) error {
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

func DoRegistration(t *testing.T, sdc client.StrongDocClient, orgs []*TestOrg, orgUsers [][]*TestUser) {
	// register for orgs and users
	for i, orgData := range orgs {
		usersInOrg := orgUsers[i]
		// register org and admin
		adminData := usersInOrg[0]
		err := RegisterOrgAndAdmin(sdc, orgData, adminData)
		assert.NilError(t, err)
		if len(usersInOrg) == 1 {
			continue
		}
		// login as admin of this org
		err = api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
		assert.NilError(t, err)
		// register normal users
		for j := 1; j < len(usersInOrg); j++ {
			// invite new user
			userData := usersInOrg[j]
			_, err = api.InviteUser(sdc, userData.Email, TestInvitationExpireTime)
			assert.NilError(t, err)
			// new user register with invitation
			var newUserID, newUserOrgID string
			newUserID, newUserOrgID, _, err = api.RegisterUser(sdc, TestInvitationCode, orgData.OrgID, userData.Name, userData.Password, userData.Email)
			assert.NilError(t, err)
			userData.UserID = newUserID
			userData.OrgID = newUserOrgID
		}
		// logout
		_, err = api.Logout(sdc)
		assert.NilError(t, err)
	}
}

func PrevTest(t *testing.T, numOfOrgs int, numOfUsersPerOrg int) (sdc client.StrongDocClient, orgs []*TestOrg, orgUsers [][]*TestUser) {
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NilError(t, err)
	orgs, orgUsers = initData(numOfOrgs, numOfUsersPerOrg)
	hardRemoveOrgs(orgs)
	return
}

// call superuser API, hard remove registered orgs
func hardRemoveOrgs(orgs []*TestOrg) error {
	err := superUserLogin()
	if err != nil {
		return err
	}
	for _, org := range orgs {
		err = hardRemoveOrg(org.Name)
		if err != nil {
			if !strings.Contains(err.Error(), "Can not find") {
				return err

			}
		}
	}
	err = superUserLogout()
	return err
}

// generate n bytes data
func GenerateRandomData(n int) []byte {
	data := make([]byte, n)
	rand.Read(data)
	return data
}
