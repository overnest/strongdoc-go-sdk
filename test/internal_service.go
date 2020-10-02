package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

/**
	implementations for InternalService
 */

const (
	LOGIN_API = "http://localhost:8081/v1/account/login"
	LOGOUT_API = "http://localhost:8081/v1/account/logout"
	REMOVE_ORG_API = "http://localhost:8081/v1/organization"
	USER_ID = "userid"
	PASSWORD = "passwd"
	AUTHENTICATION = "authorization"
	AUTHENTICATION_BEARER = "bearer"
	SUPER_USER_ID = "SUPER_USER_ID"
	SUPER_USER_PASSWORD = "SUPER_USER_PASSWORD"
	MaxIdleConnections int = 20
	RequestTimeout     int = 5
)

type TokenData struct {
	Token string
}

var internalServiceClient *http.Client
var token TokenData

func getClient() *http.Client {
	if internalServiceClient == nil {
		internalServiceClient = initializeService()
	}
	return internalServiceClient
}


func initializeService() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: MaxIdleConnections,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}
	fmt.Println("Internal Service initialized")
	return client
}

func buildRequest(method string, url string, withToken bool) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	if withToken {
		req.Header.Set(AUTHENTICATION, AUTHENTICATION_BEARER+" "+token.Token)
	}
	return req, nil
}

func sendRequest(req *http.Request) (*http.Response, error) {
	return getClient().Do(req)
}

func superUserLogin() error {
	// build request
	req, err := buildRequest("GET",
		LOGIN_API+"?"+USER_ID+"="+getSuperUserId()+"&"+PASSWORD+"="+getSuperUserPwd(),
		false)
	if err != nil {
		return err
	}
	// send request
	resp, err := sendRequest(req)
	if err != nil {
		return err
	}
	// decode token
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	return nil
}

func superUserLogout() error {
	// build request
	req, err := buildRequest("PUT", LOGOUT_API+"?"+USER_ID+"="+getSuperUserId(), true)
	if err != nil {
		return err
	}
	// send request
	_, err = sendRequest(req)
	return err
}

func hardRemoveOrg(orgid string) error {
	// build request
	req, err := buildRequest("DELETE", REMOVE_ORG_API+"/"+orgid, true)
	if err != nil {
		return err
	}
	// send request
	_, err = sendRequest(req)
	return err
}

func getSuperUserId() string {
	return os.Getenv(SUPER_USER_ID)
}

func getSuperUserPwd() string {
	return os.Getenv(SUPER_USER_PASSWORD)
}