package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	sdc, err := GetStrongDocClient()
	assert.Error(t, err)
	assert.Nil(t, sdc)

	sdm, err := InitStrongDocManager(unset, false)
	assert.Error(t, err)
	assert.Nil(t, sdm)

	sdm, err = InitStrongDocManager(DEFAULT, false)
	assert.NoError(t, err)
	assert.NotNil(t, sdm)

	sdm, err = InitStrongDocManager(LOCAL, false)
	assert.Error(t, err)
	assert.Nil(t, sdm)

	sdm, err = InitStrongDocManager(LOCAL, true)
	assert.NoError(t, err)
	assert.NotNil(t, sdm)

	assert.Nil(t, sdm.GetAuthConn())
	assert.NotNil(t, sdm.GetNoAuthConn())

	getSdm, err := GetStrongDocManager()
	assert.NoError(t, err)
	assert.Equal(t, sdm, getSdm)

	assert.NotNil(t, sdm.GetClient())
	sdc, err = GetStrongDocClient()
	assert.NoError(t, err)
	assert.Equal(t, sdc, sdm.GetClient())
}
