package query

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	"github.com/overnest/strongdoc-go-sdk/utils"

	scom "github.com/overnest/strongdoc-go-sdk/search/index/searchidx/common"
	"gotest.tools/assert"
)

func TestQuery(t *testing.T) {
	scom.EnableAllLocal()
	cred := OpenCredentials()
	defer Cleanup(cred)

	testDocs, err := docidx.InitTestDocumentIdx(10, false)
	assert.NilError(t, err)

	uploadDocs := make([]string, len(testDocs))
	for i, testDoc := range testDocs {
		uploadDocs[i] = testDoc.DocFilePath
	}

	docs, err := UploadDocument(cred, uploadDocs)
	assert.NilError(t, err)

	fmt.Println(utils.JsonPrint(docs))

	phrases := []string{
		"almost no restrictions",
		"almost no restrictions doesnotexist",
		"world more ridiculous than",
		"world more doesnotexist ridiculous than",
	}

	for _, phrase := range phrases {
		result, err := Search(cred, phrase)
		assert.NilError(t, err)

		fmt.Printf("\"%v\" hits(%v): %v\n", phrase, len(result), utils.JsonPrint(result))
	}
}

func TestMe(t *testing.T) {
	frame := getTopStackFrame()
	fmt.Println("result:", utils.JsonPrint(frame))

	fmt.Println("---------------")
	test1()
}

func test1() {
	test2()
}

func test2() {
	frame := getTopStackFrame()
	fmt.Println("result:", utils.JsonPrint(frame))

}

func getTopStackFrame() (result runtime.Frame) {
	var pc []uintptr = make([]uintptr, 1000)
	n := runtime.Callers(0, pc)
	pc = pc[:n]

	frames := runtime.CallersFrames(pc)
	var frame runtime.Frame
	var more bool = true

	// Skip runtime
	for more {
		frame, more = frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			result = frame
			break
		}

		fmt.Println(utils.JsonPrint(frame))
	}

	for more {
		frame, more = frames.Next()
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "testing/") {
			fmt.Println(utils.JsonPrint(frame))
			break
		}

		result = frame
		fmt.Println(utils.JsonPrint(frame))
	}

	return
}
