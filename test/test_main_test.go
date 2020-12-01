package test

import (
	"log"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
)

const (
	DefaultConfig = "dev"
)

/**
register orgs and users
*/
func testSetup(numOfOrgs int, numOfUsersPerOrg int) (client.StrongDocClient, []*testOrg, [][]*testUser, []string, error) {
	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	orgs, orgUsers := initData(numOfOrgs, numOfUsersPerOrg)
	var registeredOrgIds []string

	// register for orgs and users
	for i, orgData := range orgs {
		usersInOrg := orgUsers[i]
		// register org and admin
		adminData := usersInOrg[0]
		if err := registerOrgAndAdmin(sdc, orgData, adminData); err != nil {
			return nil, nil, nil, nil, err
		}
		registeredOrgIds = append(registeredOrgIds, orgData.OrgID)
		if numOfUsersPerOrg <= 1 {
			continue
		}
		// login as admin of this org
		_, err := api.Login(sdc, adminData.UserID, adminData.Password, orgData.OrgID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		// register normal users
		for j := 1; j < len(usersInOrg); j++ {
			// invite new user
			userData := usersInOrg[j]
			_, err = api.InviteUser(sdc, userData.Email, TestInvitationExpireTime)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			// new user register with invitation
			newUserID, newUserOrgID, _, err := api.RegisterUser(sdc, TestInvitationCode, orgData.OrgID, userData.Name, userData.Password, userData.Email)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			userData.UserID = newUserID
			userData.OrgID = newUserOrgID
		}
		// logout
		_, err = api.Logout(sdc)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	return sdc, orgs, orgUsers, registeredOrgIds, err
}

// hard remove registered org
func testTeardown(orgid []string) error {
	// hard remove organization
	if err := HardRemoveOrgs(orgid); err != nil {
		log.Println("failed to tear down")
		return err
	}
	return nil
}

// call superuser API, hard remove registeredOrgs
func HardRemoveOrgs(orgids []string) error {
	if err := superUserLogin(); err != nil {
		return err
	}
	for _, orgid := range orgids {
		hardRemoveOrg(orgid)
	}
	return superUserLogout()
}

// control all tests within package, load config before testing
func TestMain(m *testing.M) {
	if err := loadConfig(DefaultConfig); err != nil {
		log.Println("fail to load config file: ", err)
		return
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}
