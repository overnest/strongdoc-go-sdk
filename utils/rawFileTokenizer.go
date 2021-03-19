package utils

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"
)

// RawFileTokenizer tokenizes a file
type RawFileTokenizer interface {
	NextToken() (term string, space rune, wordCounter uint64, err error) // return lower-case alphaNumeric token
	NextRawToken() (term string, space rune, err error)                  // return raw token
}

type rawFileTokenizer struct {
	wordCounter uint64
	reader      *bufio.Reader
}

// OpenRawFileTokenizer opens a reader for tokenization
func OpenRawFileTokenizer(storage Storage) (RawFileTokenizer, error) {
	_, freader, err := GetFileTypeAndReader(storage)
	if err != nil {
		return nil, err
	}

	tokenizer := &rawFileTokenizer{
		reader:      bufio.NewReader(freader),
		wordCounter: 0,
	}

	return tokenizer, nil
}

func (token *rawFileTokenizer) NextToken() (term string, space rune, wordCounter uint64, err error) {
	for {
		term, space, err = token.NextRawToken()
		if err != nil {
			return
		}
		if IsAlphaNumeric(term) {
			term = strings.ToLower(term)
			wordCounter = token.wordCounter
			token.wordCounter++
			return
		}
	}
}

func (token *rawFileTokenizer) NextRawToken() (term string, space rune, err error) {
	err = nil
	term = ""
	space = unicode.ReplacementChar

	var buffer bytes.Buffer
	bufWriter := bufio.NewWriter(&buffer)

	for err == nil {
		var r rune
		r, _, err = token.reader.ReadRune()
		if err != nil && err != io.EOF {
			return
		}

		if err == io.EOF || unicode.IsSpace(r) {
			space = r
			break
		} else {
			_, err = bufWriter.WriteRune(r)
			if err != nil {
				return
			}
		}
	}

	bufWriter.Flush()
	if buffer.Len() > 0 {
		var berr error
		term, berr = buffer.ReadString(' ')
		if berr != nil && berr != io.EOF {
			return
		}
	}

	return
}
