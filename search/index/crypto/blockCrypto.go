package crypto

import (
	"io"

	ssblocks "github.com/overnest/strongsalt-common-go/blocks"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"

	"github.com/go-errors/errors"
)

const (
	BLOCK_CRYPTO_V1     = uint32(1)
	BLOCK_CRYPT_VERSION = BLOCK_CRYPTO_V1
)

// BlockCrypto is used to encrypt/decrypt StrongSalt block stream
type BlockCrypto interface {
	GetKey() *sscrypto.StrongSaltKey
	GetNonce() []byte
	Encrypt(plaintext []byte) ([]byte, error)
	EncryptAt(ciphertext []byte, blockID int64) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	DecryptAt(ciphertext []byte, blockID int64) ([]byte, error)
}

type blockCrypto struct {
	key           *sscrypto.StrongSaltKey
	nonce         []byte
	plainHdrBody  []byte
	cipherHdrBody []byte
	offset        int64
	writer        ssblocks.BlockListWriterV1
	reader        ssblocks.BlockListReaderV1
}

// BlockPlainHdrBody contains information needed to encrypt/decrypt StrongSalt block stream
type BlockPlainHdrBody interface {
	SetBlockCryptoVersion(version uint32)
	GetBlockCryptoVersion() uint32
	SetKeyType(keyType string)
	GetKeyType() string
	SetNonce(nonce []byte)
	GetNonce() []byte
	Serialize() ([]byte, error)
	Deserialize(data []byte) (BlockPlainHdrBody, error)
}

// BlockCipherHdrBody contains information needed to encrypt/decrypt StrongSalt block stream
type BlockCipherHdrBody interface {
	SetBlockCryptoVersion(version uint32)
	GetBlockCryptoVersion() uint32
	Serialize() ([]byte, error)
	Deserialize(data []byte) (BlockPlainHdrBody, error)
}

func createNonce(key *sscrypto.StrongSaltKey) ([]byte, error) {
	if midStreamKey, ok := key.Key.(sscryptointf.KeyMidstream); ok {
		nonce, err := midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
		return nonce, nil
	}

	return nil, errors.Errorf("StrongSalt key is not of type KeyMidstream")
}

// CreateBlockCrypto   Create an block crypto
func CreateBlockCrypto(key *sscrypto.StrongSaltKey, paddedBlockSize uint32,
	plainHdrBody BlockPlainHdrBody, cipherHdrBody BlockCipherHdrBody,
	store interface{}) (BlockCrypto, error) {

	if key.Type != sscrypto.Type_XChaCha20 {
		return nil, errors.Errorf("Key type %v is not supported. The only supported key type is %v",
			key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	nonce, err := createNonce(key)
	if err != nil {
		return nil, errors.New(err)
	}

	cipherHdrBody.SetBlockCryptoVersion(BLOCK_CRYPT_VERSION)
	plainHdrBody.SetBlockCryptoVersion(BLOCK_CRYPT_VERSION)
	plainHdrBody.SetKeyType(key.Type.Name)
	plainHdrBody.SetNonce(nonce)

	plainSerial, err := plainHdrBody.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	cipherSerial, err := cipherHdrBody.Serialize()
	if err != nil {
		return nil, errors.New(err)
	}

	writer, ok := store.(io.Writer)
	if !ok {
		return nil, errors.Errorf("The passed in storage does not implement io.Writer")
	}

	n, err := writer.Write(plainSerial)
	if err != nil {
		return nil, errors.New(err)
	}
	if n != len(plainSerial) {
		return nil, errors.Errorf("Failed to write the entire plaintext header")
	}

	crypto := &blockCrypto{
		key:           key,
		nonce:         nonce,
		plainHdrBody:  plainSerial,
		cipherHdrBody: cipherSerial,
		offset:        0,
		writer:        nil,
		reader:        nil,
	}

	crypto.writer, err = ssblocks.NewBlockListWriterV1(store, paddedBlockSize, 0)
	if err != nil {
		return nil, errors.New(err)
	}

	return crypto, nil
}

func (crypto *blockCrypto) GetKey() *sscrypto.StrongSaltKey {
	return crypto.key
}

func (crypto *blockCrypto) GetNonce() []byte {
	return crypto.nonce
}

func (crypto *blockCrypto) calcBlockCounter(offset int64) (blockCount int32, blockOffset int) {
	blockCount = int32(offset / int64(crypto.key.BlockSize()))
	blockOffset = int(offset % int64(crypto.key.BlockSize()))
	return
}

// Encrypt   Encrypt data using block crypto
func (crypto *blockCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(crypto.offset)
	plaintextPadded := append(make([]byte, blockOffset), plaintext...)
	ciphertext, err := crypto.GetKey().EncryptIC(plaintextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	crypto.offset += int64(len(plaintext))
	return ciphertext[blockOffset:], nil
}

// Decrypt   Decrypt data using block crypto
func (crypto *blockCrypto) Decrypt(ciphertext []byte) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(crypto.offset)
	ciphertextPadded := append(make([]byte, blockOffset), ciphertext...)
	plaintext, err := crypto.GetKey().DecryptIC(ciphertextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	crypto.offset += int64(len(ciphertext))
	return plaintext[blockOffset:], nil
}

// DecryptAt decrypts data starting at the given offset. Note that
// this function does not store state, and will not effect the
// result of any other function call
func (crypto *blockCrypto) DecryptAt(ciphertext []byte, offset int64) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(offset)
	ciphertextPadded := append(make([]byte, blockOffset), ciphertext...)
	plaintext, err := crypto.GetKey().DecryptIC(ciphertextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	return plaintext[blockOffset:], nil
}

func (crypto *blockCrypto) EncryptAt(plaintext []byte, offset int64) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(offset)
	plaintextPadded := append(make([]byte, blockOffset), plaintext...)
	ciphertext, err := crypto.GetKey().EncryptIC(plaintextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	return ciphertext[blockOffset:], nil
}
