```
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║   ██╗  ██╗ ◆  ████████╗██╗   ██╗███╗   ███╗██╗   ██╗███████╗            ║
║   ╚██╗██╔╝ ◆  ╚══██╔══╝╚██╗ ██╔╝████╗ ████║██║   ██║██╔════╝            ║
║    ╚███╔╝  ◆     ██║    ╚████╔╝ ██╔████╔██║██║   ██║███████╗             ║
║    ██╔██╗  ◆     ██║     ╚██╔╝  ██║╚██╔╝██║██║   ██║╚════██║             ║
║   ██╔╝ ██╗ ◆     ██║      ██║   ██║ ╚═╝ ██║╚██████╔╝███████║             ║
║   ╚═╝  ╚═╝ ◆     ╚═╝      ╚═╝   ╚═╝     ╚═╝ ╚═════╝ ╚══════╝            ║
║                                                                           ║
║      ☠  PRO VERSION  ☠                           ⚡ v4.0.0 ⚡            ║
╚═══════════════════════════════════════════════════════════════════════════╝
```

> **x-tymus** — Advanced man-in-the-middle phishing framework with StealthAI bot detection, proxy rotation, GoPhish integration, and Playwright browser automation.

**Contact / Order:** https://t.me/x-tymus

---

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Command Reference](#command-reference)
- [Configuration Reference](#configuration-reference)
- [StealthAI](#stealthai)
- [GoPhish Integration](#gophish-integration)
- [Proxy System](#proxy-system)
- [Blacklist & Bot Protection](#blacklist--bot-protection)
- [Admin API](#admin-api)
- [Playwright Automation](#playwright-automation)
- [CAPTCHA Support](#captcha-support)
- [Directory Structure](#directory-structure)
- [Build](#build)

---

## Features

### Phishing Engine
- YAML-based phishlet templates with full customisation
- Multi-domain proxy hosting — map subdomains to legitimate targets
- Automatic credential capture (username, password, custom POST fields)
- Cookie and session token interception across multiple domains
- HTTP header and response body token extraction
- JavaScript injection into proxied pages with trigger support
- HTTP response interception — return custom bodies and MIME types
- Force-POST injection — silently inject form fields (e.g. "Remember me")
- Parent/child phishlet templates with dynamic parameter substitution `{param}`
- Subdomain content substitution (`sub_filters`)
- Per-phishlet `unauth_url` override

### Lure System
- Per-lure custom hostname, path, and redirect URL
- User-Agent regex filtering per lure
- OpenGraph metadata per lure (title, description, image, URL)
- HTML redirector pages
- Lure pause/unpause with duration (`1d2h3m`)
- Custom info labels per lure
- Batch parameter import from TXT / CSV / JSON
- Batch URL export to TXT / CSV / JSON

### Session & Credential Capture
- Real-time credential logging with timestamps
- Full cookie/token dump per session
- Custom field capture beyond standard credentials
- Remote IP, User-Agent, and landing URL per session
- Session range deletion (`1-5,7,10-15`) and bulk delete
- Session export to JSON or CSV

### StealthAI Bot Detection
- Python ML service (LSTM + Isolation Forest + IOC scoring)
- Three-tier scoring: 60% behaviour + 30% anomaly + 10% threat intel
- Score > 0.85 → redirect to Google (bot)
- Score 0.5–0.85 → redirect to Bing (suspicious)
- Score < 0.5 → allow through
- Runs as persistent HTTP server on `127.0.0.1:5001`
- Graceful fallback if ML libraries unavailable

### Bot Blocking
- Built-in detection for GoogleBot, BingBot, BaiduSpider, Yandex, DuckDuckBot, Yahoo Slurp, FacebookExternalHit, TwitterBot, LinkedInBot, AdsBot-Google, AppleBot
- Custom User-Agent regex blocklist
- Automatic IP blacklisting on bot detection
- IP feed integration with configurable update interval

### Proxy System
- HTTP, HTTPS, SOCKS5, SOCKS5H upstream proxy support
- Proxy list auto-loaded from `core/proxylist.txt`
- Proxy rotation (round-robin)
- Per-proxy connection validation at startup
- GoPhish requests routed through proxy

### Blacklist & IP Filtering
- Individual IP and CIDR range blacklisting
- Persistent storage with audit log
- Whitelist CIDRs to prevent false positives
- ASN-based whitelist with configurable lookup API
- Four modes: `all`, `unauth`, `noadd`, `off`
- Admin HTTP API for runtime management

### TLS / Certificate Management
- Let's Encrypt AutoCert with automatic renewal (`certmagic`)
- Self-signed CA + per-site certificates (developer mode)
- Custom certificate loading from `~/.x-tymus/crt/sites/<hostname>/`
- Supports `fullchain.pem` / `privkey.pem` and `.pem`/`.crt` + `.key` pairs
- `config autocert on/off` toggle

### GoPhish Integration
- Email open tracking
- Link click tracking
- Credential submission reporting
- Proxy-aware API calls
- Connection test command

### Playwright Browser Automation
- Headless Chrome automation
- Human-like typing simulation (character-by-character)
- Cookie collection and serialisation
- Anti-automation detection bypass flags
- JavaScript injection

### CAPTCHA Support
- Cloudflare Turnstile (site key + private key)
- Google reCAPTCHA v2/v3 (site key + private key)

### DNS
- Internal DNS resolver for A and CNAME records
- Configurable DNS port (default 53)
- Custom DNS entries via `config dnsentry`

### Notifications
- Telegram webhook — instant alert on credential capture
- Configurable bot token and chat ID

### Terminal
- Interactive CLI with command history
- Tab-completion for all commands and arguments
- Coloured output with status indicators
- ASCII table formatting for sessions, phishlets, lures

---

## Requirements

- Linux (recommended) or macOS
- Go 1.22+
- Python 3.9+ (for StealthAI)
- git

---

## Installation

```bash
# Clone / copy the project
cd /root/x-tymus

# Install Go dependencies
go mod download

# Build
go build -o build/x-tymus .

# Set up StealthAI (optional but recommended)
cd ai
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

Alternatively use `make`:

```bash
make        # builds to ./build/x-tymus
make clean  # removes build artifact
```

**Windows:**

```bat
build.bat          :: build only
build_run.bat      :: build + run in developer mode
```

---

## Quick Start

```bash
# Start StealthAI service (background)
cd ai && source .venv/bin/activate
nohup python3 stealth_ai_server.py &> /tmp/stealthai.log &

# Run x-tymus
cd /root/x-tymus
./build/x-tymus -p ./phishlets -t ./redirectors

# Developer mode (self-signed certs, no Let's Encrypt)
./build/x-tymus -p ./phishlets -developer -debug
```

**Minimum setup inside the terminal:**

```
config domain yourdomain.com
config ipv4 external 1.2.3.4
phishlets enable <phishlet>
lures create <phishlet>
lures get-url 0
```

---

## Command Reference

### General

| Command | Description |
|---|---|
| `help` | Show all commands |
| `help <cmd>` | Show help for specific command |
| `clear` | Clear screen |
| `exit` / `quit` / `q` | Exit |

### Config

| Command | Description |
|---|---|
| `config domain <domain>` | Set base phishing domain |
| `config ipv4 external <ip>` | Set external IPv4 address |
| `config ipv4 bind <ip>` | Set bind IPv4 address |
| `config https_port <port>` | HTTPS port (default 443) |
| `config dns_port <port>` | DNS port (default 53) |
| `config unauth_url <url>` | Redirect for unauthorized requests |
| `config autocert on\|off` | Enable/disable Let's Encrypt |
| `config dnsentry <name> <type> <value>` | Add custom DNS A/CNAME entry |
| `config stealthai on\|off` | Enable/disable StealthAI |
| `config webhook_telegram <token> <chat_id>` | Telegram notifications |
| `config turnstile_sitekey <key>` | Cloudflare Turnstile site key |
| `config turnstile_privkey <key>` | Cloudflare Turnstile private key |
| `config recaptcha_sitekey <key>` | Google reCAPTCHA site key |
| `config recaptcha_privkey <key>` | Google reCAPTCHA private key |

### Phishlets

| Command | Description |
|---|---|
| `phishlets` | List all phishlets |
| `phishlets <name>` | Show phishlet details |
| `phishlets enable <name>` | Enable phishlet (requests cert) |
| `phishlets disable <name>` | Disable phishlet |
| `phishlets hide <name>` | Hide phishing page |
| `phishlets unhide <name>` | Unhide phishing page |
| `phishlets hostname <name> <hostname>` | Set phishlet hostname |
| `phishlets unauth_url <name> <url>` | Override redirect for this phishlet |
| `phishlets create <template> <child> [k=v ...]` | Create child phishlet |
| `phishlets delete <name>` | Delete child phishlet |
| `phishlets get-hosts <name>` | Print /etc/hosts entries for local testing |

### Lures

| Command | Description |
|---|---|
| `lures` | List all lures |
| `lures <id>` | Show lure details |
| `lures create <phishlet>` | Create new lure |
| `lures delete <id\|range\|all>` | Delete lure(s) |
| `lures get-url <id> [params]` | Generate phishing URL |
| `lures get-url <id> import <file> export <file> <format>` | Batch URL generation |
| `lures edit <id> hostname <hostname>` | Custom hostname |
| `lures edit <id> path <path>` | Custom URL path |
| `lures edit <id> redirect_url <url>` | Post-capture redirect |
| `lures edit <id> redirector <dir>` | HTML redirector page |
| `lures edit <id> phishlet <name>` | Change phishlet |
| `lures edit <id> info <text>` | Campaign label |
| `lures edit <id> ua_filter <regex>` | User-Agent filter |
| `lures edit <id> og_title <title>` | OpenGraph title |
| `lures edit <id> og_desc <desc>` | OpenGraph description |
| `lures edit <id> og_image <url>` | OpenGraph image URL |
| `lures edit <id> og_url <url>` | OpenGraph URL |
| `lures pause <id> <duration>` | Pause lure (e.g. `1d2h30m`) |
| `lures unpause <id>` | Resume paused lure |

### Sessions

| Command | Description |
|---|---|
| `sessions` | List all captured sessions |
| `sessions <id>` | Full session detail (creds, tokens, cookies) |
| `sessions delete <id\|range\|all>` | Delete session(s) |
| `sessions export <id\|all> <json\|csv>` | Export sessions to file |

### Proxy

| Command | Description |
|---|---|
| `proxy` | Show proxy configuration |
| `proxy enable\|disable` | Toggle proxy usage |
| `proxy type <http\|https\|socks5\|socks5h>` | Set proxy type |
| `proxy address <addr>` | Set proxy address |
| `proxy port <port>` | Set proxy port |
| `proxy username <user>` | Proxy auth username |
| `proxy password <pass>` | Proxy auth password |
| `proxy add <type> <addr> <port> [user] [pass]` | Add proxy to rotation list |
| `proxy rotate` | Rotate to next proxy |

### Blacklist

| Command | Description |
|---|---|
| `blacklist all` | Block all requests |
| `blacklist unauth` | Block unauthorized requests only |
| `blacklist noadd` | Block but don't persist |
| `blacklist off` | Disable blacklist |

### GoPhish

| Command | Description |
|---|---|
| `config gophish admin_url <url>` | GoPhish admin URL |
| `config gophish api_key <key>` | GoPhish API key |
| `config gophish insecure <true\|false>` | Skip TLS verify |
| `config gophish test` | Test connection |

---

## Configuration Reference

Default config: `~/.x-tymus/config.json`

```json
{
  "general": {
    "domain": "yourdomain.com",
    "external_ipv4": "1.2.3.4",
    "bind_ipv4": "0.0.0.0",
    "https_port": 443,
    "dns_port": 53,
    "unauth_url": "https://www.google.com",
    "autocert": true
  },
  "blacklist": {
    "mode": "unauth",
    "ua_regex": ["(?i)googlebot|bingbot|bot|crawler|selenium|headless"],
    "feeds": ["https://example.com/badips.txt"],
    "feed_interval": 3600,
    "whitelist": ["127.0.0.1/32"],
    "enable_asn_lookup": false,
    "asn_lookup_url": "",
    "asn_whitelist": []
  },
  "stealthai": true,
  "gophish": {
    "admin_url": "",
    "api_key": "",
    "insecure": false
  },
  "turnstile_sitekey": "",
  "turnstile_privkey": "",
  "recaptcha_sitekey": "",
  "recaptcha_privkey": ""
}
```

---

## StealthAI

StealthAI is a Python ML service that scores each incoming request for bot-like behaviour.

**Start the service:**

```bash
cd ai
source .venv/bin/activate
nohup python3 stealth_ai_server.py &> /tmp/stealthai.log &
```

**Scoring model:**

| Layer | Weight | Method |
|---|---|---|
| Behaviour analysis | 60% | LSTM recurrent model |
| Anomaly detection | 30% | Isolation Forest |
| Threat intel IOC | 10% | Known bad indicator matching |

**Score thresholds:**

| Score | Action |
|---|---|
| > 0.85 | Redirect → google.com (bot) |
| 0.5 – 0.85 | Redirect → bing.com (suspicious) |
| < 0.5 | Allow through |

**Enable/disable:**

```
config stealthai on
config stealthai off
```

---

## GoPhish Integration

```
config gophish admin_url https://127.0.0.1:3333
config gophish api_key YOUR_API_KEY
config gophish insecure true
config gophish test
```

Events automatically reported to GoPhish:
- Email opened
- Link clicked
- Credentials submitted

---

## Proxy System

**Load from file** — add proxies to `core/proxylist.txt` (one per line):

```
socks5://user:pass@1.2.3.4:1080
http://1.2.3.4:8080
```

Proxies are validated at startup. Valid proxies are shown in green, failed in red. Rotation is automatic.

**Runtime:**

```
proxy add socks5 1.2.3.4 1080 user pass
proxy rotate
proxy enable
```

---

## Blacklist & Bot Protection

```
blacklist unauth        # recommended default
```

**IP Feed auto-update:**

```json
"feeds": ["https://example.com/feed.txt"],
"feed_interval": 3600
```

**Runtime blacklist file:** `~/.x-tymus/blacklist.txt`  
**Audit log:** `~/.x-tymus/blacklist.txt.audit.log`

---

## Admin API

Exposed on port 80. Restrict to localhost in production.

```bash
# List
curl http://127.0.0.1:80/admin/blacklist

# Add
curl -X POST -H "Content-Type: application/json" \
  -d '{"ip":"1.2.3.4"}' http://127.0.0.1:80/admin/blacklist

# Remove
curl -X DELETE -H "Content-Type: application/json" \
  -d '{"ip":"1.2.3.4"}' http://127.0.0.1:80/admin/blacklist

# Flush all
curl -X POST http://127.0.0.1:80/admin/blacklist/flush
```

---

## Playwright Automation

Built-in headless Chrome automation for automated session collection:
- Human-like typing simulation
- Cookie serialisation
- Anti-detection browser flags
- JavaScript injection support

---

## CAPTCHA Support

```
config turnstile_sitekey <key>
config turnstile_privkey <key>

config recaptcha_sitekey <key>
config recaptcha_privkey <key>
```

---

## Directory Structure

```
~/.x-tymus/
├── config.json              # main configuration
├── data.db                  # session database
├── blacklist.txt            # persistent IP blacklist
├── blacklist.txt.audit.log  # audit log
└── crt/
    └── sites/
        └── <hostname>/
            ├── fullchain.pem
            └── privkey.pem

./
├── phishlets/               # phishlet YAML files
├── redirectors/             # HTML redirector templates
├── ai/                      # StealthAI Python service
│   ├── stealth_ai_server.py
│   └── stealth_ai.py
└── build/
    └── x-tymus              # compiled binary
```

---

## Build

```bash
# Linux / macOS
make

# Manual
go build -o build/x-tymus .

# Windows
build.bat
```

**Flags:**

| Flag | Description |
|---|---|
| `-p <path>` | Phishlets directory |
| `-t <path>` | Redirectors directory |
| `-c <path>` | Config directory |
| `-debug` | Enable debug logging |
| `-developer` | Self-signed certs, skip Let's Encrypt |
| `-v` | Show version |

---

© 2025 x-tymus — https://t.me/x-tymus
# project-x
