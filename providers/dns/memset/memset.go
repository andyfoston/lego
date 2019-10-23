// Package memset implements a DNS provider for solving the DNS-01 challenge using memset DNS.
package memset

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-acme/lego/v3/challenge/dns01"
	"github.com/go-acme/lego/v3/platform/config/env"
)

// Config is used to configure the creation of the DNSProvider
type Config struct {
	BaseURL            string
	AuthToken          string
	TTL                int
	PropagationTimeout time.Duration
	PollingInterval    time.Duration
	HTTPClient         *http.Client
}

// NewDefaultConfig returns a default configuration for the DNSProvider
func NewDefaultConfig() *Config {
	// Memset's API appears to silently fail if an incorrect TTL is provided. Validate it here first
	ttl := env.GetOrDefaultInt("MEMSET_TTL", 300)
	return &Config{
		BaseURL:            defaultBaseURL,
		TTL:                ttl,
		PropagationTimeout: env.GetOrDefaultSecond("MEMSET_PROPAGATION_TIMEOUT", 60*time.Second),
		PollingInterval:    env.GetOrDefaultSecond("MEMSET_POLLING_INTERVAL", 5*time.Second),
		HTTPClient: &http.Client{
			Timeout: env.GetOrDefaultSecond("MEMSET_HTTP_TIMEOUT", 30*time.Second),
		},
	}
}

// DNSProvider is an implementation of the challenge.Provider interface
// that uses Memset's REST API to manage TXT records for a domain.
type DNSProvider struct {
	config      *Config
	recordIDs   map[string]string
	recordIDsMu sync.Mutex
}

// NewDNSProvider returns a DNSProvider instance configured for Memset.
// Credentials must be passed in the environment variable:
// MEMSET_AUTH_TOKEN.
func NewDNSProvider() (*DNSProvider, error) {
	values, err := env.Get("MEMSET_AUTH_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("memset: %v", err)
	}

	config := NewDefaultConfig()
	config.AuthToken = values["MEMSET_AUTH_TOKEN"]

	return NewDNSProviderConfig(config)
}

// NewDNSProviderConfig return a DNSProvider instance configured for Memset.
func NewDNSProviderConfig(config *Config) (*DNSProvider, error) {
	if config == nil {
		return nil, errors.New("memset: the configuration of the DNS provider is nil")
	}

	if config.AuthToken == "" {
		return nil, fmt.Errorf("memset: credentials missing")
	}

	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}

	// Ensure a valid TTL is provided, because the API fails silently if the TTL is not within this range.
	switch config.TTL {
	case
		0,
		300,
		600,
		1800,
		3600,
		7200,
		10800,
		21600,
		43200,
		86400:
		break
	default:
		return nil, fmt.Errorf("memset: TTL %d is invalid", config.TTL)
	}

	return &DNSProvider{
		config:    config,
		recordIDs: make(map[string]string),
	}, nil
}

// Timeout returns the timeout and interval to use when checking for DNS propagation.
// Adjusting here to cope with spikes in propagation times.
func (d *DNSProvider) Timeout() (timeout, interval time.Duration) {
	return d.config.PropagationTimeout, d.config.PollingInterval
}

// Present creates a TXT record using the specified parameters
func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	respData, err := d.addTxtRecord(fqdn, value)
	if err != nil {
		return fmt.Errorf("memset: %v", err)
	}

	d.recordIDsMu.Lock()
	d.recordIDs[fqdn] = respData.ID
	d.recordIDsMu.Unlock()

	return nil
}

// CleanUp removes the TXT record matching the specified parameters
func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)

	// get the record's unique ID from when we created it
	d.recordIDsMu.Lock()
	recordID, ok := d.recordIDs[fqdn]
	d.recordIDsMu.Unlock()
	if !ok {
		return fmt.Errorf("memset: unknown record ID for '%s'", fqdn)
	}

	err := d.removeTxtRecord(recordID)
	if err != nil {
		return fmt.Errorf("memset: %v", err)
	}

	// Delete record ID from map
	d.recordIDsMu.Lock()
	delete(d.recordIDs, fqdn)
	d.recordIDsMu.Unlock()

	return nil
}
