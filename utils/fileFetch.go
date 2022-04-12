package utils

import (
	"path"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
)

// FetchFileLoc fetches the file location relative to the
// module root, eg. the same directory as go.mod.
func FetchFileLoc(relativeFilePath string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("cannot get runtime caller")
	}
	rootFilePath := path.Join(path.Dir(filename), "..")
	if strings.HasPrefix(relativeFilePath, rootFilePath) {
		// The relative file path already starts with the absolute root file path
		return relativeFilePath, nil
	}
	absFilepath := path.Join(rootFilePath, relativeFilePath)
	return absFilepath, nil
}

// FetchFileLocRoot fetches the file location relative to the
// module root, eg. the same directory as go.mod.
func FetchFileLocRoot(relativeFilePath string) (string, error) {
	rootFilePath := GetRootFilePath()
	if strings.HasPrefix(relativeFilePath, rootFilePath) {
		// The relative file path already starts with the absolute root file path
		return relativeFilePath, nil
	}
	absFilepath := path.Join(rootFilePath, relativeFilePath)
	return absFilepath, nil
}

func GetRootFilePath() string {
	frame := getTopStackFrame()
	return path.Dir(frame.File)
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
	}

	for more {
		frame, more = frames.Next()
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "testing/") {
			break
		}

		result = frame
	}

	return
}
