package core

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strconv"
	"github.com/elazarl/goproxy"
	"github.com/x-tymus/x-tymus/log"
)

type HttpProxy struct {
	BindIP    string
	Port      int
	Cfg       *Config
	cfg       *Config
	gophish   *GoPhishConfig
	bl        *BlacklistConfig
	developer bool
	crt_db    *CertDb
	proxy     *goproxy.ProxyHttpServer
}

func NewHttpProxy(bindIP string, port int, cfg *Config) *HttpProxy {
	hp := &HttpProxy{
		BindIP:    bindIP,
		Port:      port,
		Cfg:       cfg,
		cfg:       cfg,
		gophish:   cfg.gophishConfig,
		bl:        cfg.blacklistConfig,
		developer: false,
		crt_db:    cfg.CrtDb,
	}
	hp.proxy = goproxy.NewProxyHttpServer()
	hp.proxy.Verbose = false
	hp.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	if hp.crt_db != nil {
		// Assuming CertDb has a method to get CA cert; adjust as per actual implementation
		hp.proxy.CertStore = hp.crt_db
	}
	hp.updateTransport()
	return hp
}

func (hp *HttpProxy) Start() {
	hp.updateTransport()
	server := &http.Server{
		Addr:    hp.BindIP + ":" + strconv.Itoa(hp.Port),
		Handler: hp.proxy,
	}
	log.Info("starting HTTPS proxy on %s", server.Addr)
	server.ListenAndServe()
}

func (hp *HttpProxy) setProxy(enabled bool, ptype, address string, port int, username, password string) error {
	hp.Cfg.EnableProxy(enabled)
	hp.Cfg.SetProxyType(ptype)
	hp.Cfg.SetProxyAddress(address)
	hp.Cfg.SetProxyPort(port)
	hp.Cfg.SetProxyUsername(username)
	hp.Cfg.SetProxyPassword(password)
	hp.updateTransport()
	return nil
}

func (hp *HttpProxy) updateTransport() {
	currentProxy := hp.Cfg.GetCurrentProxy()
	if currentProxy != nil {
		proxyURL := &url.URL{
			Scheme: currentProxy.Type,
			Host:   currentProxy.Address + ":" + strconv.Itoa(currentProxy.Port),
		}
		if currentProxy.Username != "" || currentProxy.Password != "" {
			proxyURL.User = url.UserPassword(currentProxy.Username, currentProxy.Password)
		}
		hp.proxy.Tr = &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		hp.proxy.Tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
}
