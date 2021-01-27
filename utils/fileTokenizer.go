package utils

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"os"
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
	Close() error
}

type fileTokenizer struct {
	fileName  string
	mediaType string
	file      *os.File
	reader    io.Reader
	scanner   *scanner.Scanner
}

// OpenFileTokenizer opens a file for tokenization
func OpenFileTokenizer(fileName string) (FileTokenizer, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, errors.New(err)
	}

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		file.Close()
		return nil, errors.New(err)
	}
	file.Close()

	contentType := http.DetectContentType(buffer[:n])
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, errors.New(err)
	}

	file, err = os.Open(fileName)
	if err != nil {
		return nil, errors.New(err)
	}

	tokenizer := &fileTokenizer{fileName: fileName, mediaType: mediaType, file: file}
	err = tokenizer.Reset()
	if err != nil {
		return nil, errors.New(err)
	}

	return tokenizer, nil
}

func (token *fileTokenizer) Close() error {
	err := token.file.Close()
	if err != nil {
		return errors.New(err)
	}
	return nil
}

func (token *fileTokenizer) Reset() error {
	_, err := token.file.Seek(0, io.SeekStart)
	if err != nil {
		return errors.New(err)
	}

	if mimeText.MatchString(token.mediaType) {
		token.reader = token.file
	} else if mimeGzip.MatchString(token.mediaType) {
		reader, err := gzip.NewReader(token.file)
		if err != nil {
			return errors.New(err)
		}
		token.reader = reader
	} else {
		token.file.Close()
		return errors.Errorf("Can not process %v file", token.mediaType)
	}

	var scanner scanner.Scanner
	token.scanner = scanner.Init(token.reader)

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
