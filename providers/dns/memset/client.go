package memset

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-acme/lego/v3/challenge/dns01"
)

const defaultBaseURL = "https://api.memset.com"

// record represents a response from Memset's API after making a TXT record
type record struct {
	ID       string `json:"id,omitempty"`
	ZoneID   string `json:"zone_id,omitempty"`
	Type     string `json:"type,omitempty"`
	Record   string `json:"record,omitempty"`
	Address  string `json:"address,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Relative bool   `json:"relative,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

type errorResponse struct {
	ErrorType string `json:"error_type"`
	ErrorCode string `json:"error_code"`
	Error     string `json:"error"`
}

type jobStatus struct {
	ID       string `json:"id"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`
	Service  string `json:"service,omitempty"`
	Finished bool   `json:"finished,omitempty"`
	Error    bool   `json:"error,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

type apiKey struct {
	APIKey string `json:"api_key"`
}

type zoneDomain struct {
	ID     string `json:"zone_id,omitempty"`
	Domain string `json:"domain"`
	APIKey string `json:"api_key,omitempty"`
}

func (d *DNSProvider) removeTxtRecord(recordID string) error {
	recordToRemove := record{
		ID:     recordID,
		APIKey: d.config.AuthToken,
	}
	reqURL := fmt.Sprintf("%s/v1/json/dns.zone_record_delete", d.config.BaseURL)
	recordData, err := json.Marshal(recordToRemove)
	if err != nil {
		return err
	}
	removedRecord := record{}
	return d.newRequest(reqURL, recordData, &removedRecord)
}

func (d *DNSProvider) addTxtRecord(fqdn, value string) (*record, error) {
	authZone, err := dns01.FindZoneByFqdn(dns01.ToFqdn(fqdn))
	if err != nil {
		return nil, fmt.Errorf("could not determine zone for domain: '%s'. %s", fqdn, err)
	}

	// Lookup the ID of the zone domain
	zoneLookup := zoneDomain{
		APIKey: d.config.AuthToken,
		Domain: dns01.UnFqdn(authZone),
	}
	queryData, err := json.Marshal(zoneLookup)
	if err != nil {
		return nil, err
	}
	zone := zoneDomain{}
	reqURL := fmt.Sprintf("%s/v1/json/dns.zone_domain_info", d.config.BaseURL)
	err = d.newRequest(reqURL, queryData, &zone)
	if err != nil {
		return nil, err
	}

	// Create record
	host := dns01.UnFqdn(strings.TrimSuffix(dns01.UnFqdn(fqdn), dns01.UnFqdn(authZone)))
	newRecord := record{
		ZoneID:  zone.ID,
		Type:    "TXT",
		Record:  host,
		Address: value,
		TTL:     d.config.TTL,
		APIKey:  d.config.AuthToken,
	}
	reqURL = fmt.Sprintf("%s/v1/json/dns.zone_record_create", d.config.BaseURL)
	recordData, err := json.Marshal(newRecord)
	if err != nil {
		return nil, err
	}
	newTxtRecord := &record{}
	err = d.newRequest(reqURL, recordData, newTxtRecord)
	if err != nil {
		return nil, err
	}

	// Trigger a DNS reload to put the record live
	reqURL = fmt.Sprintf("%s/v1/json/dns.reload", d.config.BaseURL)
	apiKeyObj := apiKey{APIKey: d.config.AuthToken}
	apiKeyData, err := json.Marshal(apiKeyObj)
	if err != nil {
		return nil, err
	}
	err = d.newRequest(reqURL, apiKeyData, &jobStatus{})
	if err != nil {
		return nil, err
	}

	// The reload may not have completed yet, but rather than wait indefinitely return the new record
	return newTxtRecord, nil
}

func (d *DNSProvider) newRequest(reqURL string, data []byte, v interface{}) error {
	resp, err := http.PostForm(reqURL, url.Values{"parameters": {string(data)}})
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &v)
	if err != nil {
		// This might be an error response. Attempt to parse it.
		apiError := errorResponse{}
		apiErr := json.Unmarshal(content, &apiError)
		if apiErr == nil {
			return fmt.Errorf("unexpected response from memset: %s - %s", apiError.ErrorType, apiError.Error)
		}
	}
	return err
}
