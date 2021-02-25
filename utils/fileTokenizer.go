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

var mimeGzip = regexp.MustCompilePOSIX(`^application\/.*gzip$`)
var mimeText = regexp.MustCompilePOSIX(`^text\/plain$`)

// FileTokenizer tokenizes a file
type FileTokenizer interface {
	NextToken() (string, *scanner.Position, error)
	Reset() error
}

type fileTokenizer struct {
	mediaType string
	storage   interface{}
	scanner   *scanner.Scanner
}

// convert storage to io.Reader
func (token *fileTokenizer) getReader() (io.Reader, error) {
	reader, ok := token.storage.(io.Reader)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Reader")
	}
	return reader, nil
}

// convert storage to io.Seeker
func (token *fileTokenizer) getSeeker() (io.Seeker, error) {
	seeker, ok := token.storage.(io.Seeker)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Seeker")
	}
	return seeker, nil
}

// OpenFileTokenizer opens a reader for tokenization
// storage implements Seeker, Reader interfaces
func OpenFileTokenizer(storage interface{}) (FileTokenizer, error) {
	tokenizer := &fileTokenizer{storage: storage}

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := tokenizer.read(buffer)
	if err != nil && err != io.EOF {
		return nil, errors.New(err)
	}

	contentType := http.DetectContentType(buffer[:n])
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, errors.New(err)
	}

	tokenizer.mediaType = mediaType

	err = tokenizer.Reset() // seek to 0
	if err != nil {
		return nil, errors.New(err)
	}

	return tokenizer, nil
}

func (token *fileTokenizer) read(p []byte) (n int, err error) {
	reader, err := token.getReader()
	if err != nil {
		return 0, errors.New(err)
	}
	return reader.Read(p)
}

func (token *fileTokenizer) seek(offset int64, whence int) (n int64, err error) {
	seeker, err := token.getSeeker()
	if err != nil {
		return 0, errors.New(err)
	}
	return seeker.Seek(offset, whence)
}

func (token *fileTokenizer) Reset() error {
	_, err := token.seek(0, SeekSet) // data source seek to 0
	if err != nil {
		return errors.New(err)
	}

	reader, err := token.getReader()
	if err != nil {
		return errors.New(err)
	}

	if mimeGzip.MatchString(token.mediaType) { // gzip file
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return errors.New(err)
		}
	} else if !mimeText.MatchString(token.mediaType) {
		return errors.Errorf("Can not process %v file", token.mediaType)
	}

	var scanner scanner.Scanner
	token.scanner = scanner.Init(reader)

	return nil
}

func (token *fileTokenizer) NextToken() (string, *scanner.Position, error) {
	pos := token.scanner.Pos()
	rune := token.scanner.Scan()
	if rune == scanner.EOF {
		return "", nil, io.EOF
	}

	return token.scanner.TokenText(), &pos, nil
}
