package docidxv1

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/utils"
)

// DocTermSourceV1 is the Document Term Source V1
type DocTermSourceV1 interface {
	// Returns io.EOF error if there are no more terms
	GetNextTerm() (string, uint64, error)
	Reset() error
	Close() error
}

//
// Text File Source
//
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

//
// Document Offset Index Source
//
type docTermSourceDocOffsetV1 struct {
	doi      common.DocOffsetIdx
	termsLoc map[string][]uint64
	terms    []string
}

// OpenDocTermSourceDocOffsetV1 opens the Document Offset source
func OpenDocTermSourceDocOffsetV1(doi common.DocOffsetIdx) (DocTermSourceV1, error) {
	switch doi.GetDoiVersion() {
	case common.DOI_V1:
		_, ok := doi.(*DocOffsetIdxV1)
		if !ok {
			return nil, errors.Errorf("Document offset index is not version %v",
				doi.GetDoiVersion())
		}
	default:
		return nil, errors.Errorf("Document offset index version %v is not supported",
			doi.GetDoiVersion())
	}

	return &docTermSourceDocOffsetV1{doi, make(map[string][]uint64), nil}, nil
}

// GetNextTerm gets the next term from the DOI source
func (dts *docTermSourceDocOffsetV1) GetNextTerm() (term string, loc uint64, err error) {
	term = ""
	loc = 0
	err = nil

	switch dts.doi.GetDoiVersion() {
	case common.DOI_V1:
		doiv1, ok := dts.doi.(*DocOffsetIdxV1)
		if !ok {
			err = errors.Errorf("Document offset index is not version %v",
				dts.doi.GetDoiVersion())
			return
		}

		if dts.terms == nil || len(dts.terms) == 0 {
			blk, berr := doiv1.ReadNextBlock()
			if berr != nil && berr != io.EOF {
				err = berr
				return
			}

			if blk != nil && len(blk.TermLoc) > 0 {
				dts.termsLoc = blk.TermLoc
				dts.terms = make([]string, 0, len(dts.termsLoc))
				for term := range dts.termsLoc {
					dts.terms = append(dts.terms, term)
				}
			}
		}

		if dts.terms != nil && len(dts.terms) > 0 {
			t := dts.terms[0]
			locs := dts.termsLoc[t]
			if len(locs) > 0 {
				term = t
				loc = locs[0]
				locs = locs[1:]
				dts.termsLoc[t] = locs
			}

			if len(locs) == 0 {
				dts.terms = dts.terms[1:]
			}

			err = nil
			return
		}

		err = io.EOF
		return
	default:
		return "", 0, errors.Errorf("Document offset index version %v is not supported",
			dts.doi.GetDoiVersion())
	}
}

func (dts *docTermSourceDocOffsetV1) Reset() error {
	switch dts.doi.GetDoiVersion() {
	case common.DOI_V1:
		doiv1, ok := dts.doi.(*DocOffsetIdxV1)
		if !ok {
			return errors.Errorf("Document offset index is not version %v",
				dts.doi.GetDoiVersion())
		}
		return doiv1.Reset()
	default:
		return errors.Errorf("Document offset index version %v is not supported",
			dts.doi.GetDoiVersion())
	}
}

func (dts *docTermSourceDocOffsetV1) Close() error {
	switch dts.doi.GetDoiVersion() {
	case common.DOI_V1:
		doiv1, ok := dts.doi.(*DocOffsetIdxV1)
		if !ok {
			return errors.Errorf("Document offset index is not version %v",
				dts.doi.GetDoiVersion())
		}
		return doiv1.Close()
	default:
		return errors.Errorf("Document offset index version %v is not supported",
			dts.doi.GetDoiVersion())
	}
}
