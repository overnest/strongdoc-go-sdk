package doctermidx

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// DocTermSourceV1 is the Document Term Source V1
type DocTermSourceV1 interface {
	// Returns io.EOF error if there are no more terms
	GetNextTerm() (string, uint64, error)
	Reset() error
	Close() error
}

type docTermSourceTextFileV1 struct {
	filename  string
	tokenizer utils.FileTokenizer
}

// OpenDocTermSourceTextFileV1 opens the text file Document Term Source V1
func OpenDocTermSourceTextFileV1(filename string) (DocTermSourceV1, error) {
	tokenizer, err := utils.OpenFileTokenizer(filename)
	if err != nil {
		return nil, errors.New(err)
	}
	return &docTermSourceTextFileV1{filename, tokenizer}, nil

}

func (dts *docTermSourceTextFileV1) GetNextTerm() (string, uint64, error) {
	token, pos, err := dts.tokenizer.NextToken()
	if err == io.EOF {
		if pos != nil {
			return token, uint64(pos.Offset), err
		}
		return "", 0, err
	}
	if err != nil {
		return "", 0, errors.New(err)
	}
	return token, uint64(pos.Offset), nil
}

func (dts *docTermSourceTextFileV1) Reset() error {
	return dts.tokenizer.Reset()
}

func (dts *docTermSourceTextFileV1) Close() error {
	return dts.tokenizer.Close()
}
