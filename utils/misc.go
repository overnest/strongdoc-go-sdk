package utils

import (
	"fmt"
	"github.com/go-errors/errors"
	"os"
	"path/filepath"
	"runtime"
)

func OpenLocalFile(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, os.O_RDWR, 0755)
}

func MakeDirAndCreateFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		fmt.Println(path)
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
