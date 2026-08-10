package secfetch

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},             // loopback
		{"::1", true},                   // loopback v6
		{"10.0.0.5", true},              // RFC1918
		{"172.16.0.5", true},            // RFC1918
		{"192.168.1.5", true},           // RFC1918
		{"169.254.169.254", true},       // cloud metadata (link-local)
		{"169.254.1.1", true},           // link-local
		{"fe80::1", true},               // link-local v6
		{"fc00::1", true},               // unique local v6
		{"224.0.0.1", true},             // multicast
		{"100.64.0.1", true},            // CGNAT
		{"192.0.2.1", true},             // TEST-NET-1
		{"0.0.0.0", true},               // unspecified
		{"8.8.8.8", false},              // public
		{"1.1.1.1", false},              // public
		{"2606:4700:4700::1111", false}, // public v6
	}
	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("test bug: %q did not parse as an IP", tc.ip)
		}
		if got := isBlockedIP(ip); got != tc.blocked {
			t.Errorf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.blocked)
		}
	}
}

func TestMatchesHost(t *testing.T) {
	cases := []struct {
		host, pattern string
		want          bool
	}{
		{"example.com", "example.com", true},
		{"Example.com", "example.com", false}, // matchesHost does not lowercase host; caller must
		{"api.example.com", "example.com", false},
		{"api.example.com", "*.example.com", true},
		{"example.com", "*.example.com", true},
		{"evilexample.com", "*.example.com", false},
		{"example.com", "other.com", false},
		{"example.com", "", false},
	}
	for _, tc := range cases {
		if got := matchesHost(tc.host, tc.pattern); got != tc.want {
			t.Errorf("matchesHost(%q, %q) = %v, want %v", tc.host, tc.pattern, got, tc.want)
		}
	}
}

func TestHostAllowed(t *testing.T) {
	allowed := []string{"*.example.com", "api.internal-partner.com"}
	denied := []string{"blocked.example.com"}

	cases := []struct {
		host string
		want bool
	}{
		{"docs.example.com", true},
		{"api.internal-partner.com", true},
		{"blocked.example.com", false}, // denylist overrides allowlist
		{"not-allowed.com", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := hostAllowed(tc.host, allowed, denied); got != tc.want {
			t.Errorf("hostAllowed(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestHostAllowedFailsClosedWithEmptyAllowlist(t *testing.T) {
	if hostAllowed("example.com", nil, nil) {
		t.Fatal("hostAllowed with an empty allowlist must reject everything, not allow it")
	}
}
