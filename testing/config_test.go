package testing

import (
	"fmt"
	"github.com/overnest/strongdoc-go/api"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestGetConfig(t *testing.T) {

	_, _, err := api.RegisterOrganization(api.Organization, "", api.AdminName,
		api.AdminPassword, api.AdminEmail)
	if err != nil {
		log.Printf("Failed to register organization: %s", err)
		return
	}
	token, err := api.Login(api.AdminEmail, api.AdminPassword, api.Organization)
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

	config, err := api.GetConfiguration(token)
	fmt.Printf("config: [%s]", config)
	assert.NotEmpty(t, config)

}
