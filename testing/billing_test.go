package testing

import (
	"fmt"
	"github.com/overnest/strongdoc-go/api"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestGetBillingDetails(t *testing.T) {

	//_, _, err := RegisterOrganization(Organization, "", AdminName,
	//	AdminPassword, AdminEmail)
	//if err != nil {
	//	log.Printf("Failed to register organization: %s", err)
	//	return
	//}

	token, err := api.Login(api.AdminEmail, api.AdminPassword, api.Organization)
	if err != nil {
		log.Printf("Failed to log in: %s", err)
		return
	}
	
	billingDetails := api.Billing(token)
	tr, err := billingDetails.Traffic()
	assert.NotNil(t, err)
	fmt.Println(tr.Cost())

}
