package api

import (
	"fmt"
	"time"

	"github.com/overnest/strongdoc-go-sdk/proto"
	cryptoKey "github.com/overnest/strongsalt-crypto-go"
	cryptoKdf "github.com/overnest/strongsalt-crypto-go/kdf"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Kdf struct {
	Kdf      *cryptoKdf.StrongSaltKdf
	Password string
}

type Key struct {
	PKey *proto.Key
	SKdf *Kdf
	SKey *cryptoKey.StrongSaltKey
}

///////////////////////////////////////////////////////////////////////////////////////
//
//                     Convert To/From Grpc + Server Format
//
///////////////////////////////////////////////////////////////////////////////////////

func NewUserKeys(userID, password string) (kdf, asym, term, search *Key, err error) {
	kdf, err = NewKdfKey(userID, password)
	if err != nil {
		return
	}

	asym, err = NewAsymKey(userID, []*Key{kdf})
	if err != nil {
		return
	}

	term, err = NewHmacKey(userID, []*Key{kdf})
	if err != nil {
		return
	}

	search, err = NewSymKey(userID, []*Key{kdf})
	if err != nil {
		return
	}

	return
}

func NewKdfKey(ownerID, password string) (*Key, error) {
	userKdf, err := cryptoKdf.New(cryptoKdf.Type_Argon2, cryptoKey.Type_Secretbox)
	if err != nil {
		return nil, err
	}

	serialKdf, err := userKdf.Serialize()
	if err != nil {
		return nil, err
	}

	pKey := &proto.Key{
		KeyID:         "", // This is only available after being stored on the server
		KeyType:       proto.KeyType_KDF,
		PublicData:    serialKdf,
		OwnerID:       ownerID,
		EncryptedKeys: nil, // KDF keys are not encrypted on the server
		CreatedAt:     timestamppb.New(time.Now()),
	}

	return &Key{
		PKey: pKey,
		SKey: nil,
		SKdf: &Kdf{
			Kdf:      userKdf,
			Password: password,
		},
	}, nil
}

func NewAsymKey(ownerID string, encryptors []*Key) (*Key, error) {
	asymKey, err := cryptoKey.GenerateKey(cryptoKey.Type_X25519)
	if err != nil {
		return nil, err
	}

	return newKey(ownerID, asymKey, encryptors)
}

func NewSymKey(ownerID string, encryptors []*Key) (*Key, error) {
	symKey, err := cryptoKey.GenerateKey(cryptoKey.Type_XChaCha20)
	if err != nil {
		return nil, err
	}

	return newKey(ownerID, symKey, encryptors)
}

func NewHmacKey(ownerID string, encryptors []*Key) (*Key, error) {
	hmacKey, err := cryptoKey.GenerateKey(cryptoKey.Type_HMACSha512)
	if err != nil {
		return nil, err
	}

	return newKey(ownerID, hmacKey, encryptors)
}

func newKey(ownerID string, sskey *cryptoKey.StrongSaltKey, encryptors []*Key) (*Key, error) {
	if sskey == nil {
		return nil, nil
	}

	var pubKey []byte = nil
	var err error = nil
	var keyType proto.KeyType

	if sskey.IsSymmetric() {
		keyType = proto.KeyType_SYMMETRIC
	} else if sskey.IsAsymmetric() {
		keyType = proto.KeyType_ASYMMETRIC
		pubKey, err = sskey.SerializePublic()
		if err != nil {
			return nil, err
		}
	} else if sskey.CanMAC() {
		keyType = proto.KeyType_HMAC
	} else {
		return nil, fmt.Errorf("only the following key types are supported: %v,%v,%v",
			proto.KeyType_SYMMETRIC, proto.KeyType_ASYMMETRIC, proto.KeyType_HMAC)
	}

	key := &Key{
		PKey: &proto.Key{
			KeyID:         "", // This is only available after being stored on the server
			KeyType:       keyType,
			PublicData:    pubKey,
			OwnerID:       ownerID,
			EncryptedKeys: make([]*proto.EncryptedKey, 0, len(encryptors)),
			CreatedAt:     timestamppb.New(time.Now()),
		},
		SKey: sskey,
	}

	err = key.NewEncryptedKeys(encryptors)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (k *Key) NewEncryptedKeys(encryptors []*Key) error {
	if k == nil || len(encryptors) == 0 || k.SKey == nil { // k.SKdf does not need encryption
		return nil
	}

	serialKey, err := k.SKey.Serialize()
	if err != nil {
		return err
	}

	for _, encryptor := range encryptors {
		if encryptor == nil || encryptor.PKey == nil ||
			(encryptor.SKey == nil && encryptor.SKdf == nil) {
			continue
		}

		var encryptedKey []byte = nil

		if encryptor.SKdf != nil {
			passwdKey, err := encryptor.SKdf.Kdf.GenerateKey([]byte(encryptor.SKdf.Password))
			if err != nil {
				return err
			}

			encryptedKey, err = passwdKey.Encrypt(serialKey)
			if err != nil {
				return err
			}
		} else {
			encryptedKey, err = encryptor.SKey.Encrypt(serialKey)
			if err != nil {
				return err
			}
		}

		k.PKey.EncryptedKeys = append(k.PKey.GetEncryptedKeys(),
			&proto.EncryptedKey{
				KeyID: k.PKey.GetKeyID(),
				Encryptor: &proto.Encryptor{
					Location:   "", // This is set on the server
					LocationID: encryptor.PKey.GetKeyID(),
					OwnerID:    encryptor.PKey.GetOwnerID(),
				},
				Ciphertext: encryptedKey,
				CreatedAt:  timestamppb.New(time.Now()),
				DeletedAt:  nil,
			},
		)
	} // for _, encryptor := range encryptors

	return nil
}
