package adminapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

const apiEndpointQuery = "/api/dataset/query"

// ServerObject is a map of key-value attributes of a SA object
type ServerObject struct {
	attributes map[string]any
	// todo: add changes + .Set() etc here
}

func (s ServerObject) Get(attribute string) any {
	// todo: .GetInt() etc?

	// treat float64 as int, as json numbers are float64 in go
	if val, ok := s.attributes[attribute].(float64); ok {
		return int(val)
	}

	return s.attributes[attribute]
}

func sendRequest(endpoint string, config config, postData any) (*http.Response, error) {
	postStr, _ := json.Marshal(postData)

	fmt.Println(string(postStr))

	req, err := http.NewRequest("GET", config.baseURL+endpoint, bytes.NewBuffer(postStr))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	now := time.Now().Unix()
	req.Header.Set("Content-Type", "application/x-json")
	req.Header.Set("X-Timestamp", strconv.FormatInt(now, 10))
	req.Header.Set("X-API-Version", config.apiVersion)

	if config.sshSigner != nil {
		// sign with private key or SSH agent
		messageToSign := calcMessage(now, postStr)
		signature, sigErr := config.sshSigner.Sign(rand.Reader, messageToSign)
		if sigErr != nil {
			return nil, fmt.Errorf("failed to sign request: %w", sigErr)
		}

		publicKey := base64.StdEncoding.EncodeToString(config.sshSigner.PublicKey().Marshal())
		sshSignature := base64.StdEncoding.EncodeToString(ssh.Marshal(signature))

		req.Header.Set("X-PublicKeys", publicKey)
		req.Header.Set("X-Signatures", sshSignature)
	} else if config.authToken != "" {
		req.Header.Set("X-SecurityToken", calcSecurityToken(config.authToken, now, postStr))
		req.Header.Set("X-Application", calcAppID(config.authToken))
	}

	fmt.Println(req.Header)

	return http.DefaultClient.Do(req)
}

// hmac sha1 of the timestamp + ":" + message
func calcSecurityToken(authToken string, timestamp int64, data []byte) string {
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write(calcMessage(timestamp, data))

	return hex.EncodeToString(mac.Sum(nil))
}

// just concatenate: timestamp + ":" + message
func calcMessage(timestamp int64, data []byte) []byte {
	return append(
		append(
			[]byte(strconv.FormatInt(timestamp, 10)),
			':',
		),
		data...,
	)
}

// just a sha1 hash of the API token
func calcAppID(authToken string) string {
	hasher := sha1.New()
	hasher.Write([]byte(authToken))

	return hex.EncodeToString(hasher.Sum(nil))
}
