package apidef

import (
	"fmt"
	"time"

	"github.com/karlseguin/ccache"
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

var (
	keyCache        = ccache.New(ccache.Configure().MaxSize(10000).ItemsToPrune(100))
	keyCacheEnabled = true
	keycacheTTL     = time.Minute * 30
)

///////////////////////////////////////////////////////////////////////////////////////
//
//                     Convert To/From Grpc + Client Format
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

	passwdKey, err := userKdf.GenerateKey([]byte(password))
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
		SKey: passwdKey,
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

func ParseKdfProtoKey(pkey *proto.Key, password string) (*Key, error) {
	if pkey == nil {
		return nil, nil
	}

	if pkey.KeyType != proto.KeyType_KDF {
		return nil, fmt.Errorf("ParseKdfProtoKey can only parse KDF key.")
	}

	kdf, err := cryptoKdf.DeserializeKdf(pkey.PublicData)
	if err != nil {
		return nil, nil
	}

	passwdKey, err := kdf.GenerateKey([]byte(password))
	if err != nil {
		return nil, nil
	}

	key := &Key{
		PKey: pkey,
		SKdf: &Kdf{
			Kdf:      kdf,
			Password: password,
		},
		SKey: passwdKey,
	}

	if keyCacheEnabled {
		keyCache.Set(pkey.KeyID, key, keycacheTTL)
	}

	return key, nil
}

func ParseProtoKey(pkey *proto.Key, decryptionKeys ...*Key) (*Key, error) {
	if pkey == nil {
		return nil, nil
	}

	key := &Key{
		PKey: pkey,
		SKdf: nil,
		SKey: nil,
	}

	if pkey.KeyType == proto.KeyType_KDF {
		return nil, fmt.Errorf("ParseProtoKey can not parse KDF key.")
	}

	// Process other types of keys
	var decryptKey *Key = nil
	var encryptKey *proto.EncryptedKey = nil
	for _, encryptedKey := range pkey.EncryptedKeys {
		if keyCacheEnabled {
			// Find the decryption key in cache first
			item := keyCache.Get(encryptedKey.Encryptor.LocationID)
			if item != nil && !item.Expired() {
				if key, ok := item.Value().(*Key); ok {
					decryptKey = key
					encryptKey = encryptedKey
					break
				}
			}
		}

		// Find the decryption key from the arguments
		for _, decryptionKey := range decryptionKeys {
			if decryptionKey.PKey.KeyID == encryptedKey.Encryptor.LocationID {
				decryptKey = decryptionKey
				encryptKey = encryptedKey
				break
			}
		}

		if decryptKey != nil {
			break
		}
	}

	if decryptKey == nil {
		return nil, fmt.Errorf("can not decrypt key %v(%v) with provided decryption keys",
			pkey.GetKeyID(), pkey.GetKeyType())
	}

	plainKey, err := decryptKey.Decrypt(encryptKey.Ciphertext)
	if err != nil {
		return nil, err
	}

	skey, err := cryptoKey.DeserializeKey(plainKey)
	if err != nil {
		return nil, err
	}

	key.SKey = skey
	if keyCacheEnabled {
		keyCache.Set(pkey.KeyID, key, keycacheTTL)
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
		if encryptor == nil || encryptor.PKey == nil {
			continue
		}

		encryptedKey, err := encryptor.Encrypt(serialKey)
		if err != nil {
			return err
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

func (k *Key) Encrypt(plaintext []byte) ([]byte, error) {
	if k.SKey == nil {
		return nil, fmt.Errorf("the encryption key is missing")
	}

	return k.SKey.Encrypt(plaintext)
}

func (k *Key) Decrypt(ciphertext []byte) ([]byte, error) {
	if k.SKey == nil {
		return nil, fmt.Errorf("the decryption key is missing")
	}

	return k.SKey.Decrypt(ciphertext)
}
