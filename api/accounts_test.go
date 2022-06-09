package api

import (
	"testing"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/stretchr/testify/require"
)

const (
	userName     = "UserName"
	userPassword = "UserPassword"
	userEmail    = "email@user.com"
)

func TestUserLogin(t *testing.T) {
	assert := require.New(t)

	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NoError(err, FromError(err))

	_, succ, err := RegisterUser(sdc, userName, userPassword, userEmail)
	assert.NoError(err, FromError(err))
	assert.True(succ)

	err = sdc.Login(userEmail, userPassword)
	assert.NoError(err, FromError(err))
}
