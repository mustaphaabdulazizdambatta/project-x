package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/log"
)

// GetASNForIP queries a provided ASN lookup endpoint or attempts a simple
// heuristic to extract ASN. The config-provided URL is used when non-empty.
// The function returns the ASN number on success or an error.
func GetASNForIP(ip string, lookupURL string, timeout time.Duration) (int, error) {
	client := &http.Client{Timeout: timeout}
	// Use proxy if configured
	if GlobalConfig != nil {
		currentProxy := GlobalConfig.GetCurrentProxy()
		if currentProxy != nil {
			proxyURL := &url.URL{
				Scheme: currentProxy.Type,
				Host:   currentProxy.Address + ":" + strconv.Itoa(currentProxy.Port),
			}
			if currentProxy.Username != "" || currentProxy.Password != "" {
				proxyURL.User = url.UserPassword(currentProxy.Username, currentProxy.Password)
			}
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
		GlobalConfig.RotateProxy() // Rotate after use
	}
	if lookupURL != "" {
		// assume the URL accepts a query parameter 'ip'
		u, err := url.Parse(lookupURL)
		if err != nil {
			return 0, err
		}
		q := u.Query()
		q.Set("ip", ip)
		u.RawQuery = q.Encode()
		resp, err := client.Get(u.String())
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		// try to parse common responses
		// JSON: {"asn": 1234}
		var j map[string]interface{}
		if err := json.Unmarshal(body, &j); err == nil {
			if v, ok := j["asn"]; ok {
				switch t := v.(type) {
				case float64:
					return int(t), nil
				case string:
					n, _ := strconv.Atoi(t)
					return n, nil
				}
			}
		}
		// plain text number
		s := strings.TrimSpace(string(body))
		n, err := strconv.Atoi(s)
		if err == nil {
			return n, nil
		}
		return 0, fmt.Errorf("asn lookup: unrecognized response: %s", s)
	}

	// No lookup URL provided; attempt a minimal Team Cymru-like HTTP lookup
	// using whois.cymru.com is not HTTP - so we bail out here.
	log.Debug("asn: no lookup URL provided, skipping ASN lookup for %s", ip)
	return 0, fmt.Errorf("asn lookup not configured")
}
