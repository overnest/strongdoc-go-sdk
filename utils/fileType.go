package utils

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"regexp"
	"text/scanner"

	"github.com/go-errors/errors"
)

const (
	FT_GZIP    = "GZIP"
	FT_TEXT    = "TEXT"
	FT_UNKNWON = "UNKNOWN"
)

// FileType is the file type
type FileType string

var mimeGzip = regexp.MustCompilePOSIX(`^application\/.*gzip$`)
var mimeText = regexp.MustCompilePOSIX(`^text\/plain$`)

// getFileType get the file type
func getFileType(file Storage) (fileType FileType, err error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return
	}

	contentType := http.DetectContentType(buffer[:n])
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return
	}

	if mimeText.MatchString(mediaType) {
		fileType = FT_TEXT
	} else if mimeGzip.MatchString(mediaType) {
		fileType = FT_GZIP
	} else {
		fileType = FT_UNKNWON
	}
	return
}

// GetFileTypeAndReader get type of storage and reader based on fileType
func GetFileTypeAndReader(storage Storage) (fileType FileType, reader io.Reader, err error) {
	fileType, err = getFileType(storage)
	if err != nil {
		return
	}

	reader, err = resetFile(fileType, storage)
	return
}

// resetFile resets a file to be read again
func resetFile(fileType FileType, storage Storage) (io.Reader, error) {
	//  data source seek to file beginning
	_, err := storage.Seek(0, SeekSet)
	if err != nil {
		return nil, err
	}

	// get reader based on fileType
	var reader io.Reader
	switch fileType {
	case FT_TEXT:
		reader = storage
		break
	case FT_GZIP:
		var err error
		reader, err = gzip.NewReader(storage)
		if err != nil {
			return nil, err
		}
		break
	default:
		return nil, errors.Errorf("Can not process %v file", fileType)
	}
	return reader, nil
}

func readerToScanner(reader io.Reader) *scanner.Scanner {
	var scanner scanner.Scanner
	return scanner.Init(reader)
}
