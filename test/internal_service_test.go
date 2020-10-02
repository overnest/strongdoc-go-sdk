package test

import (
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	assert "github.com/stretchr/testify/require"
	"testing"
)

func TestSuperuser(t *testing.T) {
	// register for an organization
	_, err := client.InitStrongDocManager(client.LOCAL, false)
	assert.NoError(t, err)
	orgID, _, err := api.RegisterOrganization(Organization, OrganizationAddr,
		OrganizationEmail, AdminName, AdminPassword, AdminEmail, Source, SourceData)
	assert.NoError(t, err)
	assert.Equal(t, orgID, Organization)
	// test superuser API
	// login
	err = login()
	assert.NoError(t, err)
	// remove organization
	err = hardRemoveOrg(Organization)
	err = logout()
	assert.NoError(t, err)
}