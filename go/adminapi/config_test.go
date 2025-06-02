package adminapi

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// clear env to not have configs from host here
	os.Clearenv()

	// make a test without SERVERADMIN_BASE_URL set
	_, err := loadConfig()
	assert.Error(t, err, "env var SERVERADMIN_BASE_URL not set")

	// spawn mocked serveradmin server
	server := httptest.NewServer(nil)
	defer server.Close()
	_ = os.Setenv("SERVERADMIN_BASE_URL", server.URL)

	t.Run("load static token", func(t *testing.T) {
		_ = os.Setenv("SERVERADMIN_TOKEN", "jolo")
		cfg, err := loadConfig()

		assert.NoError(t, err)
		assert.Nil(t, cfg.sshSigner)
		assert.Equal(t, "jolo", string(cfg.authToken))
	})

	t.Run("load valid private key", func(t *testing.T) {
		_ = os.Setenv("SERVERADMIN_KEY_PATH", "testdata/test.key")
		cfg, err := loadConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Empty(t, cfg.authToken)
	})

	t.Run("load invalid private Key", func(t *testing.T) {
		_ = os.Setenv("SERVERADMIN_KEY_PATH", "testdata/nope.key")
		_, err := loadConfig()

		assert.Error(t, err, "failed to read private key from testdata/nope.key: open testdata/nope.key: no such file or directory")
	})
}
