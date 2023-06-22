package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const (
	version        = "4.9.0"
	defaultBaseUrl = "https://serveradmin.innogames.de"
)

type config struct {
	baseURL string

	apiVersion string

	sshPrivateKey string

	// deprecated API Token
	authToken string
}

func getConfig() (config, error) {
	cfg := config{
		baseURL:    defaultBaseUrl,
		apiVersion: version,
	}

	authToken, err := getAuthToken()
	if err != nil {
		return cfg, err
	}

	cfg.authToken = authToken

	return cfg, nil
}

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
