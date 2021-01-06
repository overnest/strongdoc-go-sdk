package crypto

import (
	"encoding/json"
	"fmt"
	"testing"

	"gotest.tools/assert"
)

type TestStruct struct {
	BlockCryptoPlainHdrBodyV1S
	F1 uint32
	F2 string
}

func TestBlockCrypto(t *testing.T) {
	test := &TestStruct{
		BlockCryptoPlainHdrBodyV1S{
			BlockCryptoVersionS{10}, "keytype", []byte{1, 2, 3}},
		1,
		"f2val"}

	b, err := json.Marshal(test)
	assert.NilError(t, err)
	fmt.Println(string(b))

	bcv, err := DeserializeBlockCryptoVersion(b)
	assert.NilError(t, err)
	fmt.Println(bcv.GetVersion())

	var bcvv BlockCryptoVersion = test
	_, ok := bcvv.(*TestStruct)
	fmt.Println(ok)

}
