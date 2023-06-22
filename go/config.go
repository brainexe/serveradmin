package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	version = "4.9.0"
)

type config struct {
	baseURL    string
	apiVersion string
	sshSigner  ssh.Signer

	// deprecated API Token
	authToken string
}

// todo: load only once for all requests, maybe something for sync.Once?
func getConfig() (config, error) {
	cfg := config{
		apiVersion: version,
	}

	baseUrl := os.Getenv("SERVERADMIN_BASE_URL")
	if baseUrl == "" {
		return cfg, fmt.Errorf("env var SERVERADMIN_BASE_URL not set")
	}
	cfg.baseURL = baseUrl

	// todo: load key from disk etc...
	sshPrivateKey := []byte("")
	if len(sshPrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(sshPrivateKey)
		checkErr(err)

		cfg.sshSigner = signer
	} else if os.Getenv("SSH_AUTH_SOCK") != "" {
		sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
		checkErr(err)

		signers, err := agent.NewClient(sock).Signers()
		checkErr(err)

		for _, signer := range signers {
			_, err := signer.Sign(rand.Reader, []byte("test"))
			if err == nil {
				cfg.sshSigner = signer
				break
			}
		}
	}

	// oldschool fallback
	if cfg.sshSigner == nil {
		cfg.authToken = os.Getenv("SERVERADMIN_TOKEN")
	}

	return cfg, nil
}

// deprecated, maybe not needed at all
func getAuthToken() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configFilePath := filepath.Join(userHome, ".adminapirc")

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return "", nil
	}

	file, err := os.Open(configFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		keyValue := strings.SplitN(line, "=", 2)

		if len(keyValue) != 2 {
			continue
		}

		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])

		if key == "auth_token" {
			return value, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
