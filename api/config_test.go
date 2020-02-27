package api

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestGetConfig(t *testing.T) {

	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Failed to register organization: %s", err)
		return
	}
	token, err := Login(adminEmail, adminPassword, organization)
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

	config, err := GetConfiguration(token)
	fmt.Printf("config: [%s]", config)
	assert.NotEmpty(t, config)

}
