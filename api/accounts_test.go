package api

import (
	"fmt"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/stretchr/testify/require"
)

func TestUserRegistration(t *testing.T) {
	assert := require.New(t)

	sdc, err := client.InitStrongDocClient(client.LOCAL, false)
	assert.NoError(err)

	name := "UserName"
	password := "UserPasswd"
	email := "email@user.com"
	testString := "test encryption string"

	kdf, asym, term, search, err := NewUserKeys("", password)
	assert.NoError(err)

	kdfCipher, err := kdf.Encrypt([]byte(testString))
	assert.NoError(err)

	asymCipher, err := asym.Encrypt([]byte(testString))
	assert.NoError(err)

	searchCipher, err := search.Encrypt([]byte(testString))
	assert.NoError(err)

	userReq := &User{
		Name:     name,
		Email:    email,
		Password: password,
		UserCred: &UserCred{
			KdfKey:  kdf,
			AsymKey: asym,
		},
		SearchCred: &SearchCred{
			TermKey:   term,
			SearchKey: search,
		},
	}

	user, succ, err := registerUser(sdc, userReq)
	if err != nil {
		gerr := FromError(err)
		fmt.Println(gerr)
	}

	assert.NoError(err, err)
	assert.True(succ)

	kdfPlain, err := user.UserCred.KdfKey.Decrypt(kdfCipher)
	assert.NoError(err)
	assert.Equal(testString, string(kdfPlain))

	asymPlain, err := user.UserCred.AsymKey.Decrypt(asymCipher)
	assert.NoError(err)
	assert.Equal(testString, string(asymPlain))

	searchPlain, err := user.SearchCred.SearchKey.Decrypt(searchCipher)
	assert.NoError(err)
	assert.Equal(testString, string(searchPlain))
}
