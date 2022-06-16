package api

import (
	"fmt"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/sberr"
	"github.com/stretchr/testify/require"
)

const (
	userName     = "UserName"
	userPassword = "UserPassword"
	userEmail    = "email@user.com"
)

func TestUserLoginLogout(t *testing.T) {
	assert := require.New(t)

	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NoError(err, sberr.FromError(err))

	_, succ, err := RegisterUser(sdc, userName, userPassword, userEmail)
	_ = succ
	// assert.NoError(err, FromError(err))
	// assert.True(succ)

	err = sdc.Login(userEmail, userPassword)
	assert.NoError(err, sberr.FromError(err))

	err = sdc.Login(userEmail, "badpassword")
	assert.Error(err)

	status, err := sdc.Logout()
	assert.NoError(err, sberr.FromError(err))
	fmt.Println(status)

	_, err = sdc.Logout()
	assert.Error(err)
}
