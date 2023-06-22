package main

import (
	"fmt"
	"testing"
)

func TestGetConfig(t *testing.T) {
	cfg, err := getConfig()

	fmt.Println(cfg.sshSigner)
	fmt.Println(err)
}
