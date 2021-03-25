package test

import (
	"bytes"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"gotest.tools/assert"
	"io"
	"testing"
)

func TestFileReader(t *testing.T) {
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	t.Run("test small size data", func(t *testing.T) {
		// login
		user := users[0][0]
		err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
		assert.NilError(t, err)

		// store file with 100 bytes
		filename := "/a/b/c/testRead.txt"
		expctedFileSize := 100
		data := testUtils.GenerateRandomData(expctedFileSize)

		writer, err := api.NewFileWriter(sdc, filename)
		assert.NilError(t, err)
		err = writer.WriteFile(data, 10)
		assert.NilError(t, err, "write file")
		err = writer.Close()
		assert.NilError(t, err)

		// read [0, 30]
		reader, err := api.NewFileReader(sdc, filename)
		bufferSize := 30
		buffer := make([]byte, bufferSize)
		bytesRead, err := reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, bytes.Equal(data[0:bufferSize], buffer))

		// read [5, 35]
		offset, err := reader.Seek(5, utils.SeekSet)
		assert.NilError(t, err)
		bytesRead, err = reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, offset == 5)
		assert.Check(t, bytes.Equal(data[5:5+bufferSize], buffer))

		// read [40, 70]
		offset, err = reader.Seek(5, utils.SeekCur)
		assert.NilError(t, err)
		assert.Check(t, offset == 40)
		bytesRead, err = reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, bytes.Equal(data[40:40+bufferSize], buffer))

		// read [60, 90]
		offset, err = reader.Seek(40, utils.SeekEnd)
		assert.Check(t, offset == 60)
		assert.NilError(t, err)
		bytesRead, err = reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, bytes.Equal(data[60:60+bufferSize], buffer))

		// seek to 30
		offset, err = reader.Seek(30, utils.SeekSet)
		assert.NilError(t, err)
		assert.Check(t, offset == 30)

		// read [30, 60]
		bytesRead, err = reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, bytes.Equal(data[bufferSize:bufferSize*2], buffer))

		// readAt [10, 40]
		bytesRead, err = reader.ReadAt(buffer, 10)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, bytes.Equal(data[10:10+bufferSize], buffer))

		// read [60, 90]
		bytesRead, err = reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == bufferSize)
		assert.Check(t, bytes.Equal(data[bufferSize*2:bufferSize*3], buffer))

		// read [90, 100]
		bytesRead, err = reader.Read(buffer)
		assert.NilError(t, err)
		assert.Check(t, bytesRead == 10)
		assert.Check(t, bytes.Equal(data[bufferSize*3:100], buffer[:10]))

		// EOF
		bytesRead, err = reader.Read(buffer)
		assert.Check(t, err == io.EOF)

		// close
		err = reader.Close()
		assert.NilError(t, err)

		// Delete file
		err = api.RemoveFile(sdc, filename)
		assert.NilError(t, err)
	})
}

func TestFileWriter(t *testing.T) {
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	t.Run("test small size data", func(t *testing.T) {
		// login
		user := users[0][0]
		err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
		assert.NilError(t, err)

		filename := "/a/b/c/testWrite.txt"
		data := testUtils.GenerateRandomData(100)
		writer, err := api.NewFileWriter(sdc, filename)
		assert.NilError(t, err)
		err = writer.WriteFile(data, 10)
		assert.NilError(t, err, "write file")
		err = writer.Close()
		assert.NilError(t, err)

		// overwrite file, fail to create existing file
		writer, err = api.NewFileWriter(sdc, filename)
		assert.Check(t, err != nil)

		reader, err := api.NewFileReader(sdc, filename)
		assert.NilError(t, err)
		res, err := reader.ReadFile(90)
		assert.NilError(t, err)
		assert.Check(t, bytes.Equal(res, data))
		err = reader.Close()
		assert.NilError(t, err)

		// Delete file
		err = api.RemoveFile(sdc, filename)
		assert.NilError(t, err)
	})
}

func TestDocIndex(t *testing.T) {
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	t.Run("test small size doc index data", func(t *testing.T) {
		// login
		user := users[0][0]
		err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
		assert.NilError(t, err)

		data := testUtils.GenerateRandomData(100)
		var docID = "docID"
		var docVer uint64 = 1

		writer, err := api.NewDocOffsetIdxWriter(sdc, docID, docVer)
		assert.NilError(t, err)
		err = writer.WriteFile(data, 10)
		assert.NilError(t, err, "write doc index file")

		err = writer.Close()
		assert.NilError(t, err)

		reader, err := api.NewDocOffsetIdxReader(sdc, docID, docVer)
		assert.NilError(t, err)
		res, err := reader.ReadFile(30)
		assert.NilError(t, err)
		assert.Check(t, bytes.Equal(res, data))

		err = reader.Close()
		assert.NilError(t, err)

		err = api.RemoveDocOffsetIndex(sdc, docID, docVer)
		assert.NilError(t, err)
	})
}

func TestSearchIndex(t *testing.T) {
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	t.Run("test small size doc serach data", func(t *testing.T) {
		// login
		user := users[0][0]
		err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
		assert.NilError(t, err)

		data := testUtils.GenerateRandomData(100)

		var ownerType = utils.OwnerUser
		var term = "term"

		writer, updateID1, err := api.NewSearchTermIdxWriter(sdc, ownerType, term)
		assert.NilError(t, err)
		err = writer.WriteFile(data, 10)
		assert.NilError(t, err, "write doc search index file")

		err = writer.Close()
		assert.NilError(t, err)

		writer, updateID2, err := api.NewSearchTermIdxWriter(sdc, ownerType, term)
		assert.NilError(t, err)
		err = writer.WriteFile(data, 10)
		assert.NilError(t, err, "write doc search index file")

		err = writer.Close()
		assert.NilError(t, err)

		updateIDs, err := api.GetUpdateIDs(sdc, ownerType, term)

		assert.Equal(t, len(updateIDs), 2)
		assert.Equal(t, updateIDs[0], updateID1)
		assert.Equal(t, updateIDs[1], updateID2)
	})
}
