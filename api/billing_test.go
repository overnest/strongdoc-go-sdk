package api

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

func TestGetBillingDetails(t *testing.T) {

	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail, testSource, testSourceData)

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
		_, err = RemoveOrganization(token, true)
		if err != nil {
			log.Printf("Failed to log in: %s", err)
			return
		}
	}()
	
	billingDetails := Billing(token)
	tr, err := billingDetails.Traffic()
	assert.Nil(t, err)
	fmt.Println(tr.Cost())

}
