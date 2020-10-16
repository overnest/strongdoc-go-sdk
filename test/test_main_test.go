package test

import (
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"log"
	"os"
	"testing"
	"time"
)

const (
	DefaultConfig = "dev"
)

/**
	register orgs and users
*/
func testSetup(numOfOrgs int, numOfUsersPerOrg int) ([]*testOrg, [][]*testUser, []string, error) {
	_, err := client.InitStrongDocManager(client.LOCAL, false)
	if err != nil {
		return nil, nil, nil, err
	}
	orgs, orgUsers := initData(numOfOrgs, numOfUsersPerOrg)
	var registeredOrgIds []string

	// register for orgs and users
	for i, orgData := range orgs {
		usersInOrg := orgUsers[i]
		// register org and admin
		adminData := usersInOrg[0]
		if err := registerOrgAndAdmin(orgData, adminData); err != nil {
			return nil, nil, nil, err
		}
		registeredOrgIds = append(registeredOrgIds, orgData.OrgID)
		if numOfUsersPerOrg <= 1 {
			continue
		}
		// login as admin of this org
		_, err := api.Login(adminData.UserID, adminData.Password, orgData.OrgID, adminData.PasswordKeyPwd)
		if err != nil {
			return nil, nil, nil, err
		}
		// register normal users
		for j := 1; j < len(usersInOrg); j++ {
			// invite new user
			userData := usersInOrg[j]
			_, err = api.InviteUser(userData.Email, TestInvitationExpireTime)
			if err != nil {
				return nil, nil, nil, err
			}
			if err := generateUserClientKeys(userData); err != nil {
				return nil, nil, nil, err
			}
			// new user register with invitation
			newUserID, newUserOrgID, _, err := api.RegisterWithInvitation(TestInvitationCode, orgData.OrgID, userData.Name,
				userData.Password,  userData.Email, userData.serialKdf, userData.pubKey, userData.encPriKey)
			if err != nil {
				return nil, nil, nil, err
			}
			userData.UserID = newUserID
			userData.OrgID = newUserOrgID
		}
		// logout
		_, err = api.Logout()
		if err != nil {
			return nil, nil, nil, err
		}
	}
	time.Sleep(2*time.Second) // todo temporary fix busy login problem
	return orgs, orgUsers, registeredOrgIds, err
}

// hard remove registered org
func testTeardown(orgid []string ) error {
	// hard remove organization
	if err :=  HardRemoveOrgs(orgid); err != nil {
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
	for _, orgid := range(orgids) {
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