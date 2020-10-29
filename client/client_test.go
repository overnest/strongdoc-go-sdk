package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	sdc, err := GetStrongDocClient()
	assert.Error(t, err)
	assert.Nil(t, sdc)

	sdc, err = InitStrongDocClient(unset, false)
	assert.Error(t, err)
	assert.Nil(t, sdc)

	sdc, err = InitStrongDocClient(DEFAULT, false)
	assert.NoError(t, err)
	assert.NotNil(t, sdc)

	sdc, err = InitStrongDocClient(LOCAL, false)
	assert.Error(t, err)
	assert.Nil(t, sdc)

	sdc, err = InitStrongDocClient(LOCAL, true)
	assert.NoError(t, err)
	assert.NotNil(t, sdc)

	assert.Nil(t, sdc.GetAuthConn())
	assert.NotNil(t, sdc.GetNoAuthConn())

	getSdc, err := GetStrongDocClient()
	assert.NoError(t, err)
	assert.Equal(t, sdc, getSdc)

	assert.NotNil(t, sdc.GetGrpcClient())
	sdgc, err := GetStrongDocGrpcClient()
	assert.NoError(t, err)
	assert.Equal(t, sdgc, sdc.GetGrpcClient())
}
