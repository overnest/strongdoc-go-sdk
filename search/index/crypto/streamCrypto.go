package crypto

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc/crypto/common"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	sscryptointf "github.com/overnest/strongsalt-crypto-go/interfaces"
)

// StreamCrypto is a streaming crypto interface that implements both the StrongSalt style crypto and
// traditional golang reader/readerat/writer/writerat/seeker interface
type StreamCrypto interface {
	common.StrongSaltCrypto
	io.Reader
	io.ReaderAt
	io.Writer
	io.WriterAt
}

type streamCrypto struct {
	key        *sscrypto.StrongSaltKey
	nonce      []byte
	initOffset int64
	curOffset  int64
	writer     io.Writer
	writerat   io.WriterAt
	reader     io.Reader
	readerat   io.ReaderAt
	seeker     io.Seeker
}

// CreateNonce creates an streaming crypto nonce
func CreateNonce(key *sscrypto.StrongSaltKey) ([]byte, error) {
	if midStreamKey, ok := key.Key.(sscryptointf.KeyMidstream); ok {
		nonce, err := midStreamKey.GenerateNonce()
		if err != nil {
			return nil, errors.New(err)
		}
		return nonce, nil
	}

	return nil, errors.Errorf("StrongSalt key is not a KeyMidstream type")
}

// CreateStreamCrypto creates an streaming crypto for encryption writing encrypted data to the provided storage
func CreateStreamCrypto(key *sscrypto.StrongSaltKey, nonce []byte, store interface{}, initOffset int64) (StreamCrypto, error) {
	if _, ok := store.(io.Writer); store != nil && !ok {
		return nil, errors.Errorf("The storage interface must implement io.Writer")
	}

	return initStreamCrypto(key, nonce, store, initOffset)
}

// OpenStreamCrypto opens a streaming crypto for decryption or reading decrypted from an encrypted storage
func OpenStreamCrypto(key *sscrypto.StrongSaltKey, nonce []byte, store interface{}, initOffset int64) (StreamCrypto, error) {
	if _, ok := store.(io.Reader); store != nil && !ok {
		return nil, errors.Errorf("The storage interface must implement io.Reader")
	}

	return initStreamCrypto(key, nonce, store, initOffset)
}

func initStreamCrypto(key *sscrypto.StrongSaltKey, nonce []byte, store interface{}, initOffset int64) (StreamCrypto, error) {
	if key.Type.Name != sscrypto.Type_XChaCha20.Name {
		return nil, errors.Errorf("Key %v passed in, but requires %v key", key.Type.Name, sscrypto.Type_XChaCha20.Name)
	}

	if len(nonce) != key.NonceSize() {
		return nil, errors.Errorf("Nonce of size %v passed in, but nonce of size %v", len(nonce), key.NonceSize())
	}

	var ok bool
	crypto := &streamCrypto{key, nonce, initOffset, initOffset, nil, nil, nil, nil, nil}

	if crypto.writer, ok = store.(io.Writer); !ok {
		crypto.writer = nil
	}

	if crypto.writerat, ok = store.(io.WriterAt); !ok {
		crypto.writerat = nil
	}

	if crypto.reader, ok = store.(io.Reader); !ok {
		crypto.reader = nil
	}

	if crypto.readerat, ok = store.(io.ReaderAt); !ok {
		crypto.readerat = nil
	}

	if crypto.seeker, ok = store.(io.Seeker); !ok {
		crypto.seeker = nil
	}

	return crypto, nil
}

func (crypto *streamCrypto) GetKey() *sscrypto.StrongSaltKey {
	return crypto.key
}

func (crypto *streamCrypto) GetNonce() []byte {
	return crypto.nonce
}

func (crypto *streamCrypto) calcBlockCounter(offset int64) (blockCount int32, blockOffset int) {
	blockCount = int32(int64(offset) / int64(crypto.key.BlockSize()))
	blockOffset = int(int64(offset) % int64(crypto.key.BlockSize()))
	return
}

// Encrypt data using streaming crypto
func (crypto *streamCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(crypto.curOffset - crypto.initOffset)
	plaintextPadded := append(make([]byte, blockOffset), plaintext...)
	ciphertext, err := crypto.GetKey().EncryptIC(plaintextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	crypto.curOffset += int64(len(plaintext))
	return ciphertext[blockOffset:], nil
}

// Write encrypts data using streaming crypto and write it to the underlying storage
func (crypto *streamCrypto) Write(plaintext []byte) (int, error) {
	if crypto.writer == nil {
		return 0, errors.Errorf("The underlying storage does not implement io.Writer interface")
	}

	ciphertext, err := crypto.Encrypt(plaintext)
	if err != nil {
		return 0, errors.New(err)
	}

	return crypto.writer.Write(ciphertext)
}

// Decrypt data using streaming crypto
func (crypto *streamCrypto) Decrypt(ciphertext []byte) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(crypto.curOffset - crypto.initOffset)
	ciphertextPadded := append(make([]byte, blockOffset), ciphertext...)
	plaintext, err := crypto.GetKey().DecryptIC(ciphertextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	crypto.curOffset += int64(len(ciphertext))
	return plaintext[blockOffset:], nil
}

// Read encrypted data from the underlying storage and decrypts it
func (crypto *streamCrypto) Read(plaintext []byte) (n int, err error) {
	if crypto.reader == nil {
		return 0, errors.Errorf("The underlying storage does not implement io.Reader interface")
	}

	ciphertext := make([]byte, len(plaintext))

	n, err = crypto.reader.Read(ciphertext)
	if err != nil && err != io.EOF {
		err = errors.New(err)
		return
	}

	var plain []byte
	plain, derr := crypto.Decrypt(ciphertext[:n])
	if derr != nil {
		err = errors.New(derr)
		return
	}

	copy(plaintext, plain)
	return
}

// DecryptAt decrypts data starting at the given offset. Note that
// this function does not store state, and will not effect the
// result of any other function call
func (crypto *streamCrypto) DecryptAt(ciphertext []byte, offset int64) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(offset)
	ciphertextPadded := append(make([]byte, blockOffset), ciphertext...)
	plaintext, err := crypto.GetKey().DecryptIC(ciphertextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	return plaintext[blockOffset:], nil
}

// ReadAt reads encrypted data at a certain offset from the underlying storage and decrypts it
func (crypto *streamCrypto) ReadAt(plaintext []byte, off int64) (n int, err error) {
	if crypto.readerat == nil {
		return 0, errors.Errorf("The underlying storage does not implement io.ReaderAt interface")
	}

	ciphertext := make([]byte, len(plaintext))
	n, err = crypto.readerat.ReadAt(ciphertext, off)
	if err != nil && err != io.EOF {
		err = errors.New(err)
		return
	}

	var plain []byte
	plain, derr := crypto.DecryptAt(ciphertext[:n], off-crypto.initOffset)
	if derr != nil {
		err = errors.New(derr)
		return
	}

	copy(plaintext, plain)
	return
}

func (crypto *streamCrypto) EncryptAt(plaintext []byte, offset int64) ([]byte, error) {
	blockCount, blockOffset := crypto.calcBlockCounter(offset)
	plaintextPadded := append(make([]byte, blockOffset), plaintext...)
	ciphertext, err := crypto.GetKey().EncryptIC(plaintextPadded, crypto.GetNonce(), blockCount)
	if err != nil {
		return nil, errors.New(err)
	}
	return ciphertext[blockOffset:], nil
}

func (crypto *streamCrypto) WriteAt(plaintext []byte, off int64) (n int, err error) {
	if crypto.writerat == nil {
		return 0, errors.Errorf("The underlying storage does not implement io.WriterAt interface")
	}

	ciphertext, err := crypto.EncryptAt(plaintext, off-crypto.initOffset)
	if err != nil {
		return 0, errors.New(err)
	}

	return crypto.writerat.WriteAt(ciphertext, off)
}

func (crypto *streamCrypto) Seek(offset int64) (n int64, err error) {
	if offset < 0 || offset < crypto.initOffset {
		return 0, errors.Errorf("Can not seek past begging of stream")
	}

	n = offset
	if crypto.seeker != nil {
		n, err = crypto.seeker.Seek(offset, io.SeekStart)
		if err != nil {
			return n, errors.New(err)
		}
	}

	crypto.curOffset = offset
	return n, nil
}

// End function for streaming crypto
func (crypto *streamCrypto) End() ([]byte, error) {
	return nil, nil
}
