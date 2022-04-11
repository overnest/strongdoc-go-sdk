package common

import (
	"io"
	"os"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

func OpenDocOffsetIdxWriter(sdc client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if LocalDocIdx() {
		outputWriter, err = utils.MakeDirAndCreateFile(buildDocOffsetIdxPath(docID, docVer))
	} else {
		outputWriter, err = api.NewDocOffsetIdxWriter(sdc, docID, docVer)
	}
	return
}

func OpenDocTermIdxWriter(sdc client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if LocalDocIdx() {
		outputWriter, err = utils.MakeDirAndCreateFile(buildDocTermIdxPath(docID, docVer))
	} else {
		outputWriter, err = api.NewDocTermIdxWriter(sdc, docID, docVer)
	}
	return
}

func OpenDocOffsetIdxReader(sdc client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if LocalDocIdx() {
		reader, err = utils.OpenLocalFile(buildDocOffsetIdxPath(docID, docVer))
	} else {
		reader, err = api.NewDocOffsetIdxReader(sdc, docID, docVer)
	}
	return
}

func OpenDocTermIdxReader(sdc client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if LocalDocIdx() {
		reader, err = utils.OpenLocalFile(buildDocTermIdxPath(docID, docVer))
	} else {
		reader, err = api.NewDocTermIdxReader(sdc, docID, docVer)
	}
	return
}

// remove all version of doc indexes
func RemoveDocIdxs(sdc client.StrongDocClient, docID string) error {
	if LocalDocIdx() {
		return os.RemoveAll(buildDocIdxBasePath(docID))
	} else {
		return api.RemoveDocIndexesAllVersions(sdc, docID)
	}
}

func GetDocTermIndexSize(sdc client.StrongDocClient, docID string, docVer uint64) (uint64, error) {
	if LocalDocIdx() {
		path := buildDocTermIdxPath(docID, docVer)
		return utils.GetLocalFileSize(path)
	} else {
		return api.GetDocTermIndexSize(sdc, docID, docVer)
	}
}
