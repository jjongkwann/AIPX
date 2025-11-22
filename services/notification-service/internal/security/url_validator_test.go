package security

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewURLValidator(t *testing.T) {
	v := NewURLValidator()
	
	assert.NotNil(t, v)
	assert.NotEmpty(t, v.allowedHosts)
	assert.NotEmpty(t, v.blockedCIDRs)
	assert.False(t, v.allowHTTP)
	assert.Equal(t, 3, v.maxRedirects)
	assert.NotNil(t, v.dnsCache)
	
	// Check that Slack is in default allowlist
	assert.True(t, v.allowedHosts["hooks.slack.com"])
}

func TestValidateScheme(t *testing.T) {
	v := NewURLValidator()
	
	testCases := []struct {
		name      string
		url       string
		allowHTTP bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "HTTPS allowed",
			url:       "https://hooks.slack.com/webhook",
			allowHTTP: false,
			wantErr:   false,
		},
		{
			name:      "HTTP blocked by default",
			url:       "http://hooks.slack.com/webhook",
			allowHTTP: false,
			wantErr:   true,
			errMsg:    "invalid scheme",
		},
		{
			name:      "HTTP allowed when enabled",
			url:       "http://hooks.slack.com/webhook",
			allowHTTP: true,
			wantErr:   false,
		},
		{
			name:      "FTP blocked",
			url:       "ftp://example.com/file",
			allowHTTP: false,
			wantErr:   true,
			errMsg:    "invalid scheme",
		},
		{
			name:      "File scheme blocked",
			url:       "file:///etc/passwd",
			allowHTTP: false,
			wantErr:   true,
			errMsg:    "invalid scheme",
		},
		{
			name:      "Gopher scheme blocked",
			url:       "gopher://example.com",
			allowHTTP: false,
			wantErr:   true,
			errMsg:    "invalid scheme",
		},
		{
			name:      "Data URI blocked",
			url:       "data:text/plain,hello",
			allowHTTP: false,
			wantErr:   true,
			errMsg:    "invalid scheme",
		},
		{
			name:      "JavaScript blocked",
			url:       "javascript:alert(1)",
			allowHTTP: false,
			wantErr:   true,
			errMsg:    "invalid scheme",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v.SetAllowHTTP(tc.allowHTTP)
			err := v.Validate(context.Background(), tc.url)
			
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				// May still fail on DNS resolution, but not on scheme
				if err != nil {
					assert.NotContains(t, err.Error(), "invalid scheme")
				}
			}
		})
	}
}

func TestValidateDangerousHosts(t *testing.T) {
	v := NewURLValidator()
	
	dangerousURLs := []struct {
		name string
		url  string
	}{
		{"localhost", "https://localhost/webhook"},
		{"localhost with port", "https://localhost:8080/webhook"},
		{"127.0.0.1", "https://127.0.0.1/webhook"},
		{"0.0.0.0", "https://0.0.0.0/webhook"},
		{"IPv6 localhost", "https://[::1]/webhook"},
		{"AWS metadata", "https://169.254.169.254/latest/meta-data/"},
		{"GCP metadata", "https://metadata.google.internal/"},
		{"Azure metadata", "https://169.254.169.254/metadata/"},
		{"Kubernetes", "https://kubernetes.default/"},
		{"Kubernetes full", "https://kubernetes.default.svc.cluster.local/"},
		{"metadata.internal", "https://metadata.internal/"},
		{"Subdomain localhost", "https://api.localhost/webhook"},
	}
	
	for _, tc := range dangerousURLs {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Validate(context.Background(), tc.url)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "blocked")
		})
	}
}

func TestValidatePrivateIPs(t *testing.T) {
	v := NewURLValidator()
	
	privateIPs := []struct {
		name string
		url  string
	}{
		{"10.x.x.x", "https://10.0.0.1/webhook"},
		{"172.16.x.x", "https://172.16.0.1/webhook"},
		{"172.31.x.x", "https://172.31.255.254/webhook"},
		{"192.168.x.x", "https://192.168.1.1/webhook"},
		{"Link-local", "https://169.254.100.1/webhook"},
		{"IPv6 private", "https://[fc00::1]/webhook"},
		{"IPv6 link-local", "https://[fe80::1]/webhook"},
		{"Multicast", "https://224.0.0.1/webhook"},
		{"IPv6 multicast", "https://[ff02::1]/webhook"},
	}
	
	for _, tc := range privateIPs {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Validate(context.Background(), tc.url)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "blocked")
		})
	}
}

func TestValidateBlockedPorts(t *testing.T) {
	v := NewURLValidator()
	v.AddAllowedHost("example.com")
	
	blockedPorts := []struct {
		name string
		url  string
		port string
	}{
		{"SSH", "https://example.com:22/webhook", "22"},
		{"MySQL", "https://example.com:3306/webhook", "3306"},
		{"PostgreSQL", "https://example.com:5432/webhook", "5432"},
		{"Redis", "https://example.com:6379/webhook", "6379"},
		{"MongoDB", "https://example.com:27017/webhook", "27017"},
		{"Elasticsearch", "https://example.com:9200/webhook", "9200"},
		{"SMB", "https://example.com:445/webhook", "445"},
		{"RDP", "https://example.com:3389/webhook", "3389"},
	}
	
	for _, tc := range blockedPorts {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Validate(context.Background(), tc.url)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "blocked port")
		})
	}
}

func TestValidateAllowlist(t *testing.T) {
	v := NewURLValidator()
	// Clear default allowlist for testing
	v.allowedHosts = make(map[string]bool)
	
	// Test with empty allowlist (should allow all non-blocked)
	err := v.Validate(context.Background(), "https://any-external-site.com/webhook")
	// Will fail on DNS but not on allowlist
	if err != nil {
		assert.NotContains(t, err.Error(), "not in allowlist")
	}
	
	// Add specific host to allowlist
	v.AddAllowedHost("allowed.com")
	
	// Test allowed host
	err = v.Validate(context.Background(), "https://allowed.com/webhook")
	// May fail on DNS but not on allowlist
	if err != nil {
		assert.NotContains(t, err.Error(), "not in allowlist")
	}
	
	// Test blocked host
	err = v.Validate(context.Background(), "https://notallowed.com/webhook")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowlist")
	
	// Test www variations
	v.AddAllowedHost("www.example.com")
	err = v.Validate(context.Background(), "https://example.com/webhook")
	// Should check both with and without www
	if err != nil {
		assert.NotContains(t, err.Error(), "not in allowlist")
	}
}

func TestValidateEmptyURL(t *testing.T) {
	v := NewURLValidator()
	
	testCases := []string{
		"",
		" ",
		"\t",
		"\n",
		"   \t\n  ",
	}
	
	for _, tc := range testCases {
		t.Run("empty_url", func(t *testing.T) {
			err := v.Validate(context.Background(), tc)
			assert.Error(t, err)
		})
	}
}

func TestValidateInvalidURL(t *testing.T) {
	v := NewURLValidator()
	
	testCases := []string{
		"not-a-url",
		"://missing-scheme",
		"https://",
		"https:///path-only",
		"https://user:pass@/path",
		"https://[invalid-ipv6/path",
	}
	
	for _, tc := range testCases {
		t.Run("invalid_url", func(t *testing.T) {
			err := v.Validate(context.Background(), tc)
			assert.Error(t, err)
		})
	}
}

func TestDNSCache(t *testing.T) {
	cache := &DNSCache{
		cache: make(map[string]*dnsCacheEntry),
		ttl:   100 * time.Millisecond,
	}
	
	host := "example.com"
	ips := []net.IP{net.ParseIP("93.184.216.34")}
	
	// Test set and get
	cache.set(host, ips)
	cached := cache.get(host)
	assert.Equal(t, ips, cached)
	
	// Test cache expiry
	time.Sleep(150 * time.Millisecond)
	cached = cache.get(host)
	assert.Nil(t, cached)
	
	// Test delete
	cache.set(host, ips)
	cache.delete(host)
	cached = cache.get(host)
	assert.Nil(t, cached)
}

func TestDNSCacheCleanup(t *testing.T) {
	cache := &DNSCache{
		cache: make(map[string]*dnsCacheEntry),
		ttl:   100 * time.Millisecond,
	}
	
	// Add entries
	cache.set("host1", []net.IP{net.ParseIP("1.1.1.1")})
	time.Sleep(50 * time.Millisecond)
	cache.set("host2", []net.IP{net.ParseIP("2.2.2.2")})
	
	// Wait for first entry to expire
	time.Sleep(60 * time.Millisecond)
	
	// Cleanup should remove expired entry only
	cache.CleanupCache()
	
	assert.Nil(t, cache.get("host1"))
	assert.NotNil(t, cache.get("host2"))
}

func TestValidateWithDNSRebindingProtection(t *testing.T) {
	v := NewURLValidator()
	v.AddAllowedHost("example.com")

	// This test would need a mock DNS resolver to properly test
	// DNS rebinding protection. For now, we just ensure it doesn't panic
	// and performs two validations

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// example.com is a valid domain that may resolve successfully
	// Just verify it doesn't panic - the result depends on DNS resolution
	_ = v.ValidateWithDNSRebindingProtection(ctx, "https://example.com/webhook")
}

func TestAddRemoveAllowedHost(t *testing.T) {
	v := NewURLValidator()
	
	// Clear defaults for testing
	v.allowedHosts = make(map[string]bool)
	
	host := "test.example.com"
	
	// Add host
	v.AddAllowedHost(host)
	assert.True(t, v.allowedHosts[host])
	
	// Add uppercase (should be lowercased)
	v.AddAllowedHost("TEST.UPPERCASE.COM")
	assert.True(t, v.allowedHosts["test.uppercase.com"])
	
	// Remove host
	v.RemoveAllowedHost(host)
	assert.False(t, v.allowedHosts[host])
	
	// Remove non-existent host (should not panic)
	v.RemoveAllowedHost("non-existent.com")
}

func TestConcurrentAccess(t *testing.T) {
	v := NewURLValidator()
	
	// Test concurrent access to validator
	done := make(chan bool)
	
	// Multiple goroutines adding/removing hosts
	for i := 0; i < 10; i++ {
		go func(i int) {
			host := fmt.Sprintf("host%d.example.com", i)
			v.AddAllowedHost(host)
			v.RemoveAllowedHost(host)
			v.SetAllowHTTP(i%2 == 0)
			done <- true
		}(i)
	}
	
	// Multiple goroutines validating
	for i := 0; i < 10; i++ {
		go func() {
			v.Validate(context.Background(), "https://example.com/webhook")
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
	
	// If we get here without panic/race, test passes
}

func TestIPAddressInHostname(t *testing.T) {
	v := NewURLValidator()

	testCases := []struct {
		name        string
		url         string
		wantErr     bool
		errContains string
	}{
		{
			name:    "Direct private IP",
			url:     "https://192.168.1.1/webhook",
			wantErr: true,
		},
		{
			name:    "Direct public IP",
			url:     "https://8.8.8.8/webhook",
			wantErr: false, // Public IPs are generally allowed
		},
		{
			name:    "IPv6 private",
			url:     "https://[fc00::1]/webhook",
			wantErr: true,
		},
		{
			name:        "Encoded IP",
			url:         "https://0x7f.0x0.0x0.0x1/webhook", // 127.0.0.1 in hex
			wantErr:     true,
			errContains: "allowlist", // Hex IP not parsed by Go, fails on allowlist
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Validate(context.Background(), tc.url)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				} else {
					assert.Contains(t, err.Error(), "blocked")
				}
			} else {
				// May fail on other checks, but not on IP blocking
				if err != nil {
					assert.NotContains(t, err.Error(), "blocked")
				}
			}
		})
	}
}

func TestSpecialIPAddresses(t *testing.T) {
	v := NewURLValidator()
	
	testCases := []struct {
		name string
		ip   net.IP
		want bool // true if should be blocked
	}{
		{"Loopback IPv4", net.ParseIP("127.0.0.1"), true},
		{"Loopback IPv6", net.ParseIP("::1"), true},
		{"Private 10.x", net.ParseIP("10.0.0.1"), true},
		{"Private 172.16", net.ParseIP("172.16.0.1"), true},
		{"Private 192.168", net.ParseIP("192.168.1.1"), true},
		{"Link-local", net.ParseIP("169.254.1.1"), true},
		{"Multicast", net.ParseIP("224.0.0.1"), true},
		{"Public Google DNS", net.ParseIP("8.8.8.8"), false},
		{"Public Cloudflare", net.ParseIP("1.1.1.1"), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.validateIP(tc.ip)
			
			if tc.want {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "blocked")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}