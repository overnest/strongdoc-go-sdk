package utils

import (
	"io"
	"os"
	"regexp"
	"strings"
	"text/scanner"

	"github.com/go-errors/errors"
)

var alphaNumeric = regexp.MustCompilePOSIX(`^[a-zA-Z0-9]+$`)

// FileTokenizer tokenizes a file
type FileTokenizer interface {
	NextToken() (string, *scanner.Position, error)
	Reset() error
	Close() error
}

type fileTokenizer struct {
	fileName string
	fileType FileType
	file     *os.File
	reader   io.Reader
	scanner  *scanner.Scanner
}

// IsAlphaNumeric shows whether term is alphanumeric
func IsAlphaNumeric(term string) bool {
	return alphaNumeric.MatchString(term)
}

// OpenFileTokenizer opens a file for tokenization
func OpenFileTokenizer(fileName string) (FileTokenizer, error) {
	fileType, file, reader, err := OpenFile(fileName)
	if err != nil {
		return nil, err
	}

	tokenizer := &fileTokenizer{
		fileName: fileName,
		fileType: fileType,
		file:     file,
		reader:   reader}

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

func (token *fileTokenizer) Reset() error {
	reader, err := ResetFile(token.fileType, token.file)
	if err != nil {
		return err
	}
	token.reader = reader

	var scanner scanner.Scanner
	token.scanner = scanner.Init(reader)

	return nil
}

func (token *fileTokenizer) NextToken() (string, *scanner.Position, error) {
	for {
		pos := token.scanner.Pos()
		rune := token.scanner.Scan()
		if rune == scanner.EOF {
			return "", nil, io.EOF
		}

		t := token.scanner.TokenText()
		if IsAlphaNumeric(t) {
			return strings.ToLower(token.scanner.TokenText()), &pos, nil
		}
	}
}
