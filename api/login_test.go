package api

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestLogout(t *testing.T) {

	pass := adminPassword
	orgID := organization
	adminID := adminEmail


	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)
	if err != nil {
		log.Printf("Failed to register organization: %s", err)
		return
	}

	token, err := Login(adminID, pass, orgID)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	defer func() {
		_, err = RemoveOrganization(token)
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

