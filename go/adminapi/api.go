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

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	apiEndpointQuery     = "/api/dataset/query"
	apiEndpointNewObject = "/api/dataset/new_object"
)

// ServerObject is a map of key-value attributes of a SA object
type ServerObject struct {
	attributes map[string]any
	// todo: add changes + .Set() etc here
}

// Get safely retrieves an attribute, converting JSON float64 numbers to int when needed
func (s ServerObject) Get(attribute string) any {
	if val, ok := s.attributes[attribute]; ok {
		if floatVal, isFloat := val.(float64); isFloat {
			return int(floatVal)
		}
		return val
	}
	return nil
}

func sendRequest(endpoint string, postData any) (*http.Response, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	postStr, _ := json.Marshal(postData)

	log.Info("Sending request to: ", config.baseURL+endpoint)

	fmt.Println(config.baseURL + endpoint)
	fmt.Println(string(postStr))

	req, err := http.NewRequest("GET", config.baseURL+endpoint, bytes.NewBuffer(postStr))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	now := time.Now().Unix()
	req.Header.Set("Content-Type", "application/x-json")
	req.Header.Set("X-Timestamp", strconv.FormatInt(now, 10))
	req.Header.Set("User-Agent", userAgent)

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
	} else if len(config.authToken) > 0 {
		req.Header.Set("X-SecurityToken", calcSecurityToken(config.authToken, now, postStr))
		req.Header.Set("X-Application", calcAppID(config.authToken))
	}

	// todo compression

	fmt.Println(req.Header)

	return http.DefaultClient.Do(req)
}

// calcSecurityToken calculates HMAC-SHA1 of timestamp:data
func calcSecurityToken(authToken []byte, timestamp int64, data []byte) string {
	mac := hmac.New(sha1.New, authToken)
	mac.Write(calcMessage(timestamp, data))

	return hex.EncodeToString(mac.Sum(nil))
}

// calcMessage efficiently concatenates timestamp:data without redundant allocations
func calcMessage(timestamp int64, data []byte) []byte {
	return append(append(strconv.AppendInt(nil, timestamp, 10), ':'), data...)
}

// calcAppID computes SHA-1 hash of the auth token
func calcAppID(authToken []byte) string {
	hash := sha1.Sum(authToken)

	return hex.EncodeToString(hash[:])
}
