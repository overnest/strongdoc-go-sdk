package common

import (
	"fmt"
)

// The production document index and version will be stored in S3 at the following location:
//   <bucket>/<orgID>/docs/<docID>/<ver>/idx/offsets
//   <bucket>/<orgID>/docs/<docID>/<ver>/idx/terms
//   <bucket>/<orgID>/docs/<docID>/<ver>/idx/start_time
//   <bucket>/<orgID>/docs/<docID>/<ver>/idx/end_time
// The production document is stored at the same location as the document index at:
//   <bucket>/<orgID>/docs/<docID>/<ver>/doc

const (
	LOCAL_DOC_IDX_BASE = "/tmp/strongdoc/docidx"
)

var (
	localDocIdx = false
)

// return TEST_DOC_IDX_BASE/<docID>
func buildDocIdxBasePath(docID string) string {
	return fmt.Sprintf("/%v/%v", LOCAL_DOC_IDX_BASE, docID)
}

// return TEST_DOC_IDX_BASE/<docID>/<docVer>
func buildDocIdxPath(docID string, docVer uint64) string {
	return fmt.Sprintf("%v/%v", buildDocIdxBasePath(docID), docVer)
}

// return TEST_DOC_IDX_BASE/<docID>/<docVer>/offsetIdx
func buildDocOffsetIdxPath(docID string, docVer uint64) string {
	return fmt.Sprintf("%v/offsetIdx", buildDocIdxPath(docID, docVer))
}

// return TEST_DOC_IDX_BASE/<docID>/<docVer>/termIdx
func buildDocTermIdxPath(docID string, docVer uint64) string {
	return fmt.Sprintf("%v/termIdx", buildDocIdxPath(docID, docVer))
}

func LocalDocIdx() bool {
	return localDocIdx
}

func EnableLocalDocIdx() {
	localDocIdx = true
}

func DisableLocalDocIdx() {
	localDocIdx = false
}
