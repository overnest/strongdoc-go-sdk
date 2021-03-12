package utils

import (
	"os"
	"path/filepath"
)

func OpenLocalFile(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, os.O_RDWR, 0755)
}

func CreateLocalFile(filepath string) (*os.File, error) {
	return os.Create(filepath)
}

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

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
