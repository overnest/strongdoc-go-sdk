package utils

import (
	"io"
	"regexp"
	"strings"
	"text/scanner"
)

var alphaNumeric = regexp.MustCompilePOSIX(`^[a-zA-Z0-9]+$`)

// FileTokenizer tokenizes a file
type FileTokenizer interface {
	NextToken() (string, *scanner.Position, uint64, error)
	Reset() error
}

type Storage interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

type fileTokenizer struct {
	fileType    FileType
	storage     Storage
	scanner     *scanner.Scanner
	wordCounter uint64
}

// OpenFileTokenizer opens a reader for tokenization
func OpenFileTokenizer(storage Storage) (FileTokenizer, error) {
	fileType, reader, err := GetFileTypeAndReader(storage)
	if err != nil {
		return nil, err
	}

	tokenizer := &fileTokenizer{
		fileType: fileType,
		storage:  storage,
	}

	tokenizer.scanner = readerToScanner(reader)
	tokenizer.wordCounter = 0

	return tokenizer, nil
}

func (token *fileTokenizer) Reset() error {
	reader, err := resetFile(token.fileType, token.storage)
	if err != nil {
		return err
	}

	token.scanner = readerToScanner(reader)
	token.wordCounter = 0

	return nil
}

// IsAlphaNumeric shows whether term is alphanumeric
func IsAlphaNumeric(term string) bool {
	return alphaNumeric.MatchString(term)
}

func (token *fileTokenizer) NextToken() (string, *scanner.Position, uint64, error) {
	for {
		pos := token.scanner.Pos()
		rune := token.scanner.Scan() // error reports to os.Stderr

		if rune == scanner.EOF {
			return "", nil, 0, io.EOF
		}

		t := token.scanner.TokenText()

		if IsAlphaNumeric(t) {
			wordCounter := token.wordCounter
			token.wordCounter++
			return strings.ToLower(t), &pos, wordCounter, nil
		}
	}
}
