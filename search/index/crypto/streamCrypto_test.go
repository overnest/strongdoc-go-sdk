package crypto

import (
	"math/rand"
	"os"
	"testing"

	sscrypto "github.com/overnest/strongsalt-crypto-go"

	"gotest.tools/assert"
)

var keycontent = `
Before the widespread use of message authentication codes and authenticated encryption, 
it was common to discuss the "error propagation" properties as a selection criterion for 
a mode of operation. It might be observed, for example, that a one-block error in the 
transmitted ciphertext would result in a one-block error in the reconstructed plaintext 
for ECB mode encryption, while in CBC mode such an error would affect two blocks.
Some felt that such resilience was desirable in the face of random errors (e.g., line 
noise), while others argued that error correcting increased the scope for attackers to 
maliciously tamper with a message.
However, when proper integrity protection is used, such an error will result (with high 
probability) in the entire message being rejected. If resistance to random error is 
desirable, error-correcting codes should be applied to the ciphertext before transmission.
Authenticated encryption
Main article: Authenticated encryption
A number of modes of operation have been designed to combine secrecy and authentication in 
a single cryptographic primitive. Examples of such modes are XCBC,[25] IACBC, IAPM,[26] OCB, 
EAX, CWC, CCM, and GCM. Authenticated encryption modes are classified as single-pass modes 
or double-pass modes. Some single-pass authenticated encryption algorithms, such as OCB mode, 
are encumbered by patents, while others were specifically designed and released in a way to 
avoid such encumberment.
In addition, some modes also allow for the authentication of unencrypted associated data, 
and these are called AEAD (authenticated encryption with associated data) schemes. For example, 
EAX mode is a double-pass AEAD scheme while OCB mode is single-pass.
`

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func TestStreamCrypto(t *testing.T) {
	testEncryptDecrypt(t)
	testReadWriteEncrypt(t, 0)
}

func testEncryptDecrypt(t *testing.T) {
	content := []byte(keycontent)

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	nonce, err := CreateNonce(key)
	assert.NilError(t, err)
	crypto, err := CreateStreamCrypto(key, nonce, nil, 0)
	assert.NilError(t, err)

	ciphertext, err := crypto.Encrypt(content)
	assert.NilError(t, err)
	_, err = crypto.Seek(0)
	assert.NilError(t, err)
	plaintext, err := crypto.Decrypt(ciphertext)
	assert.NilError(t, err)
	assert.DeepEqual(t, content, plaintext)

	// Random sized encryption
	_, err = crypto.Seek(0)
	remain := len(content)
	offset := int(0)
	for remain > 0 {
		size := rand.Intn(key.BlockSize() * 2)
		if size > remain {
			size = remain
		}

		ctext, err := crypto.Encrypt(content[offset : offset+size])
		assert.NilError(t, err)
		assert.DeepEqual(t, ctext, ciphertext[offset:offset+size])

		offset += size
		remain -= size
	}

	// Random sized decryption
	_, err = crypto.Seek(0)
	remain = len(content)
	offset = 0
	for remain > 0 {
		size := rand.Intn(key.BlockSize() * 2)
		if size > remain {
			size = remain
		}

		ptext, err := crypto.Decrypt(ciphertext[offset : offset+size])
		assert.NilError(t, err)
		assert.DeepEqual(t, ptext, plaintext[offset:offset+size])

		offset += size
		remain -= size
	}

	// Random location and sized decryption
	for i := 0; i < 100; i++ {
		start := rand.Intn(len(content))
		end := rand.Intn(len(content)-start) + start

		ptext, err := crypto.DecryptAt(ciphertext[start:end], int64(start))
		assert.NilError(t, err)
		assert.DeepEqual(t, ptext, plaintext[start:end])
	}

	// Random location and sized encryption
	for i := 0; i < 100; i++ {
		start := rand.Intn(len(content))
		end := rand.Intn(len(content)-start) + start

		ctext, err := crypto.EncryptAt(plaintext[start:end], int64(start))
		assert.NilError(t, err)
		assert.DeepEqual(t, ctext, ciphertext[start:end])
	}

	// Do DecryptAt for every position
	for at := 0; at < len(ciphertext); at++ {
		b, err := crypto.DecryptAt(ciphertext[at:min(at+511, len(ciphertext))], int64(at))
		assert.NilError(t, err)

		c := []byte(plaintext)[at : at+len(b)]
		assert.DeepEqual(t, b, c)
	}
}

func testReadWriteEncrypt(t *testing.T, initOffset int64) {
	fileName := "/tmp/blocklistv1_test"
	content := []byte(keycontent)

	// Create file with encrypted content
	file, err := os.Create(fileName)
	assert.NilError(t, err)
	defer os.Remove(fileName)
	defer file.Close()

	if initOffset > 0 {
		garbage := make([]byte, initOffset)
		n, err := rand.Read(garbage)
		assert.NilError(t, err)
		assert.Equal(t, n, len(garbage))
		n, err = file.Write(garbage)
		assert.NilError(t, err)
		assert.Equal(t, n, len(garbage))
	}

	key, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
	assert.NilError(t, err)
	nonce, err := CreateNonce(key)
	assert.NilError(t, err)
	crypto, err := CreateStreamCrypto(key, nonce, file, initOffset)
	assert.NilError(t, err)

	// Random sized writes with readat verification
	remain := len(content)
	offset := int(0)
	for remain > 0 {
		size := rand.Intn(key.BlockSize() * 2)
		if size > remain {
			size = remain
		}

		n, err := crypto.Write(content[offset : offset+size])
		assert.NilError(t, err)
		assert.Equal(t, n, size)

		piece := make([]byte, size)
		n, err = crypto.ReadAt(piece, initOffset+int64(offset))
		assert.NilError(t, err)
		assert.Equal(t, n, size)

		assert.DeepEqual(t, piece, content[offset:offset+size])

		offset += size
		remain -= size
	}

	file.Close()

	// Open file with encrypted content
	file, err = os.OpenFile(fileName, os.O_RDWR, 0755)
	assert.NilError(t, err)
	defer file.Close()
	// stat, err := file.Stat()
	// assert.NilError(t, err)

	// Skip the initial offset
	if initOffset > 0 {
		garbage := make([]byte, initOffset)
		n, err := file.Read(garbage)
		assert.NilError(t, err)
		assert.Equal(t, n, len(garbage))
	}

	crypto, err = OpenStreamCrypto(key, nonce, file, initOffset)
	assert.NilError(t, err)

	// Do random read and writes in different locations
	for i := 0; i < 100; i++ {
		offset := rand.Intn(len(content))
		size := min(rand.Intn(key.BlockSize()*2), len(content)-offset)

		// Read existing data at offset
		readData := make([]byte, size)
		n, err := crypto.ReadAt(readData, initOffset+int64(offset))
		assert.NilError(t, err)
		assert.Equal(t, n, len(readData))
		assert.DeepEqual(t, readData, content[offset:offset+size])

		// Write new data at offset
		writeData := make([]byte, size)
		for i := 0; i < size; i++ {
			writeData[i] = byte('X')
		}
		n, err = crypto.WriteAt(writeData, initOffset+int64(offset))
		assert.NilError(t, err)
		assert.Equal(t, n, len(writeData))
		copy(content[offset:], writeData)

		// Read the new data back at offset
		n, err = crypto.ReadAt(readData, initOffset+int64(offset))
		assert.NilError(t, err)
		assert.Equal(t, n, len(readData))
		assert.DeepEqual(t, readData, content[offset:offset+size])
	}

	// Do random seeks
	for i := 0; i < 100; i++ {
		offset := rand.Intn(len(content))
		size := min(rand.Intn(key.BlockSize()*2), len(content)-offset)

		// Read existing data at offset
		readData := make([]byte, size)
		off, err := crypto.Seek(initOffset + int64(offset))
		assert.NilError(t, err)
		assert.Equal(t, off, initOffset+int64(offset))

		n, err := crypto.Read(readData)
		assert.NilError(t, err)
		assert.Equal(t, n, len(readData))
		assert.DeepEqual(t, readData, content[offset:offset+size])
	}
}
