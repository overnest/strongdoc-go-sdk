package test

import (
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"log"
	"os"
	"testing"
)

const (
	DefaultConfig = "dev"
)

// control all tests within package, load config before testing
func TestMain(m *testing.M) {
	if err := testUtils.LoadConfig(DefaultConfig); err != nil {
		log.Println("fail to load config file: ", err)
		return
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}
