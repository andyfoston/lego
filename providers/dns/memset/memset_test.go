package memset

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-acme/lego/v3/platform/tester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var envTest = tester.NewEnvTest("MEMSET_AUTH_TOKEN")

func setupTest() (*DNSProvider, *http.ServeMux, func()) {
	handler := http.NewServeMux()
	server := httptest.NewServer(handler)

	config := NewDefaultConfig()
	config.AuthToken = "asdf1234"
	config.BaseURL = server.URL

	provider, err := NewDNSProviderConfig(config)
	if err != nil {
		panic(err)
	}

	return provider, handler, server.Close
}

func TestNewDNSProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		envVars  map[string]string
		expected string
	}{
		{
			desc: "success",
			envVars: map[string]string{
				"MEMSET_AUTH_TOKEN": "123",
			},
		},
		{
			desc: "missing credentials",
			envVars: map[string]string{
				"MEMSET_AUTH_TOKEN": "",
			},
			expected: "memset: some credentials information are missing: MEMSET_AUTH_TOKEN",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			defer envTest.RestoreEnv()
			envTest.ClearEnv()

			envTest.Apply(test.envVars)

			p, err := NewDNSProvider()

			if len(test.expected) == 0 {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.recordIDs)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProviderConfig(t *testing.T) {
	testCases := []struct {
		desc      string
		authToken string
		expected  string
	}{
		{
			desc:      "success",
			authToken: "123",
		},
		{
			desc:     "missing credentials",
			expected: "memset: credentials missing",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			config := NewDefaultConfig()
			config.AuthToken = test.authToken

			p, err := NewDNSProviderConfig(config)

			if len(test.expected) == 0 {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.recordIDs)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestDNSProvider_Present(t *testing.T) {
	provider, mux, tearDown := setupTest()
	defer tearDown()

	mux.HandleFunc("/v1/json/dns.zone_domain_info", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method")

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		expectedReqBody := fmt.Sprintf("parameters=%s", url.QueryEscape(`{"domain":"example.com","api_key":"asdf1234"}`))
		assert.Equal(t, expectedReqBody, string(reqBody))

		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, `{
        "zone_id": "123456",
				"domain": "example.com"
		}`)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/v1/json/dns.zone_record_create", func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		expectedReqBody := fmt.Sprintf("parameters=%s", url.QueryEscape(`{"zone_id":"123456","type":"TXT","record":"_acme-challenge","address":"w6uP8Tcg6K2QR905Rms8iXTlksL6OD1KOWBxTK7wxPI","ttl":300,"api_key":"asdf1234"}`))
		assert.Equal(t, expectedReqBody, string(reqBody))

		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, `{
		    "id": "abcdef",
		    "zone_id":"123456",
		    "type":"TXT",
		    "record":"_acme-challenge",
		    "address":"w6uP8Tcg6K2QR905Rms8iXTlksL6OD1KOWBxTK7wxPI",
		    "ttl":300
		}`)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/v1/json/dns.reload", func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		expectedReqBody := fmt.Sprintf("parameters=%s", url.QueryEscape(`{"api_key":"asdf1234"}`))
		assert.Equal(t, expectedReqBody, string(reqBody))

		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, `{
		    "id": "zxcvb",
		    "type":"reload",
				"status": "pending",
				"service": "dns",
				"finished": false,
				"error": false
		}`)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	err := provider.Present("example.com", "", "foobar")
	require.NoError(t, err)
}

func TestDNSProvider_CleanUp(t *testing.T) {
	provider, mux, tearDown := setupTest()
	defer tearDown()

	mux.HandleFunc("/v1/json/dns.zone_record_delete", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method")

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		expectedReqBody := fmt.Sprintf("parameters=%s", url.QueryEscape(`{"id":"1234567","api_key":"asdf1234"}`))
		assert.Equal(t, expectedReqBody, string(reqBody))

		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, `{
		    "id": "abcdef",
		    "zone_id":"123456",
		    "type":"TXT",
		    "record":"_acme-challenge",
		    "address":"w6uP8Tcg6K2QR905Rms8iXTlksL6OD1KOWBxTK7wxPI",
		    "ttl":300
		}`)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	provider.recordIDsMu.Lock()
	provider.recordIDs["_acme-challenge.example.com."] = "1234567"
	provider.recordIDsMu.Unlock()

	err := provider.CleanUp("example.com", "", "")
	require.NoError(t, err, "fail to remove TXT record")
}
