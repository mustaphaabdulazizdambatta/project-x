package core

import (
	"net"
	"net/http"
)

// Cloudflare's published IP ranges (https://www.cloudflare.com/ips/).
// Requests arriving from these IPs are legitimate Cloudflare edge nodes —
// the real visitor IP is carried in the CF-Connecting-IP header.
var cloudflareIPv4 = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
}

var cloudflareIPv6 = []string{
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

var cfNets []*net.IPNet

func init() {
	for _, cidr := range append(cloudflareIPv4, cloudflareIPv6...) {
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			cfNets = append(cfNets, n)
		}
	}
}

// IsCloudflareIP returns true if the IP belongs to a Cloudflare edge node.
func IsCloudflareIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range cfNets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// GetRealIP returns the true visitor IP from a request.
// When Cloudflare mode is on and the request arrives from a CF edge node,
// it uses the CF-Connecting-IP header. Otherwise it falls back to the
// standard proxy header chain and finally the socket address.
func GetRealIP(req *http.Request, cfMode bool) string {
	socketIP := req.RemoteAddr
	if host, _, err := net.SplitHostPort(socketIP); err == nil {
		socketIP = host
	}

	if cfMode && IsCloudflareIP(socketIP) {
		if cf := req.Header.Get("CF-Connecting-IP"); cf != "" {
			return cf
		}
	}

	// Standard proxy headers, in priority order
	for _, h := range []string{"X-Forwarded-For", "X-Real-IP", "True-Client-IP", "Client-IP"} {
		if v := req.Header.Get(h); v != "" {
			// X-Forwarded-For can be a comma-separated list — take the first (original client)
			if idx := len(v); idx > 0 {
				for i, c := range v {
					if c == ',' {
						return v[:i]
					}
				}
			}
			return v
		}
	}

	return socketIP
}
