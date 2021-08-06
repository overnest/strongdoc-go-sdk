package common

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"github.com/overnest/strongdoc-go-sdk/search/index/docidx"
	docIndexCommon "github.com/overnest/strongdoc-go-sdk/search/index/docidx/common"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"github.com/overnest/strongdoc-go-sdk/utils"
	sscrypto "github.com/overnest/strongsalt-crypto-go"
	"gotest.tools/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

const (
	TestDocKeyID   = "docKey"
	TestTermKeyID  = "termKey"
	TestIndexKeyID = "indexKey"
)

func TestDocIndexGeneration(t *testing.T, sdc client.StrongDocClient, indexKey *sscrypto.StrongSaltKey, num int) (docIDs []string, docVers []uint64) {
	// remove existing doc indexes
	docs, err := docidx.InitTestDocuments(num, false)
	for _, doc := range docs {
		docIndexCommon.RemoveDocIndexes(sdc, doc.DocID)
	}

	// generate doc indexes
	for _, doc := range docs {
		docIDs = append(docIDs, doc.DocID)
		docVers = append(docVers, doc.DocVer)
		err = doc.CreateDoiAndDti(sdc, indexKey)
		assert.NilError(t, err)
	}
	return
}

func PrevTest(t *testing.T) client.StrongDocClient {
	if utils.TestLocal {
		return nil
	}
	// register org and admin
	sdc, orgs, users := testUtils.PrevTest(t, 1, 1)
	testUtils.DoRegistration(t, sdc, orgs, users)
	// login
	user := users[0][0]
	err := api.Login(sdc, user.UserID, user.Password, user.OrgID)
	assert.NilError(t, err)
	return sdc
}

var savedKeyPath string = path.Clean("/tmp/savedKeys")

func TestSaveKeys(keyMap map[string]*sscrypto.StrongSaltKey) error {
	if err := os.MkdirAll(savedKeyPath, 0770); err != nil {
		return err
	}

	for name, key := range keyMap {
		data, err := key.Serialize()
		if err != nil {
			return err
		}

		file, err := os.Create(path.Clean(fmt.Sprintf("%v/%v", savedKeyPath, name)))
		if err != nil {
			return err
		}
		n, err := file.Write(data)
		if err != nil {
			return err
		}
		if n != len(data) {
			return errors.Errorf("Written %v bytes instead of %v bytes", n, len(data))
		}
		file.Close()
	}

	return nil
}

func TestLoadKeys() (map[string]*sscrypto.StrongSaltKey, error) {
	keyMap := make(map[string]*sscrypto.StrongSaltKey)

	keyInfos, err := ioutil.ReadDir(savedKeyPath)
	if err != nil {
		return keyMap, err
	}

	for _, keyInfo := range keyInfos {
		if keyInfo.IsDir() {
			continue
		}

		file, err := os.Open(path.Clean(fmt.Sprintf("%v/%v", savedKeyPath, keyInfo.Name())))
		if err != nil {
			return keyMap, err
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			return keyMap, err
		}

		key, err := sscrypto.DeserializeKey(data)
		if err != nil {
			return keyMap, err
		}

		keyMap[keyInfo.Name()] = key
	}

	return keyMap, err
}

func TestGetKeys() (map[string]*sscrypto.StrongSaltKey, error) {
	keys, err := TestLoadKeys()
	if err != nil || len(keys) < 3 {
		docKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
		if err != nil {
			return nil, err
		}
		termKey, err := sscrypto.GenerateKey(sscrypto.Type_HMACSha512)
		if err != nil {
			return nil, err
		}
		indexKey, err := sscrypto.GenerateKey(sscrypto.Type_XChaCha20)
		if err != nil {
			return nil, err
		}

		keys = map[string]*sscrypto.StrongSaltKey{
			TestDocKeyID:   docKey,
			TestTermKeyID:  termKey,
			TestIndexKeyID: indexKey,
		}
		TestSaveKeys(keys)
		return keys, nil
	}

	return keys, err
}
