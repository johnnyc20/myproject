package secfetch

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func testConfig(allowed []string, maxBody int64) Config {
	return Config{
		AllowedHosts: allowed,
		Timeout:      2 * time.Second,
		MaxRedirects: 3,
		MaxBodyBytes: maxBody,
		UserAgent:    "secfetch-test/1.0",
	}
}

// insecureSkipVerify lets the test client trust httptest's self-signed cert.
func insecureSkipVerify(c *Client) {
	c.http.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
}

func TestFetch_BlocksLoopbackByDefault(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	// The host is on the allowlist, but it's a loopback address -- the
	// dial-time guard must reject it regardless of the allowlist.
	c := NewClient(testConfig([]string{u.Hostname()}, 1<<20))
	if _, err := c.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected loopback target to be blocked, got nil error")
	}
}

func TestFetch_RejectsHostNotOnAllowlist(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := NewClient(testConfig(nil, 1<<20))
	if _, err := c.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected fetch with empty allowlist to be rejected, got nil error")
	}
}

func TestFetch_RejectsNonHTTPSScheme(t *testing.T) {
	c := NewClient(testConfig([]string{"example.com"}, 1<<20))
	if err := c.validateURL(&url.URL{Scheme: "http", Host: "example.com"}); err == nil {
		t.Fatal("expected http scheme to be rejected, got nil error")
	}
}

func TestFetch_SuccessAndCookieStripping(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=leak-me")
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	dialer := &net.Dialer{}
	c := NewClient(testConfig([]string{u.Hostname()}, 1<<20), WithDialContext(dialer.DialContext))
	insecureSkipVerify(c)

	res, err := c.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", res.StatusCode)
	}
	if string(res.Body) != "hello world" {
		t.Errorf("Body = %q, want %q", res.Body, "hello world")
	}
	if res.Truncated {
		t.Error("Truncated = true, want false")
	}
	if res.Header.Get("Set-Cookie") != "" {
		t.Error("Set-Cookie header leaked to caller, want stripped")
	}
}

func TestFetch_TruncatesOversizedBody(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	dialer := &net.Dialer{}
	c := NewClient(testConfig([]string{u.Hostname()}, 5), WithDialContext(dialer.DialContext))
	insecureSkipVerify(c)

	res, err := c.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !res.Truncated {
		t.Error("Truncated = false, want true")
	}
	if string(res.Body) != "hello" {
		t.Errorf("Body = %q, want %q", res.Body, "hello")
	}
}
