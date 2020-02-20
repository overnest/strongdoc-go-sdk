package utils

import (
	"fmt"
	"path"
	"runtime"
)

// FetchFileLoc fetches the file location relative to the
// module root, eg. the same directory as go.mod.
//
func FetchFileLoc(relativeFilePath string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot get runtime caller")
	}

	fmt.Printf("caller filename %v\n", filename)
	fmt.Printf("caller path %v\n", path.Dir(filename))

	absFilepath := path.Join(path.Dir(filename), relativeFilePath)
	fmt.Printf("certpath path %v\n", absFilepath)

	return absFilepath, nil
}