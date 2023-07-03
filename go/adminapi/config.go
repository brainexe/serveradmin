package adminapi

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
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
	authToken  string
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
	cfg.baseURL = strings.TrimRight(baseUrl, "/api")

	if privateKeyPath, ok := os.LookupEnv("SERVERADMIN_KEY_PATH"); ok {
		// todo: load key from disk etc...
		_ = privateKeyPath
		sshPrivateKey := []byte("")
		signer, err := ssh.ParsePrivateKey(sshPrivateKey)
		if err != nil {
			return cfg, fmt.Errorf("failed to parse private key: %w", err)
		}

		cfg.sshSigner = signer
	} else if authSock, ok := os.LookupEnv("SSH_AUTH_SOCK"); ok {
		sock, err := net.Dial("unix", authSock)
		if err != nil {
			return cfg, fmt.Errorf("failed to connect to SSH agent: %w", err)
		}

		signers, err := agent.NewClient(sock).Signers()
		if err != nil {
			return cfg, fmt.Errorf("failed to get SSH agent signers: %w", err)
		}

		for _, signer := range signers {
			_, err := signer.Sign(rand.Reader, []byte("test"))
			if err == nil {
				cfg.sshSigner = signer
				break
			}
		}
	}

	if cfg.sshSigner == nil {
		cfg.authToken = os.Getenv("SERVERADMIN_TOKEN")
	}

	return cfg, nil
}
