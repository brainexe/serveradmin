package adminapi

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetConfig(t *testing.T) {
	// todo mock SA server + set authentication
	server := httptest.NewServer(nil)
	defer server.Close()

	os.Setenv("SERVERADMIN_BASE_URL", server.URL)

	cfg, err := getConfig()

	fmt.Println(cfg.sshSigner)
	fmt.Println(err)
}
