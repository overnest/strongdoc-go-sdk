package crypto

import (
	"encoding/json"

	"github.com/go-errors/errors"
)

// BlockCryptoPlainHdrBodyV1 is the plaintext header body of block cryto
type BlockCryptoPlainHdrBodyV1 interface {
	BlockCryptoVersion
	GetKeyType() string
	SetKeyType(keyType string)
	GetNonce() []byte
	SetNonce(nonce []byte)
	Deserialize(data []byte) (BlockCryptoPlainHdrBody, error)
}

// BlockCryptoPlainHdrBodyV1S is the plaintext header body of block cryto
type BlockCryptoPlainHdrBodyV1S struct {
	BlockCryptoVersionS
	KeyType string
	Nonce   []byte
}

// BlockCryptoCipherHdrBodyV1 is the ciphertext header body of block cryto
type BlockCryptoCipherHdrBodyV1 interface {
	BlockCryptoVersion
}

// BlockCryptoCipherHdrBodyV1S is the ciphertext header body of block cryto
type BlockCryptoCipherHdrBodyV1S struct {
	BlockCryptoVersionS
}

// GetKeyType gets the key type
func (plain *BlockCryptoPlainHdrBodyV1S) GetKeyType() string {
	return plain.KeyType
}

// SetKeyType sets the key type
func (plain *BlockCryptoPlainHdrBodyV1S) SetKeyType(keyType string) {
	plain.KeyType = keyType
}

// GetNonce gets the nonce
func (plain *BlockCryptoPlainHdrBodyV1S) GetNonce() []byte {
	return plain.Nonce
}

// SetNonce sets the nonce
func (plain *BlockCryptoPlainHdrBodyV1S) SetNonce(nonce []byte) {
	plain.Nonce = nonce
}

// // Serialize serializes structure in to JSON
// func (plain *BlockCryptoPlainHdrBodyV1) Serialize() ([]byte, error) {
// 	b, err := json.Marshal(plain)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	return b, nil
// }

// Deserialize deserializes structure from JSON
func (plain *BlockCryptoPlainHdrBodyV1S) Deserialize(data []byte) (BlockCryptoPlainHdrBody, error) {
	err := json.Unmarshal(data, plain)
	if err != nil {
		return nil, errors.New(err)
	}
	return plain, nil
}

// // Serialize serializes structure in to JSON
// func (cipher *BlockCryptoCipherHdrBodyV1) Serialize() ([]byte, error) {
// 	b, err := json.Marshal(cipher)
// 	if err != nil {
// 		return nil, errors.New(err)
// 	}
// 	return b, nil
// }

// Deserialize deserializes structure from JSON
func (cipher *BlockCryptoCipherHdrBodyV1S) Deserialize(data []byte) (BlockCryptoCipherHdrBody, error) {
	err := json.Unmarshal(data, cipher)
	if err != nil {
		return nil, errors.New(err)
	}
	return cipher, nil
}
