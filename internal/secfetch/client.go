package secfetch

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

// DialContextFunc matches net.Dialer.DialContext / http.Transport.DialContext.
type DialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// Option configures a Client beyond its Config.
type Option func(*Client)

// WithDialContext overrides the dial function used for outbound connections,
// bypassing the default SSRF guard (dialGuarded). Production code must never
// use this; it exists so tests can point the client at an httptest.Server
// without the guard rejecting the loopback address it listens on. The
// host allowlist/denylist and scheme checks in validateURL still apply.
func WithDialContext(fn DialContextFunc) Option {
	return func(c *Client) { c.dial = fn }
}

// Client performs GET requests restricted to an explicit host allowlist and
// hardened against SSRF (private/loopback/link-local/cloud-metadata
// targets), DNS rebinding, unbounded redirects, and unbounded response size.
type Client struct {
	cfg  Config
	dial DialContextFunc
	http *http.Client
}

// NewClient builds a Client from cfg. By default every connection is routed
// through dialGuarded, which re-resolves the target host and rejects it if
// any resolved address falls in a blocked range.
func NewClient(cfg Config, opts ...Option) *Client {
	c := &Client{cfg: cfg}
	dialer := &net.Dialer{Timeout: cfg.Timeout}
	c.dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialGuarded(ctx, dialer, network, addr)
	}
	for _, opt := range opts {
		opt(c)
	}

	transport := &http.Transport{
		DialContext:     func(ctx context.Context, network, addr string) (net.Conn, error) { return c.dial(ctx, network, addr) },
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	c.http = &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("secfetch: exceeded %d redirects", cfg.MaxRedirects)
			}
			// dialGuarded re-checks the resolved IP for this hop too, but
			// the host allowlist/scheme must be re-validated here since a
			// redirect can point anywhere, not just at addresses.
			if err := c.validateURL(req.URL); err != nil {
				return err
			}
			// Never forward auth/session headers to a redirect target.
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
			return nil
		},
	}
	return c
}

// validateURL enforces the https-only + host allowlist/denylist policy.
// IP-level checks happen later, in dialGuarded, at actual connect time --
// hostname checks alone can't catch DNS rebinding.
func (c *Client) validateURL(u *url.URL) error {
	if u.Scheme != "https" {
		return fmt.Errorf("secfetch: scheme %q not permitted, only https", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("secfetch: empty host")
	}
	if !hostAllowed(host, c.cfg.AllowedHosts, c.cfg.DeniedHosts) {
		return fmt.Errorf("secfetch: host %q is not on the allowlist", host)
	}
	return nil
}

// dialGuarded resolves addr's host and refuses to connect if every resolved
// IP falls in a blocked range, trying each non-blocked IP in turn. Checking
// the resolved IP -- not the hostname -- is what stops DNS rebinding: an
// allowlisted hostname can still resolve to 127.0.0.1 or a cloud metadata
// address.
func dialGuarded(ctx context.Context, dialer *net.Dialer, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("secfetch: resolve %q: %w", host, err)
	}
	var lastErr error
	for _, ip := range ips {
		if isBlockedIP(ip) {
			lastErr = fmt.Errorf("secfetch: %q resolves to blocked address %s", host, ip)
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err != nil {
			lastErr = err
			continue
		}
		return conn, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("secfetch: no addresses found for %q", host)
	}
	return nil, lastErr
}

// Result is a successful fetch outcome, with Body capped at cfg.MaxBodyBytes.
type Result struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Truncated  bool
}

// Fetch performs a GET request against rawURL, enforcing the client's
// SSRF/allowlist policy and size limits.
func (c *Client) Fetch(ctx context.Context, rawURL string) (*Result, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("secfetch: invalid URL: %w", err)
	}
	if err := c.validateURL(u); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.cfg.MaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("secfetch: read body: %w", err)
	}
	truncated := int64(len(body)) > c.cfg.MaxBodyBytes
	if truncated {
		body = body[:c.cfg.MaxBodyBytes]
	}

	// This is a read-only fetch tool, not a session proxy: never hand the
	// caller a Set-Cookie header from a site it doesn't control.
	resp.Header.Del("Set-Cookie")

	return &Result{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
		Truncated:  truncated,
	}, nil
}
