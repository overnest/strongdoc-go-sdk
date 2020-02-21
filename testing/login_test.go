package testing

import (
	"fmt"
	"github.com/overnest/strongdoc-go/api"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestLogout(t *testing.T) {

	//orgID, adminID, err := api.RegisterOrganization(api.Organization, "", api.AdminName,
	//	api.AdminPassword, api.AdminEmail)
	//if err != nil {
	//	log.Printf("Failed to register organization: %s", err)
	//	return
	//}

	token, err := api.Login(api.AdminEmail, api.AdminPassword, api.Organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}
	assert.NotEmpty(t, token)

	status, err := api.Logout(token)
	if err != nil {
		log.Printf("Failed to log out: %s", err)
		return
	}
	assert.Equal(t, status, "success")
	_, err = api.ListDocuments(token)
	assert.NotNil(t, err)

	fmt.Printf("status: %s", status)
}

