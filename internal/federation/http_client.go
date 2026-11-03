package federation

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func localPeerAllowed(host string) bool {
	if strings.EqualFold(os.Getenv("ENV"), "production") {
		return false
	}
	for _, raw := range strings.Split(os.Getenv("FEDERATION_LOCAL_PEERS"), ",") {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err == nil && u.Host == host && host != "" {
			return true
		}
	}
	return false
}
func publicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, cidr := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "2001:db8::/32"} {
		_, block, _ := net.ParseCIDR(cidr)
		if block.Contains(ip) {
			return false
		}
	}
	return true
}
func validateRemoteURL(u *url.URL) error {
	if u.User != nil || u.Host == "" {
		return fmt.Errorf("invalid remote origin")
	}
	if localPeerAllowed(u.Host) && (u.Scheme == "http" || u.Scheme == "https") {
		return nil
	}
	if u.Scheme != "https" {
		return fmt.Errorf("remote federation requires HTTPS")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicIP(ip) {
		return fmt.Errorf("private remote addresses are not allowed")
	}
	return nil
}

type guardedTransport struct{ base *http.Transport }

func (g guardedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if err := validateRemoteURL(r.URL); err != nil {
		return nil, err
	}
	resp, err := g.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if resp.ContentLength > 1024*1024 {
		resp.Body.Close()
		return nil, fmt.Errorf("remote response exceeds limit")
	}
	resp.Body = &limitedBody{ReadCloser: resp.Body, remaining: 1024 * 1024}
	return resp, nil
}

type limitedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		var one [1]byte
		n, err := b.ReadCloser.Read(one[:])
		if n > 0 {
			return 0, fmt.Errorf("remote response exceeds limit")
		}
		return 0, err
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	return n, err
}

// NewHTTPClient validates DNS at connect time as well as every redirect. It
// deliberately ignores environment HTTP proxies, which would bypass IP checks.
var sharedRemoteTransport = buildRemoteTransport()

func buildRemoteTransport() *http.Transport {
	transport := &http.Transport{MaxIdleConns: 4, MaxIdleConnsPerHost: 2, MaxConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if localPeerAllowed(address) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, address)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no remote addresses")
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, fmt.Errorf("private remote address rejected")
			}
		}
		// Dial a checked address directly, retaining the request hostname for TLS.
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return transport
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 || timeout > 15*time.Second {
		timeout = 10 * time.Second
	}
	return &http.Client{Timeout: timeout, Transport: guardedTransport{sharedRemoteTransport}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			return fmt.Errorf("signed request redirects are not allowed")
		}
		return validateRemoteURL(req.URL)
	}}
}
