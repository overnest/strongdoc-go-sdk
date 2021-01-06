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
	Close() error
}

type fileTokenizer struct {
	fileName string
	file     *os.File
	reader   io.Reader
	scanner  *scanner.Scanner
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

	tokenizer := &fileTokenizer{fileName: fileName, file: file}

	if mimeText.MatchString(mediaType) {
		tokenizer.reader = file
	} else if mimeGzip.MatchString(mediaType) {
		reader, err := gzip.NewReader(file)
		if err != nil {
			return nil, errors.New(err)
		}
		tokenizer.reader = reader
	} else {
		file.Close()
		return nil, errors.Errorf("Can not process %v file", contentType)
	}

	var scanner scanner.Scanner
	tokenizer.scanner = scanner.Init(tokenizer.reader)

	return tokenizer, nil
}

func (token *fileTokenizer) Close() error {
	err := token.file.Close()
	if err != nil {
		return errors.New(err)
	}
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
