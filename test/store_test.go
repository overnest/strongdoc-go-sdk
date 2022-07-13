package test

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/proto"
	"github.com/overnest/strongdoc-go-sdk/sberr"
	"github.com/overnest/strongdoc-go-sdk/store"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/stretchr/testify/require"
)

const (
	storeFileName = "storeTestFile"
	localFileName = "/tmp/storeTestFile"
)

func TestStore(t *testing.T) {
	assert := require.New(t)
	totalFileSize := 1024 * 1024 * 50
	maxWriteSize := 1024 * 1024 * 10
	maxReadSize := 1024 * 1024 * 10

	sdc, _ := testUtils.SetupTestUser(t, 1)

	//
	// Write to store
	//
	writer, err := store.CreateStore(sdc, &proto.StoreInit{
		Content: proto.StoreContent_GENERIC,
		Init: &proto.StoreInit_Generic{
			Generic: &proto.GenericStoreInit{
				Filename: storeFileName,
			},
		},
	})
	assert.NoError(err, sberr.FromError(err))

	localFile, err := os.Create(localFileName)
	assert.NoError(err)
	defer os.Remove(localFileName)

	writeBytesLeft := totalFileSize
	for writeBytesLeft > 0 {
		writeBytes := utils.Min(writeBytesLeft, rand.Intn(maxWriteSize))
		bytes := make([]byte, writeBytes)
		rand.Read(bytes)

		n, err := localFile.Write(bytes)
		assert.NoError(err)

		m, err := writer.Write(bytes)
		assert.NoError(err, sberr.FromError(err))

		assert.Equal(m, n)
		writeBytesLeft -= len(bytes)
	}

	assert.NoError(localFile.Close())
	assert.NoError(writer.Close())

	//
	// Read from store
	//
	localFile, err = os.Open(localFileName)
	assert.NoError(err)
	reader, err := store.OpenStore(sdc, &proto.StoreInit{
		Content: proto.StoreContent_GENERIC,
		Init: &proto.StoreInit_Generic{
			Generic: &proto.GenericStoreInit{
				Filename: storeFileName,
			},
		},
	})
	assert.NoError(err, sberr.FromError(err))

	var lerr, rerr error
	for lerr != io.EOF || rerr != io.EOF {
		readBytes := rand.Intn(maxReadSize)
		lBytes := make([]byte, readBytes)
		rBytes := make([]byte, readBytes)

		var m, n int
		m, lerr = localFile.Read(lBytes)
		assert.True(lerr == nil || lerr == io.EOF)
		n, rerr = reader.Read(rBytes)
		assert.True(rerr == nil || rerr == io.EOF, sberr.FromError(rerr))

		assert.Equal(m, n)
		assert.Equal(lBytes, rBytes)
	}

	fmt.Println("lerr:", lerr, " rerr:", rerr)

	assert.NoError(localFile.Close())
	assert.NoError(reader.Close())
}
