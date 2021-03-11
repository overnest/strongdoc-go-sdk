package utils

import (
	"path"
	"runtime"

	"github.com/go-errors/errors"
)

// FetchFileLoc fetches the file location relative to the
// module root, eg. the same directory as go.mod.
func FetchFileLoc(relativeFilePath string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("cannot get runtime caller")
	}
	absFilepath := path.Join(path.Dir(filename), "..", relativeFilePath)
	return absFilepath, nil
}
