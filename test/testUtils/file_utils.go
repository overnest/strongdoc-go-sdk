package testUtils

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"io"
	"os"
	"time"
)

func OpenOffsetIdxWriter(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	return openIdxWriter(testLocal, testClient, docID, docVer, utils.OffsetIndex)
}

func OpenTermIdxWriter(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	return openIdxWriter(testLocal, testClient, docID, docVer, utils.TermIndex)
}

// create path? create file
func OpenSearchIdxWriter(testLocal bool, testClient client.StrongDocClient, ownerID string, term string, updateID string) {
	if testLocal {
		dir := fmt.Sprintf("/tmp/search/%v/sidx/%v/%v", ownerID, term, updateID)
		utils.CreateLocalDir(dir)
	} else {

	}
}

func OpenSearchIdxOffsetWriter(testLocal bool, testClient client.StrongDocClient, ownerType utils.OwnerType, ownerID string, term string) (io.WriteCloser, error) {
	if testLocal {
		updateID := newUpdateIDV1()
		path := fmt.Sprintf("/tmp/search/%v/sidx/%v/%v/offset", ownerID, term, updateID)
		return utils.CreateLocalFile(path)
	} else {
		return api.NewSearchIdxOffsetWriter(testClient, ownerType, term)
	}
}

func newUpdateIDV1() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func openIdxWriter(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64, indexType utils.DocIndexType) (outputWriter io.WriteCloser, err error) {
	if testLocal {
		outputWriter, err = utils.CreateLocalFile(buildFilename(docID, docVer, indexType))
	} else {
		outputWriter, err = api.NewDocIndexWriter(testClient, docID, docVer, indexType)
	}
	return
}

func OpenOffsetIdxReader(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	return openIdxReader(testLocal, testClient, docID, docVer, utils.OffsetIndex)
}

func OpenTermIdxReader(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	return openIdxReader(testLocal, testClient, docID, docVer, utils.TermIndex)
}

func openIdxReader(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64, indexType utils.DocIndexType) (reader io.ReadCloser, err error) {
	if testLocal {
		reader, err = utils.OpenLocalFile(buildFilename(docID, docVer, indexType))
	} else {
		reader, err = api.NewDocIndexReader(testClient, docID, docVer, indexType)
	}
	return
}

func RemoveOffsetIndexFile(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) {
	RemoveIndexFile(testLocal, testClient, docID, docVer, utils.OffsetIndex)
}

func RemoveTermIndexFile(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) {
	RemoveIndexFile(testLocal, testClient, docID, docVer, utils.TermIndex)
}

func RemoveDocIndexes(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64) {
	RemoveOffsetIndexFile(testLocal, testClient, docID, docVer)
	RemoveTermIndexFile(testLocal, testClient, docID, docVer)
}

func buildFilename(docID string, docVer uint64, indexType utils.DocIndexType) string {
	return fmt.Sprintf("/tmp/%v_%v_%v", docID, docVer, indexType)
}

func RemoveIndexFile(testLocal bool, testClient client.StrongDocClient, docID string, docVer uint64, indexType utils.DocIndexType) {
	if testLocal {
		os.Remove(buildFilename(docID, docVer, indexType))
	} else {
		api.DeleteDocIndex(testClient, docID, docVer, indexType)
	}
}

func GetFileSize(testLocal bool, reader io.ReadCloser) (filesize uint64, err error) {
	if testLocal {
		file, ok := reader.(*os.File)
		if !ok {
			err = fmt.Errorf("cannot convert reader to file")
			return
		}
		var fileInfo os.FileInfo
		fileInfo, err = file.Stat()
		if err != nil {
			return
		}
		filesize = uint64(fileInfo.Size())
	} else {
		file, ok := reader.(api.FileReader)
		if !ok {
			err = fmt.Errorf("cannot convert reader to fileReader")
			return
		}
		filesize, err = file.GetFileSize()
	}
	return
}
