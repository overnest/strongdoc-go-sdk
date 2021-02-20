package searchidx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/search/index/crypto"
	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	ssheaders "github.com/overnest/strongsalt-common-go/headers"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
	"github.com/shengdoushi/base58"
)

// SearchTermIdxV1 is a structure for search term index V1
type SearchTermIdxV1 struct {
	StiVersionS
	Term       string
	Owner      SearchIdxOwner
	OldSti     SearchTermIdx
	DelDocs    *DeletedDocsV1
	TermKey    *sscrypto.StrongSaltKey
	IndexKey   *sscrypto.StrongSaltKey
	IndexNonce []byte
	InitOffset uint64
	termHmac   string
	updateID   string
	writer     io.WriteCloser
	reader     io.ReadCloser
	bwriter    ssblocks.BlockListWriterV1
	breader    ssblocks.BlockListReaderV1
}

func getSearchTermIdxPathV1(prefix string, owner SearchIdxOwner, term, updateID string) string {
	return fmt.Sprintf("%v/searchterm", GetSearchIdxPathV1(prefix, owner, term, updateID))
}

// CreateSearchTermIdxV1 creates a search term index writer V1
func CreateSearchTermIdxV1(owner SearchIdxOwner, term string, termKey, indexKey *sscrypto.StrongSaltKey,
	oldSti SearchTermIdx, delDocs *DeletedDocsV1) (*SearchTermIdxV1, error) {

	var err error
	sti := &SearchTermIdxV1{
		StiVersionS: StiVersionS{StiVer: STI_V1},
		Term:        term,
		Owner:       owner,
		OldSti:      oldSti,
		DelDocs:     delDocs,
		TermKey:     termKey,
		IndexKey:    indexKey,
		IndexNonce:  nil,
		InitOffset:  0,
		termHmac:    "",
		updateID:    newUpdateIDV1(),
		reader:      nil,
		bwriter:     nil,
		breader:     nil,
	}

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if midStreamKey, ok := indexKey.Key.(sscryptointf.KeyMidstream); ok {
		sti.IndexNonce, err = midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
	} else {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	sti.termHmac, err = sti.createTermHmac()
	if err != nil {
		return nil, err
	}

	_, err = sti.createWriter()
	if err != nil {
		return nil, err
	}

	// Create plaintext and ciphertext headers
	plainHdrBody := &StiPlainHdrBodyV1{
		StiVersionS: StiVersionS{StiVer: STI_V1},
		KeyType:     sti.IndexKey.Type.Name,
		Nonce:       sti.IndexNonce,
		TermHmac:    sti.termHmac,
		UpdateID:    sti.updateID,
	}

	cipherHdrBody := &StiCipherHdrBodyV1{
		BlockVersion: BlockVersion{BlockVer: STI_BLOCK_V1},
	}

	plainHdrBodySerial, err := plainHdrBody.serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	plainHdr := ssheaders.CreatePlainHdr(ssheaders.HeaderTypeJSONGzip, plainHdrBodySerial)
	plainHdrSerial, err := plainHdr.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdrBodySerial, err := cipherHdrBody.serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdr := ssheaders.CreateCipherHdr(ssheaders.HeaderTypeJSONGzip, cipherHdrBodySerial)
	cipherHdrSerial, err := cipherHdr.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	// Write the plaintext header to storage
	n, err := sti.writer.Write(plainHdrSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(plainHdrSerial) {
		return nil, errors.Errorf("Failed to write the entire plaintext header")
	}

	// Initialize the streaming crypto to encrypt ciphertext header and the
	// blocks after that
	streamCrypto, err := crypto.CreateStreamCrypto(sti.IndexKey, plainHdrBody.Nonce,
		sti.writer, int64(sti.InitOffset)+int64(n))
	if err != nil {
		return nil, errors.New(err)
	}

	// Write the ciphertext header to storage
	n, err = streamCrypto.Write(cipherHdrSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(cipherHdrSerial) {
		return nil, errors.Errorf("Failed to write the entire ciphertext header")
	}

	// Create a block list writer using the streaming crypto so the blocks will be
	// encrypted.
	sti.bwriter, err = ssblocks.NewBlockListWriterV1(streamCrypto, uint32(STI_BLOCK_SIZE_MAX),
		sti.InitOffset+uint64(len(plainHdrSerial)+len(cipherHdrSerial)))
	if err != nil {
		return nil, errors.New(err)
	}

	return sti, nil
}

// OpenSearchTermIdxV1 opens a search term index reader V1
func OpenSearchTermIdxV1(owner SearchIdxOwner, term string, termKey, indexKey *sscrypto.StrongSaltKey, updateID string) (*SearchTermIdxV1, error) {
	var err error
	sti := &SearchTermIdxV1{
		StiVersionS: StiVersionS{StiVer: STI_V1},
		Term:        term,
		Owner:       owner,
		OldSti:      nil,
		DelDocs:     nil,
		TermKey:     termKey,
		IndexKey:    indexKey,
		IndexNonce:  nil,
		InitOffset:  0,
		termHmac:    "",
		updateID:    updateID,
		reader:      nil,
		bwriter:     nil,
		breader:     nil,
	}

	if _, ok := termKey.Key.(sscryptointf.KeyMAC); !ok {
		return nil, errors.Errorf("The key type %v is not a MAC key", termKey.Type.Name)
	}

	if midStreamKey, ok := indexKey.Key.(sscryptointf.KeyMidstream); ok {
		sti.IndexNonce, err = midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
	} else {
		return nil, errors.Errorf("The key type %v is not a midstream key", indexKey.Type.Name)
	}

	sti.termHmac, err = sti.createTermHmac()
	if err != nil {
		return nil, err
	}

	reader, err := sti.createReader()
	if err != nil {
		return nil, err
	}

	plainHdr, _, err := ssheaders.DeserializePlainHdrStream(reader)
	if err != nil {
		return nil, err
	}

	plainHdrBodyData, err := plainHdr.GetBody()
	if err != nil {
		return nil, err
	}

	version, err := DeserializeStiVersion(plainHdrBodyData)
	if err != nil {
		return nil, err
	}

	if version.GetStiVersion() != STI_V1 {
		return nil, errors.Errorf("Search term index version %v is not %v", version.GetStiVersion(), STI_V1)
	}

	// Parse plaintext header body
	plainHdrBody := &StiPlainHdrBodyV1{}
	plainHdrBody, err = plainHdrBody.deserialize(plainHdrBodyData)
	if err != nil {
		return nil, errors.New(err)
	}

	return nil, nil
}

func createTermHmac(term string, termKey *sscrypto.StrongSaltKey) (string, error) {
	err := termKey.MACReset()
	if err != nil {
		return "", errors.New(err)
	}

	_, err = termKey.MACWrite([]byte(term))
	if err != nil {
		return "", errors.New(err)
	}

	hmac, err := termKey.MACSum(nil)
	if err != nil {
		return "", errors.New(err)
	}

	return base58.Encode(hmac, base58.BitcoinAlphabet), nil
}

func (sti *SearchTermIdxV1) createTermHmac() (string, error) {
	return createTermHmac(sti.Term, sti.TermKey)
}

func (sti *SearchTermIdxV1) createWriter() (io.WriteCloser, error) {
	if sti.writer != nil {
		return sti.writer, nil
	}

	var err error
	path := getSearchTermIdxPathV1(GetSearchIdxPathPrefix(), sti.Owner, sti.termHmac, sti.updateID)

	if err = os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return nil, err
	}

	sti.writer, err = os.Create(path)
	if err != nil {
		return nil, errors.New(err)
	}

	return sti.writer, nil
}

func (sti *SearchTermIdxV1) createReader() (io.ReadCloser, error) {
	if sti.reader != nil {
		return sti.reader, nil
	}

	var err error
	path := getSearchTermIdxPathV1(GetSearchIdxPathPrefix(), sti.Owner, sti.termHmac, sti.updateID)

	sti.reader, err = os.Open(path)
	if err != nil {
		return nil, errors.New(err)
	}

	return sti.reader, nil
}

func (sti *SearchTermIdxV1) GetMaxBlockDataSize() uint64 {
	if sti.bwriter != nil {
		return uint64(sti.bwriter.GetMaxDataSize())
	}

	return STI_BLOCK_SIZE_MAX
}

func (sti *SearchTermIdxV1) WriteBlock(block *SearchTermIdxBlkV1) error {
	if sti.writer == nil || sti.bwriter == nil {
		return errors.Errorf("No writer available")
	}

	if block != nil {
		b, err := block.Serialize()
		if err != nil {
			return err
		}
		_, err = sti.bwriter.WriteBlockData(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sti *SearchTermIdxV1) Close() error {
	var werr, rerr error = nil, nil

	if sti.writer != nil {
		werr = sti.writer.Close()
	}

	if sti.reader != nil {
		rerr = sti.reader.Close()
	}

	if werr != nil {
		return werr
	}

	if rerr != nil {
		return rerr
	}

	return nil
}
