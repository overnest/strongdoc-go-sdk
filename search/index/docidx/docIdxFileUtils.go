package docidx

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"io"
	"os"
)

// return /tmp/<docID>_<docVer>
func buildDocIdxBasePath(docID string, docVer uint64) string {
	return fmt.Sprintf("/tmp/%v_%v", docID, docVer)
}

// return /tmp/<docID>_<docVer>_offsetIdx
func buildDocOffsetIdxFilename(docID string, docVer uint64) string {
	return fmt.Sprintf("%v_offsetIdx", buildDocIdxBasePath(docID, docVer))
}

// return /tmp/<docID>_<docVer>_termIdx
func buildDocTermIdxFilename(docID string, docVer uint64) string {
	return fmt.Sprintf("%v_termIdx", buildDocIdxBasePath(docID, docVer))
}

func openDocOffsetIdxWriter(sdc client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if utils.TestLocal {
		outputWriter, err = utils.CreateLocalFile(buildDocOffsetIdxFilename(docID, docVer))
	} else {
		outputWriter, err = api.NewDocOffsetIdxWriter(sdc, docID, docVer)
	}
	return
}

func openDocTermIdxWriter(sdc client.StrongDocClient, docID string, docVer uint64) (outputWriter io.WriteCloser, err error) {
	if utils.TestLocal {
		outputWriter, err = utils.CreateLocalFile(buildDocTermIdxFilename(docID, docVer))
	} else {
		outputWriter, err = api.NewDocTermIdxWriter(sdc, docID, docVer)
	}
	return
}

func openDocOffsetIdxReader(sdc client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if utils.TestLocal {
		reader, err = utils.OpenLocalFile(buildDocOffsetIdxFilename(docID, docVer))
	} else {
		reader, err = api.NewDocOffsetIdxReader(sdc, docID, docVer)
	}
	return
}

func openDocTermIdxReader(sdc client.StrongDocClient, docID string, docVer uint64) (reader io.ReadCloser, err error) {
	if utils.TestLocal {
		reader, err = utils.OpenLocalFile(buildDocTermIdxFilename(docID, docVer))
	} else {
		reader, err = api.NewDocTermIdxReader(sdc, docID, docVer)
	}
	return
}

func removeDocOffsetIndex(sdc client.StrongDocClient, docID string, docVer uint64) error {
	if utils.TestLocal {
		return os.Remove(buildDocOffsetIdxFilename(docID, docVer))
	} else {
		return api.RemoveDocOffsetIndex(sdc, docID, docVer)
	}
}

func removeDocTermIndex(sdc client.StrongDocClient, docID string, docVer uint64) error {
	if utils.TestLocal {
		return os.Remove(buildDocTermIdxFilename(docID, docVer))
	} else {
		return api.RemoveDocTermIndex(sdc, docID, docVer)
	}
}

func RemoveDocIndexes(sdc client.StrongDocClient, docID string, docVer uint64) error {
	if utils.TestLocal {
		return os.RemoveAll(buildDocIdxBasePath(docID, docVer))
	} else {
		return api.RemoveDocIndexes(sdc, docID, docVer)
	}

}

func getDocTermIndexSize(sdc client.StrongDocClient, docID string, docVer uint64) (uint64, error) {
	if utils.TestLocal {
		path := buildDocTermIdxFilename(docID, docVer)
		return utils.GetLocalFileSize(path)
	} else {
		return api.GetDocTermIndexSize(sdc, docID, docVer)
	}
}
