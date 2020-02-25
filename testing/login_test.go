package testing

import (
	"fmt"
	"github.com/overnest/strongdoc-go/api"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestLogout(t *testing.T) {

	pass := adminPassword
	orgID := organization
	adminID := adminEmail


	_, _, err := api.RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)
	if err != nil {
		log.Printf("Failed to register organization: %s", err)
		return
	}

	token, err := api.Login(adminID, pass, orgID)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}

	defer func() {
		_, err = api.RemoveOrganization(token)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()

	status, err := api.Logout(token)
	fmt.Printf("status: %s", status)
	if err != nil {
		log.Printf("Failed to log out: %s", err)
		return
	}
	assert.Contains(t, status, "You have successfully logged out on")
	_, err = api.ListDocuments(token)
	assert.NotNil(t, err)

}

