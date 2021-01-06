package crypto

import (
	"encoding/json"

	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"github.com/go-errors/errors"
)

//////////////////////////////////////////////////////////////////
//
//                     Block Crypto Version
//
//////////////////////////////////////////////////////////////////

const (
	BLOCK_CRYPTO_V1     = uint32(1)
	BLOCK_CRYPT_VERSION = BLOCK_CRYPTO_V1
)

// BlockCryptoVersion stores the block crypto version
type BlockCryptoVersion interface {
	GetVersion() uint32
}

// BlockCryptoVersionS is the structure to embed in other structure
type BlockCryptoVersionS struct {
	BlockCryptoVersion uint32
}

// GetVersion gets the version
func (bcv *BlockCryptoVersionS) GetVersion() uint32 {
	return bcv.BlockCryptoVersion
}

// Serialize serializes the blockCryptoVersion object
func (bcv *BlockCryptoVersionS) Serialize() ([]byte, error) {
	b, err := json.Marshal(bcv)
	if err != nil {
		return nil, errors.New(err)
	}
	return b, nil
}

// Deserialize deserializes the blockCryptoVersion object
func (bcv *BlockCryptoVersionS) Deserialize(data []byte) (BlockCryptoVersion, error) {
	err := json.Unmarshal(data, bcv)
	if err != nil {
		return nil, errors.New(err)
	}
	return bcv, nil
}

// DeserializeBlockCryptoVersion deserializes the BlockCryptoVersion object
func DeserializeBlockCryptoVersion(data []byte) (BlockCryptoVersion, error) {
	bcv := &BlockCryptoVersionS{}
	return bcv.Deserialize(data)
}

//////////////////////////////////////////////////////////////////
//
//                      Block Crypto Headers
//
//////////////////////////////////////////////////////////////////

// BlockCryptoPlainHdrBody contains information needed to encrypt/decrypt StrongSalt block stream
type BlockCryptoPlainHdrBody interface {
	BlockCryptoVersion
	Serialize() ([]byte, error)
	Deserialize(data []byte) (BlockCryptoPlainHdrBody, error)
}

// DeserializeBlockCryptoPlainHdrBody deserializes the block crypto plaintext header object
func DeserializeBlockCryptoPlainHdrBody(data []byte) (BlockCryptoPlainHdrBody, error) {
	version, err := DeserializeBlockCryptoVersion(data)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetVersion() {
	case BLOCK_CRYPTO_V1:
		body := &BlockCryptoPlainHdrBodyV1S{}
		return body.Deserialize(data)
	default:
		return nil, errors.Errorf("Unsupported block crypto plaintext header version %v",
			version.GetVersion())
	}
}

// BlockCryptoCipherHdrBody contains information needed to encrypt/decrypt StrongSalt block stream
type BlockCryptoCipherHdrBody interface {
	BlockCryptoVersion
	Serialize() ([]byte, error)
	Deserialize(data []byte) (BlockCryptoCipherHdrBody, error)
}

// DeserializeBlockCryptoCipherHdrBody deserializes the block crypto ciphertext header object
func DeserializeBlockCryptoCipherHdrBody(data []byte) (BlockCryptoCipherHdrBody, error) {
	version, err := DeserializeBlockCryptoVersion(data)
	if err != nil {
		return nil, errors.New(err)
	}

	switch version.GetVersion() {
	case BLOCK_CRYPTO_V1:
		body := &BlockCryptoCipherHdrBodyV1S{}
		return body.Deserialize(data)
	default:
		return nil, errors.Errorf("Unsupported block crypto plaintext header version %v",
			version.GetVersion())
	}
}

//////////////////////////////////////////////////////////////////
//
//                          Block Crypto
//
//////////////////////////////////////////////////////////////////

// BlockCrypto is used to encrypt/decrypt StrongSalt block stream
type BlockCrypto interface {
	GetKey() *sscrypto.StrongSaltKey
	GetNonce() []byte
	GetPlainHeaderBody() BlockCryptoPlainHdrBody
	GetPlainHeaderData() []byte
	GetCipherHeaderBody() BlockCryptoCipherHdrBody
	GetCipherHeaderData() []byte
	Write(blockdata []byte) (err error)
	Read() (blockdata []byte, err error)
	ReadAt(blockID uint32) (blockdata []byte, err error)
}

type blockCrypto struct {
	key                 *sscrypto.StrongSaltKey
	nonce               []byte
	plainHdrBody        BlockCryptoPlainHdrBody
	plainHdrBodySerial  []byte
	cipherHdrBody       BlockCryptoCipherHdrBody
	cipherHdrBodySerial []byte
	initOffset          int64
	writer              ssblocks.BlockListWriterV1
	reader              ssblocks.BlockListReaderV1
}
