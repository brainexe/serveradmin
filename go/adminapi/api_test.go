package adminapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFakeServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := io.ReadAll(r.Body)
		// todo: assert signature etc...create more useful test cases

		expectedRequest := `{"filters":{"hostname":{"Any":[{"Regexp":"test.foo.local"},{"Regexp":".*\\.bar.local"}]}},"restrict":["hostname","object_id"]}`
		assert.Equal(t, expectedRequest, string(req))

		resp := `{"status": "success", "result": [{"object_id": 483903, "hostname": "foo.bar.local"}]}`

		w.WriteHeader(200)
		_, _ = w.Write([]byte(resp))
	}))
	defer server.Close()

	os.Clearenv()
	_ = os.Setenv("SERVERADMIN_TOKEN", "1234567890")
	_ = os.Setenv("SERVERADMIN_BASE_URL", server.URL)

	query := NewQuery(Filters{
		"hostname": Any(Regexp("test.foo.local"), Regexp(".*\\.bar.local")),
	})
	query.SetAttributes([]string{"hostname"})

	servers, err := query.All()
	assert.NoError(t, err)
	assert.Len(t, servers, 1)

	object := servers[0]
	assert.Equal(t, "foo.bar.local", object.Get("hostname"))
	assert.Equal(t, "foo.bar.local", object.GetString("hostname"))
	assert.Equal(t, 483903, object.Get("object_id"))
	assert.Equal(t, 483903, object.ObjectId())
	assert.Equal(t, nil, object.GetString("object_id"))
	assert.Nil(t, object.Get("nope"))
	assert.Nil(t, object.GetString("nope"))

	one, err := query.One()
	assert.NoError(t, err)
	assert.Equal(t, 483903, one.Get("object_id"))
}

// just some simple example tests, e2e tests might make much more sense here for full coverage
func TestAppId(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected string
	}{
		{
			input:    []byte("1234567898"),
			expected: "d396f232a5ca1f7a0ad8f1b59975515123780553",
		},
	}

	for _, testCase := range testCases {
		actual := calcAppID(testCase.input)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestSecurityToken(t *testing.T) {
	testCases := []struct {
		apiKey   []byte
		message  string
		expected string
	}{
		{
			apiKey:   []byte("1234567898"),
			message:  "",
			expected: "4199b91c6f92f3e1d29f88a5f67973ad8aaec5b5",
		},
		{
			apiKey:   []byte("1234567898"),
			message:  "foobar",
			expected: "e17ba31a1a664617653869db8289f92a49213e7b",
		},
	}

	now := int64(123456789)
	for _, testCase := range testCases {
		actual := calcSecurityToken(testCase.apiKey, now, []byte(testCase.message))
		assert.Equal(t, testCase.expected, actual)
	}
}

func BenchmarkCalcSecurityToken(b *testing.B) {
	now := int64(123456789)
	message := []byte("foobar")
	authToken := []byte("1234567898")
	for b.Loop() {
		calcSecurityToken(authToken, now, message)
	}
}
