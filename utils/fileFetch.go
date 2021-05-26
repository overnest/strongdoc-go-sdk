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
