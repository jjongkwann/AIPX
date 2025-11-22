package security

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// URLValidator validates URLs to prevent SSRF attacks
type URLValidator struct {
	allowedHosts   map[string]bool
	blockedCIDRs   []*net.IPNet
	allowHTTP      bool
	maxRedirects   int
	dnsCache       *DNSCache
	mu             sync.RWMutex
}

// DNSCache caches DNS lookups to prevent DNS rebinding attacks
type DNSCache struct {
	cache map[string]*dnsCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type dnsCacheEntry struct {
	ips       []net.IP
	timestamp time.Time
}

// NewURLValidator creates a new URL validator with secure defaults
func NewURLValidator() *URLValidator {
	v := &URLValidator{
		allowedHosts: map[string]bool{
			// Popular webhook/notification services
			"hooks.slack.com":          true,
			"api.telegram.org":         true,
			"discord.com":              true,
			"discordapp.com":           true,
			"api.discord.com":          true,
			"api.pagerduty.com":        true,
			"api.pushover.net":         true,
			"api.twilio.com":           true,
			"api.sendgrid.com":         true,
			"api.mailgun.net":          true,
			"api.postmarkapp.com":      true,
			"api.smtp2go.com":          true,
			"events.pagerduty.com":     true,
			"ingest.signifyd.com":      true,
			"app.datadoghq.com":        true,
			"api.datadoghq.com":        true,
			"api.opsgenie.com":         true,
			"api.victorops.com":        true,
			"api.statuspage.io":        true,
			"webhooks.mongodb-realm.com": true,
			"hooks.zapier.com":         true,
			"webhook.site":             true, // For testing only - remove in production
		},
		allowHTTP:    false,
		maxRedirects: 3,
		dnsCache: &DNSCache{
			cache: make(map[string]*dnsCacheEntry),
			ttl:   5 * time.Minute,
		},
	}

	// Block private/internal IP ranges (RFC 1918, RFC 6890)
	blockedRanges := []string{
		// IPv4 Private
		"10.0.0.0/8",      // Private network
		"172.16.0.0/12",   // Private network
		"192.168.0.0/16",  // Private network
		"127.0.0.0/8",     // Loopback
		"169.254.0.0/16",  // Link-local
		"100.64.0.0/10",   // Carrier-grade NAT
		"0.0.0.0/8",       // This host
		"224.0.0.0/4",     // Multicast
		"240.0.0.0/4",     // Reserved for future use
		"255.255.255.255/32", // Broadcast
		
		// IPv6 Private
		"::1/128",         // Loopback
		"fc00::/7",        // Unique local address
		"fe80::/10",       // Link-local
		"ff00::/8",        // Multicast
		"::/128",          // Unspecified address
		// Note: ::ffff:0:0/96 removed - it matches all IPv4 addresses when parsed by Go
		"64:ff9b::/96",    // IPv4/IPv6 translation
		"100::/64",        // Discard prefix
		"2001::/32",       // Teredo tunneling
		"2001:db8::/32",   // Documentation
	}

	for _, cidr := range blockedRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			v.blockedCIDRs = append(v.blockedCIDRs, network)
		}
	}

	return v
}

// AddAllowedHost adds a host to the allowlist
func (v *URLValidator) AddAllowedHost(host string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.allowedHosts[strings.ToLower(host)] = true
}

// RemoveAllowedHost removes a host from the allowlist
func (v *URLValidator) RemoveAllowedHost(host string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.allowedHosts, strings.ToLower(host))
}

// SetAllowHTTP sets whether HTTP URLs are allowed (default is false)
func (v *URLValidator) SetAllowHTTP(allow bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.allowHTTP = allow
}

// Validate checks if a URL is safe to request
func (v *URLValidator) Validate(ctx context.Context, rawURL string) error {
	// Basic URL validation
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// 1. Validate scheme
	if err := v.validateScheme(parsed); err != nil {
		return err
	}

	// 2. Validate host
	host := strings.ToLower(parsed.Hostname())
	if err := v.validateHost(host); err != nil {
		return err
	}

	// 3. Check against allowlist (if configured)
	if err := v.checkAllowlist(host); err != nil {
		return err
	}

	// 4. Resolve DNS and validate IP addresses
	ips, err := v.resolveDNS(ctx, host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed: %w", err)
	}

	// 5. Check all resolved IPs
	for _, ip := range ips {
		if err := v.validateIP(ip); err != nil {
			return err
		}
	}

	// 6. Validate port
	if err := v.validatePort(parsed); err != nil {
		return err
	}

	return nil
}

// ValidateWithDNSRebindingProtection validates URL with protection against DNS rebinding
func (v *URLValidator) ValidateWithDNSRebindingProtection(ctx context.Context, rawURL string) error {
	// First validation
	if err := v.Validate(ctx, rawURL); err != nil {
		return err
	}

	parsed, _ := url.Parse(rawURL)
	host := strings.ToLower(parsed.Hostname())

	// Wait briefly to detect DNS rebinding attacks
	time.Sleep(100 * time.Millisecond)

	// Clear cache for this host to force re-resolution
	v.dnsCache.delete(host)

	// Second validation with fresh DNS lookup
	return v.Validate(ctx, rawURL)
}

// validateScheme checks if the URL scheme is allowed
func (v *URLValidator) validateScheme(parsed *url.URL) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && (!v.allowHTTP || scheme != "http") {
		return fmt.Errorf("invalid scheme '%s': only https is allowed", scheme)
	}
	return nil
}

// validateHost checks if the hostname is dangerous
func (v *URLValidator) validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	// Check for dangerous hostnames
	dangerous := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"::1",
		"[::]",
		"metadata.google.internal",
		"metadata.google.com",
		"metadata.goog",
		"169.254.169.254",       // AWS/Azure/GCP metadata
		"metadata.internal",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
		"kubernetes",
	}

	for _, d := range dangerous {
		if host == d || strings.HasSuffix(host, "."+d) {
			return fmt.Errorf("blocked dangerous host: %s", host)
		}
	}

	// Check for IP addresses in hostname (bypass attempt)
	if net.ParseIP(host) != nil {
		// If it's an IP, validate it directly
		ip := net.ParseIP(host)
		return v.validateIP(ip)
	}

	return nil
}

// checkAllowlist verifies the host is in the allowlist
func (v *URLValidator) checkAllowlist(host string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// If allowlist is empty, allow all non-blocked hosts
	if len(v.allowedHosts) == 0 {
		return nil
	}

	// Check if host is in allowlist
	if !v.allowedHosts[host] {
		// Also check with www prefix/suffix variations
		if !v.allowedHosts["www."+host] && !v.allowedHosts[strings.TrimPrefix(host, "www.")] {
			return fmt.Errorf("host not in allowlist: %s", host)
		}
	}

	return nil
}

// resolveDNS resolves hostname to IP addresses with caching
func (v *URLValidator) resolveDNS(ctx context.Context, host string) ([]net.IP, error) {
	// Check cache first
	if cached := v.dnsCache.get(host); cached != nil {
		return cached, nil
	}

	// Create a custom resolver with timeout
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	// Set a timeout for DNS resolution
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Resolve the hostname
	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found for host: %s", host)
	}

	// Cache the result
	v.dnsCache.set(host, ips)

	return ips, nil
}

// validateIP checks if an IP address is blocked
func (v *URLValidator) validateIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("nil IP address")
	}

	// Check against blocked CIDR ranges
	for _, cidr := range v.blockedCIDRs {
		if cidr.Contains(ip) {
			return fmt.Errorf("blocked IP address %s (private/internal range)", ip.String())
		}
	}

	// Additional checks for special addresses
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("blocked special IP address: %s", ip.String())
	}

	return nil
}

// validatePort checks if the port is allowed
func (v *URLValidator) validatePort(parsed *url.URL) error {
	port := parsed.Port()
	if port == "" {
		// Default ports are OK
		return nil
	}

	// Block common internal service ports
	blockedPorts := map[string]bool{
		"22":    true, // SSH
		"23":    true, // Telnet
		"25":    true, // SMTP
		"110":   true, // POP3
		"135":   true, // RPC
		"139":   true, // NetBIOS
		"445":   true, // SMB
		"1433":  true, // MSSQL
		"3306":  true, // MySQL
		"3389":  true, // RDP
		"5432":  true, // PostgreSQL
		"5900":  true, // VNC
		"6379":  true, // Redis
		"8080":  true, // Common HTTP alt
		"8081":  true, // Common HTTP alt
		"9200":  true, // Elasticsearch
		"11211": true, // Memcached
		"27017": true, // MongoDB
	}

	if blockedPorts[port] {
		return fmt.Errorf("blocked port: %s", port)
	}

	return nil
}

// DNSCache methods

func (c *DNSCache) get(host string) []net.IP {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[host]
	if !exists {
		return nil
	}

	// Check if cache entry is still valid
	if time.Since(entry.timestamp) > c.ttl {
		return nil
	}

	// Return a copy to prevent modification
	ips := make([]net.IP, len(entry.ips))
	copy(ips, entry.ips)
	return ips
}

func (c *DNSCache) set(host string, ips []net.IP) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store a copy to prevent external modification
	ipsCopy := make([]net.IP, len(ips))
	copy(ipsCopy, ips)

	c.cache[host] = &dnsCacheEntry{
		ips:       ipsCopy,
		timestamp: time.Now(),
	}
}

func (c *DNSCache) delete(host string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, host)
}

// CleanupCache removes expired entries from the DNS cache
func (c *DNSCache) CleanupCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for host, entry := range c.cache {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(c.cache, host)
		}
	}
}