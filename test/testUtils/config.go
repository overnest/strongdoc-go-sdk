package testUtils

import (
	"encoding/json"
	"github.com/overnest/strongdoc-go-sdk/utils"
	"io/ioutil"
	"os"
)

const (
	DefaultConfig = "/test/testUtils/dev.json"
)

type configData struct {
	Superuser superuser `json:"superuser"`
}
type superuser struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	ID     string `json:"id"`
	Passwd string `json:"passwd"`
}

func setEnv(key string, value string) error {
	return os.Setenv(key, value)
}

// load config file
func LoadConfig(configFileName string) error {
	filePath, err := utils.FetchFileLoc(configFileName)
	if err != nil {
		return err
	}
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var result configData
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return err
	}
	if err := setEnv(SUPER_USER_ID, result.Superuser.ID); err != nil {
		return err
	}
	if err := setEnv(SUPER_USER_PASSWORD, result.Superuser.Passwd); err != nil {
		return err
	}
	return nil
}
