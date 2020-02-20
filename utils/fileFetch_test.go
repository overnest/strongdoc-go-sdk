package utils

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestFileFetch(t *testing.T) {
	path, _ := FetchFileLoc("./")
	f, _ := os.Open(path)
	lst, _ := f.Readdirnames(0)
	assert.Contains(t, lst, "go.mod")
}
