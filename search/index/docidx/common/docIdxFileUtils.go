package common

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"io"
	"os"
)

// return /tmp/index/<docID>
func buildDocBasePath(docID string) string {
	return fmt.Sprintf("/tmp/index/%v", docID)
}

// return /tmp/index/<docID>/<docVer>
func buildDocIdxPath(docID string, docVer uint64) string {
	return fmt.Sprintf("%v/%v", buildDocBasePath(docID), docVer)
}

// return /tmp/index/<docID>/<docVer>/offsetIdx
func buildDocOffsetIdxPath(docID string, docVer uint64) string {
	return fmt.Sprintf("%v/offsetIdx", buildDocIdxPath(docID, docVer))
}

// return /tmp/index/<docID>/<docVer>/termIdx
func buildDocTermIdxPath(docID string, docVer uint64) string {
	return fmt.Sprintf("%v/termIdx", buildDocIdxPath(docID, docVer))
}

func OpenDocOffsetIdxWriter(sdc client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if utils.TestLocal {
		outputWriter, err = utils.MakeDirAndCreateFile(buildDocOffsetIdxPath(docID, docVer))
	} else {
		outputWriter, err = api.NewDocOffsetIdxWriter(sdc, docID, docVer)
	}
	return
}

func OpenDocTermIdxWriter(sdc client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if utils.TestLocal {
		outputWriter, err = utils.MakeDirAndCreateFile(buildDocTermIdxPath(docID, docVer))
	} else {
		outputWriter, err = api.NewDocTermIdxWriter(sdc, docID, docVer)
	}
	return
}

func OpenDocOffsetIdxReader(sdc client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if utils.TestLocal {
		reader, err = utils.OpenLocalFile(buildDocOffsetIdxPath(docID, docVer))
	} else {
		reader, err = api.NewDocOffsetIdxReader(sdc, docID, docVer)
	}
	return
}

func OpenDocTermIdxReader(sdc client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if utils.TestLocal {
		reader, err = utils.OpenLocalFile(buildDocTermIdxPath(docID, docVer))
	} else {
		reader, err = api.NewDocTermIdxReader(sdc, docID, docVer)
	}
	return
}

// remove all version of doc indexes
func RemoveDocIndexes(sdc client.StrongDocClient, docID string) error {
	if utils.TestLocal {
		return os.RemoveAll(buildDocBasePath(docID))
	} else {
		return api.RemoveDocIndexesAllVersions(sdc, docID)
	}
}

func GetDocTermIndexSize(sdc client.StrongDocClient, docID string, docVer uint64) (uint64, error) {
	if utils.TestLocal {
		path := buildDocTermIdxPath(docID, docVer)
		return utils.GetLocalFileSize(path)
	} else {
		return api.GetDocTermIndexSize(sdc, docID, docVer)
	}
}
