package test

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type configData struct {
	Superuser    superuser   `json:"superuser"`
}
type superuser struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	ID     string `json:"id"`
	Passwd string `json:"passwd"`
}

func setEnv(key string, value string) error {
	val := os.Getenv(key)
	if val == "" {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

// load config file
func loadConfig(configFileName string) error {
	jsonFile, err := os.Open(configFileName+".json")
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
