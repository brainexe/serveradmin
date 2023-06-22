package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const apiEndpointQuery = "/api/dataset/query"

// ServerObject is a map of key-value attributes of a SA object
type ServerObject struct {
	attributes map[string]string
	// todo add changes + .Set() etc here
}

func (s ServerObject) Get(attribute string) string {
	return s.attributes[attribute]
}

func sendRequest(endpoint string, settings config, postData any) (*http.Response, error) {
	postStr, _ := json.Marshal(postData)

	fmt.Println(string(postStr))

	req, err := http.NewRequest("GET", settings.baseURL+endpoint, bytes.NewBuffer(postStr))
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	req.Header.Set("Content-Type", "application/x-json")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", now))
	req.Header.Set("X-API-Version", settings.apiVersion)

	// add other authentications
	req.Header.Set("X-SecurityToken", calcSecurityToken(settings.authToken, now, postStr))
	req.Header.Set("X-Application", settings.appID)

	return http.DefaultClient.Do(req)
}

func calcSecurityToken(authToken string, timestamp int64, data []byte) string {
	message := calcMessage(timestamp, data)
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(message))

	return hex.EncodeToString(mac.Sum(nil))
}

func calcMessage(timestamp int64, data []byte) string {
	return fmt.Sprintf("%d:%s", timestamp, data)
}

func calcAppID(authToken string) string {
	hasher := sha1.New()
	hasher.Write([]byte(authToken))

	return hex.EncodeToString(hasher.Sum(nil))
}
