package client

import (
	"context"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"

	utls "github.com/refraction-networking/utls"
)

// roundTripper replaces Go's default transport with one that uses utls.
// utls controls the TLS handshake ClientHello, mimicking a real browser
// and avoiding detection via JA3 fingerprinting.
type roundTripper struct {
	dialer *net.Dialer
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	port := req.URL.Port()
	if port == "" {
		port = "443"
	}

	tcpConn, err := rt.dialer.DialContext(req.Context(), "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}

	uconn := utls.UClient(tcpConn, &utls.Config{ServerName: host}, utls.HelloChrome_Auto)
	if err := uconn.HandshakeContext(req.Context()); err != nil {
		_ = uconn.Close()
		return nil, err
	}

	// Verifica o protocolo que foi negociado no ALPN
	switch uconn.ConnectionState().NegotiatedProtocol {
	case "h2":
		// Servidor negociou HTTP/2
		t := &http2.Transport{}
		conn, err := t.NewClientConn(uconn)
		if err != nil {
			_ = uconn.Close()
			return nil, err
		}
		return conn.RoundTrip(req)
	default:
		// HTTP/1.1 ou sem ALPN
		t := &http.Transport{
			DialTLSContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return uconn, nil
			},
		}
		return t.RoundTrip(req)
	}
}

// BrowserClient is a shared http.Client configured to mimic a browser.
// It should be used to leverage the connection pool across multiple requests.
var BrowserClient = NewBrowserClient()

// NewBrowserClient returns a new http.Client configured to mimic a browser.
func NewBrowserClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &roundTripper{
			dialer: &net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			},
		},
	}
}

// SetBrowserHeaders adds the necessary headers to simulate a real browser.
func SetBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
}
