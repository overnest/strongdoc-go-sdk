package utils

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"os"
	"regexp"

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

// GetFileType get the file type
func GetFileType(fileName string) (FileType, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return FT_UNKNWON, errors.New(err)
	}

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		file.Close()
		return FT_UNKNWON, errors.New(err)
	}
	file.Close()

	contentType := http.DetectContentType(buffer[:n])
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return FT_UNKNWON, errors.New(err)
	}

	if mimeText.MatchString(mediaType) {
		return FT_TEXT, nil
	} else if mimeGzip.MatchString(mediaType) {
		return FT_GZIP, nil
	}

	return FT_UNKNWON, nil
}

// OpenFile opens a file for reading
func OpenFile(fileName string) (FileType, *os.File, io.ReadCloser, error) {
	fileType, err := GetFileType(fileName)
	if err != nil {
		return FT_UNKNWON, nil, nil, err
	}

	var reader io.ReadCloser = nil
	file, err := os.Open(fileName)
	if err != nil {
		return fileType, nil, nil, errors.New(err)
	}

	switch fileType {
	case FT_TEXT:
		reader = file
	case FT_GZIP:
		reader, err = gzip.NewReader(file)
		if err != nil {
			return fileType, nil, nil, errors.New(err)
		}
	default:
		return fileType, nil, nil, errors.Errorf("Can not process %v file", fileType)
	}

	return fileType, file, reader, nil
}

// ResetFile resets a file to be read again
func ResetFile(fileType FileType, file *os.File) (io.ReadCloser, error) {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.New(err)
	}

	switch fileType {
	case FT_TEXT:
		return file, nil
	case FT_GZIP:
		reader, err := gzip.NewReader(file)
		if err != nil {
			return nil, errors.New(err)
		}
		return reader, nil
	default:
		return nil, errors.Errorf("Can not process %v file", fileType)
	}
}
