package api

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func _TestGetBillingDetails(t *testing.T) {

	_, _, err := RegisterOrganization(organization, "", adminName,
		adminPassword, adminEmail)
	if err != nil {
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
	
	billingDetails := Billing(token)
	tr, err := billingDetails.Traffic()
	assert.NotNil(t, err)
	fmt.Println(tr.Cost())

}