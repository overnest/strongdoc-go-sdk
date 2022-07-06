package test

import (
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api/apidef"
	"github.com/overnest/strongdoc-go-sdk/document"
	"github.com/overnest/strongdoc-go-sdk/sberr"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"github.com/stretchr/testify/require"
)

const (
	docFileName      = "docTestFile"
	docLocalFileName = "/tmp/docTestFile"
)

func TestDocument(t *testing.T) {
	assert := require.New(t)

	totalFileSize := 1024 * 1024 * 50
	maxWriteSize := 1024 * 1024 * 10
	maxReadSize := 1024 * 1024 * 10
	sdc, user := testUtils.SetupTestUser(t, 1)

	//
	// Create document
	//
	docKey, err := apidef.NewSymKey(user.UserID, []*apidef.Key{user.UserCred.AsymKey})
	assert.NoError(err, sberr.FromError(err))

	writer, err := document.CreateDocument(sdc, docFileName, docKey)
	assert.NoError(err, sberr.FromError(err))
	docKey = writer.DocKey()

	localFile, err := os.Create(localFileName)
	assert.NoError(err)
	defer os.Remove(localFileName)

	writeBytesLeft := totalFileSize
	for writeBytesLeft > 0 {
		writeBytes := utils.Min(writeBytesLeft, rand.Intn(maxWriteSize))
		bytes := make([]byte, writeBytes)
		rand.Read(bytes)
		b64bytes := []byte(base64.StdEncoding.EncodeToString(bytes))

		m, err := localFile.Write(b64bytes)
		assert.NoError(err)

		n, err := writer.Write(b64bytes)
		assert.NoError(err, sberr.FromError(err))

		assert.Equal(m, n)
		writeBytesLeft -= len(b64bytes)
	}

	assert.NoError(localFile.Close())
	assert.NoError(writer.Close())

	//
	// Read document
	//
	localFile, err = os.Open(localFileName)
	assert.NoError(err)
	reader, err := document.OpenDocument(sdc, writer.DocID(), docKey)
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
