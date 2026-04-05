package core

import (
	"bufio"
	"net/http"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/log"
)

// UpdateBlacklistFromFeeds fetches newline-separated IPs from the provided URLs
// and adds them to the GlobalBlacklist. It returns number of IPs added and error if any.
func UpdateBlacklistFromFeeds(urls []string, timeout time.Duration) (int, error) {
	added := 0
	client := &http.Client{Timeout: timeout}
	for _, u := range urls {
		resp, err := client.Get(u)
		if err != nil {
			log.Error("ipfeed: failed to fetch %s: %v", u, err)
			continue
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// support comment lines starting with '#'
			if strings.HasPrefix(line, "#") {
				continue
			}
			// strip port if present
			ip := line
			if idx := strings.LastIndex(ip, ":"); idx > -1 {
				ip = ip[:idx]
			}
			if GlobalBlacklist != nil {
				// consult config-based whitelist/ASN before adding
				if IsIPPermitted(ip, GlobalConfig) {
					if GlobalBlacklist.IsVerbose() {
						log.Info("ipfeed: skipping whitelisted IP %s from feed %s", ip, u)
					}
					continue
				}
				if err := GlobalBlacklist.AddIP(ip); err == nil {
					added++
				}
			}
		}
		resp.Body.Close()
	}
	return added, nil
}
