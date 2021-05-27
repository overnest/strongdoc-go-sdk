package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
)

// open local file with read-write mode (overwrite existing file)
func OpenLocalFile(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, os.O_RDWR, 0755)
}

// create local file or open local file to append (make sure filepath directory is valid)
func createOrOpenLocalFile(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0755)
}

// create filepath directory when necessary and create file
func MakeDirAndCreateFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return nil, err
	}

	writer, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return writer, nil
}

func GetLocalFileSize(filepath string) (uint64, error) {
	file, err := OpenLocalFile(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	filesize := uint64(fileInfo.Size())
	return filesize, nil
}

// BinarySearchU64 finds a number in a sorted list.
// Returns the index where the value is found.
// Returns -1 if value is not found
func BinarySearchU64(list []uint64, val uint64) int {
	if list == nil || len(list) == 0 {
		return -1
	}

	left := 0
	right := len(list) - 1
	for true {
		mid := (left + right) / 2
		if list[mid] == val {
			return mid
		}

		// Can't find the value
		if left == right {
			return -1
		}

		if list[mid] > val {
			if mid > left {
				right = mid - 1
			} else {
				right = left
			}
		} else {
			if mid < right {
				left = mid + 1
			} else {
				left = right
			}
		}
	}

	return -1
}

// Min returns the minimum value
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// PrintStackTrace prints stack trace: https://pkg.go.dev/github.com/go-errors/errors
func PrintStackTrace(err error) {
	if eerr, ok := err.(*errors.Error); ok {
		fmt.Println(eerr.ErrorStack())
	}

}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// Keep only the first error
func FirstError(curErr, newErr error) error {
	if curErr == nil && newErr != nil {
		curErr = newErr
	}
	return curErr
}

// DiffSliceString returns the difference between two string slices
func DiffSliceString(slice1 []string, slice2 []string) []string {
	diffStr := []string{}
	m := map[string]int{}

	for _, s1Val := range slice1 {
		m[s1Val] = 1
	}
	for _, s2Val := range slice2 {
		m[s2Val] = m[s2Val] + 1
	}

	for mKey, mVal := range m {
		if mVal == 1 {
			diffStr = append(diffStr, mKey)
		}
	}

	return diffStr
}

func Grep(dir string, terms []string, removeNewline bool) (map[string][]uint64, error) { // Document -> TermOffsets in bytes
	regex := regexp.MustCompilePOSIX(strings.Join(terms, "[^a-zA-Z0-9]+"))
	removeNewlineRegex := regexp.MustCompilePOSIX("[\n\r]")

	dirPath, err := FetchFileLoc(dir)
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	type chanResult struct {
		offsets []uint64
		err     error
	}

	docMap := make(map[string][]uint64) // Document -> TermOffsets
	fileNameToChan := make(map[string](chan *chanResult))

	for _, file := range files {
		fileName := path.Join(dirPath, file.Name())

		if file.IsDir() {
			childDocMap, err := Grep(fileName, terms, removeNewline)
			if err != nil {
				return nil, err
			}
			for doc, offsets := range childDocMap {
				docMap[doc] = offsets
			}
		} else {
			channel := make(chan *chanResult)
			fileNameToChan[fileName] = channel

			// This executes in a separate thread
			go func(fileName string, channel chan<- *chanResult) {
				defer close(channel)
				result := &chanResult{
					offsets: []uint64{},
					err:     nil,
				}

				f, err := os.Open(fileName)
				if err != nil {
					result.err = err
					channel <- result
					return
				}
				defer f.Close()

				_, reader, _, err := getFileTypeAndReaderCloser(f)
				if err != nil {
					result.err = err
					channel <- result
					return
				}

				c, err := ioutil.ReadAll(reader)
				if err != nil {
					result.err = err
					channel <- result
					return
				}

				content := string(c)
				if removeNewline {
					content = removeNewlineRegex.ReplaceAllString(content, " ")
				}

				offsets := regex.FindAllStringIndex(content, -1)
				if len(offsets) > 0 {
					docOffsets := make([]uint64, len(offsets))
					result.offsets = docOffsets
					for i, offset := range offsets {
						docOffsets[i] = uint64(offset[0])
					}
				}

				channel <- result
			}(fileName, channel)
		}
	} // for _, file := range files

	for fileName, channel := range fileNameToChan {
		result := <-channel
		if result.err != nil {
			return nil, result.err
		}

		if len(result.offsets) > 0 {
			docMap[fileName] = result.offsets
		}
	}

	return docMap, nil
}
