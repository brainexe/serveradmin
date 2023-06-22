package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
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
	req.Header.Set("X-Timestamp", strconv.FormatInt(now, 10))
	req.Header.Set("X-API-Version", settings.apiVersion)

	messageToSign := []byte(calcMessage(now, postStr))

	if settings.authToken != "" {
		// old way...
		req.Header.Set("X-SecurityToken", calcSecurityToken(settings.authToken, now, postStr))
		req.Header.Set("X-Application", calcAppID(settings.authToken))
	} else if settings.sshPrivateKey != "" {
		// todo load it from disk is not done yet
		signer, err := ssh.ParsePrivateKey([]byte(settings.sshPrivateKey))
		signature, publicKey, err := getSignatureAndPublicKey(signer, messageToSign)
		checkErr(err)

		req.Header.Set("X-PublicKeys", publicKey)
		req.Header.Set("X-Signatures", signature)
	} else {
		sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
		checkErr(err)

		signers, err := agent.NewClient(sock).Signers()
		checkErr(err)

		for _, signer := range signers {
			signature, publicKey, err := getSignatureAndPublicKey(signer, messageToSign)
			if err != nil {
				continue
			}

			req.Header.Set("X-PublicKeys", publicKey)
			req.Header.Set("X-Signatures", signature)
			break
		}
	}

	fmt.Println(req.Header)

	return http.DefaultClient.Do(req)
}

func getSignatureAndPublicKey(signer ssh.Signer, message []byte) (string, string, error) {
	signature, sigErr := signer.Sign(rand.Reader, message)
	if sigErr != nil {
		return "", "", sigErr
	}

	sshSignature := base64.StdEncoding.EncodeToString(ssh.Marshal(signature))
	publicKey := base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal())

	return sshSignature, publicKey, nil
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

// just a sha1 hash of the API token
func calcAppID(authToken string) string {
	hasher := sha1.New()
	hasher.Write([]byte(authToken))

	return hex.EncodeToString(hasher.Sum(nil))
}
