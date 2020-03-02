package api

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestListUsers(t *testing.T) {
	pass := adminPassword
	orgID := organization
	adminID := adminEmail

	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Failed to register organization: %s", err)
		return
	}

	token, err := Login(adminID, pass, orgID)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	defer func() {
		_, err = RemoveOrganization(token, true)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()

	stevenID, err := RegisterUser(token, "steven", "lion", "steven@crystalgems.com", true)
	if err != nil {
		log.Printf("Failed to register: %s", err)
		return
	}
	pearlID, err := RegisterUser(token, "pearl", "pd", "pearl@crystalgems.com", false)
	if err != nil {
		log.Printf("Failed to register: %s", err)
		return
	}

	users, err := ListUsers(token)
	printUsers(users)

	_, err = RemoveUser(token, stevenID)
	_, err = RemoveUser(token, pearlID)
	users, err = ListUsers(token)
	printUsers(users)
}

func printUsers(users []User) {
	for i, u := range users {
		fmt.Printf("%d | UserName: [%s], UserID: [%s], IsAdmin: [%t]\n--------\n", i, u.UserName, u.UserID, u.IsAdmin)
	}
}

func TestLogout(t *testing.T) {

	pass := adminPassword
	orgID := organization
	adminID := adminEmail


	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Failed to register organization: %s", err)
		return
	}

	token, err := Login(adminID, pass, orgID)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	defer func() {
		_, err = RemoveOrganization(token, true)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()

	status, err := Logout(token)
	fmt.Printf("status: %s", status)
	if err != nil {
		log.Printf("Failed to log out: %s", err)
		return
	}
	assert.Contains(t, status, "You have successfully logged out on")
	_, err = ListDocuments(token)
	assert.NotNil(t, err)
}

