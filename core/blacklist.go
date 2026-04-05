package core

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/log"
)

type BlockIP struct {
	ipv4 net.IP
	mask *net.IPNet
}

type Blacklist struct {
	ips        map[string]*BlockIP
	masks      []*BlockIP
	configPath string
	verbose    bool
}

// GlobalBlacklist is the runtime-loaded blacklist instance used by the server.
var GlobalBlacklist *Blacklist

func NewBlacklist(path string) (*Blacklist, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bl := &Blacklist{
		ips:        make(map[string]*BlockIP),
		configPath: path,
		verbose:    true,
	}

	fs := bufio.NewScanner(f)
	fs.Split(bufio.ScanLines)

	for fs.Scan() {
		l := fs.Text()
		// remove comments
		if n := strings.Index(l, ";"); n > -1 {
			l = l[:n]
		}
		l = strings.Trim(l, " ")

		if len(l) > 0 {
			if strings.Contains(l, "/") {
				ipv4, mask, err := net.ParseCIDR(l)
				if err == nil {
					bl.masks = append(bl.masks, &BlockIP{ipv4: ipv4, mask: mask})
				} else {
					log.Error("blacklist: invalid ip/mask address: %s", l)
				}
			} else {
				ipv4 := net.ParseIP(l)
				if ipv4 != nil {
					bl.ips[ipv4.String()] = &BlockIP{ipv4: ipv4, mask: nil}
				} else {
					log.Error("blacklist: invalid ip address: %s", l)
				}
			}
		}
	}

	log.Info("blacklist: loaded %d ip addresses and %d ip masks", len(bl.ips), len(bl.masks))
	return bl, nil
}

func (bl *Blacklist) GetStats() (int, int) {
	return len(bl.ips), len(bl.masks)
}

func (bl *Blacklist) AddIP(ip string) error {
	if bl.IsBlacklisted(ip) {
		return nil
	}

	ipv4 := net.ParseIP(ip)
	if ipv4 != nil {
		bl.ips[ipv4.String()] = &BlockIP{ipv4: ipv4, mask: nil}
	} else {
		return fmt.Errorf("invalid ip address: %s", ip)
	}

	// write to file
	f, err := os.OpenFile(bl.configPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(ipv4.String() + "\n")
	if err != nil {
		return err
	}

	// audit log
	if bl.configPath != "" {
		auditPath := bl.configPath + ".audit.log"
		af, err := os.OpenFile(auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			defer af.Close()
			af.WriteString(fmt.Sprintf("%s added\n", ipv4.String()))
		}
	}

	return nil
}

func (bl *Blacklist) IsBlacklisted(ip string) bool {
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil {
		return false
	}

	if _, ok := bl.ips[ip]; ok {
		return true
	}
	for _, m := range bl.masks {
		if m.mask != nil && m.mask.Contains(ipv4) {
			return true
		}
	}
	return false
}

func (bl *Blacklist) SetVerbose(verbose bool) {
	bl.verbose = verbose
}

func (bl *Blacklist) IsVerbose() bool {
	return bl.verbose
}

func (bl *Blacklist) IsWhitelisted(ip string) bool {
	if ip == "127.0.0.1" {
		return true
	}
	return false
}

// IsIPPermitted checks whether an IP should be treated as whitelisted/allowed
// based on configured CIDR whitelist and optional ASN whitelist via config.
// Returns true when the IP is allowed and should NOT be added to the blacklist.
func IsIPPermitted(ip string, cfg *Config) bool {
	if ip == "127.0.0.1" {
		return true
	}
	if cfg == nil || cfg.blacklistConfig == nil {
		return false
	}
	// check CIDR whitelist
	if IsIPInCIDRs(ip, cfg.blacklistConfig.Whitelist) {
		return true
	}
	// optional ASN check
	if cfg.blacklistConfig.EnableASNLookup {
		// use a short timeout for ASN lookups
		asn, err := GetASNForIP(ip, cfg.blacklistConfig.ASNLookupURL, 5*time.Second)
		if err == nil {
			for _, a := range cfg.blacklistConfig.ASNWhitelist {
				if a == asn {
					return true
				}
			}
		}
	}
	return false
}

// IsIPInCIDRs checks whether ip belongs to any of the provided CIDR strings.
func IsIPInCIDRs(ip string, cidrs []string) bool {
	if ip == "" || len(cidrs) == 0 {
		return false
	}
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil {
		return false
	}
	for _, c := range cidrs {
		_, netw, err := net.ParseCIDR(c)
		if err != nil {
			continue
		}
		if netw.Contains(ipv4) {
			return true
		}
	}
	return false
}

// RemoveIP removes an IP from in-memory blacklist and rewrites the config file.
func (bl *Blacklist) RemoveIP(ip string) error {
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil {
		return fmt.Errorf("invalid ip address: %s", ip)
	}
	delete(bl.ips, ipv4.String())

	// rewrite file without this ip
	f, err := os.OpenFile(bl.configPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for k := range bl.ips {
		f.WriteString(k + "\n")
	}
	return nil
}

// Flush clears the blacklist (in-memory and on-disk)
func (bl *Blacklist) Flush() error {
	bl.ips = make(map[string]*BlockIP)
	bl.masks = []*BlockIP{}
	f, err := os.OpenFile(bl.configPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}
