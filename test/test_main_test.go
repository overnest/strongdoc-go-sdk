package test

import (
	"flag"
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"os"
	"testing"
)

const (
	ORG1       = "org1"
	ORG1_Addr  = "org1Addr"
	ORG1_Email = "org1@example.com"
	ORG1_Admin_Name  = "org1_admin"
	ORG1_Admin_Pwd   = "admin_password"
	ORG1_Admin_Email = "admin@example.com"
	ORG1_Source      = "Test Active"
	ORG1_SourceData  = ""
)
var ORG1_Admin_ID string

func testSetup() error {
	_, err := client.InitStrongDocManager(client.LOCAL, false)
	if err != nil {
		return err
	}
	// register for an org
	_, ORG1_Admin_ID, err = api.RegisterOrganization(ORG1, ORG1_Addr,
		ORG1_Email, ORG1_Admin_Name, ORG1_Admin_Pwd, ORG1_Admin_Email, ORG1_Source, ORG1_SourceData)
	return err
}

func testTeardown() error {
	// hard remove organization
	if err := superUserLogin(); err != nil {
		return err
	}
	if err := hardRemoveOrg(ORG1); err != nil {
		return err
	}
	return superUserLogout()
}

// control all tests within package
func TestMain(m *testing.M) {
	var opt string
	flag.StringVar(&opt, "option", "", "test options")
	flag.Parse()
	var exitVal int
	if opt != "" {
		if err := loadConfig(opt); err != nil {
			fmt.Println("fail to load config file: ", err)
			return
		}
		if err := testSetup(); err != nil {
			fmt.Println("fail to set up: ", err)
			return
		}
		exitVal = m.Run()
		if err := testTeardown(); err != nil {
			fmt.Println("fail to tear down: ", err)
			return
		}
	}else{
		exitVal = m.Run()
	}
	os.Exit(exitVal)
}

