package utils

import (
	"compress/gzip"
	"github.com/go-errors/errors"
	"io"
	"mime"
	"net/http"
	"regexp"
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
func getFileType(source Source) (fileType FileType, err error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := source.Read(buffer)
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

// getFileTypeAndReaderCloser get fileType, reader and Closer
func getFileTypeAndReaderCloser(source Source) (fileType FileType, reader io.Reader, closer io.Closer, err error) {
	fileType, err = getFileType(source)
	if err != nil {
		return
	}

	reader, closer, err = resetFile(fileType, source)
	return
}

// resetFile resets a file to be read again
func resetFile(fileType FileType, source Source) (io.Reader, io.Closer, error) {
	//  data source seek to file beginning
	_, err := source.Seek(0, SeekSet)
	if err != nil {
		return nil, nil, err
	}

	// get reader and closer based on fileType
	switch fileType {
	case FT_TEXT:
		return source, nil, nil
	case FT_GZIP:
		readerCloser, err := gzip.NewReader(source)
		if err != nil {
			return nil, nil, err
		}
		return readerCloser, readerCloser, nil
	default:
		return nil, nil, errors.Errorf("Can not process %v file", fileType)
	}

}
