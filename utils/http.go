package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// createTransport builds an http.Transport with HTTP/HTTPS or SOCKS5 proxy support.
func createTransport(proxyURL string) (*http.Transport, error) {
	transport := &http.Transport{}
	if proxyURL == "" {
		return transport, nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(parsed.Scheme) {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if parsed.User != nil {
			auth = &proxy.Auth{User: parsed.User.Username()}
			if pass, ok := parsed.User.Password(); ok {
				auth.Password = pass
			}
		}
		dialer, err := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}
		if cd, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = cd.DialContext
		} else {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
	default:
		transport.Proxy = http.ProxyURL(parsed)
	}

	return transport, nil
}

// CreateHTTPClient creates an HTTP client with proxy support (HTTP/HTTPS or SOCKS5) and configurable timeout.
func CreateHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	transport, err := createTransport(proxyURL)
	if err != nil {
		return nil, err
	}
	if timeout == 0 {
		timeout = 300 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// HTTPGetWithRetry performs GET with retries, supports HTTP/HTTPS and SOCKS5 proxy.
func HTTPGetWithRetry(urlStr string, retries int, delay time.Duration, proxyURL string) (*http.Response, error) {
	transport, err := createTransport(proxyURL)
	if err != nil {
		transport = &http.Transport{}
	}
	client := &http.Client{Timeout: 60 * time.Second, Transport: transport}
	var resp *http.Response
	for i := 0; i <= retries; i++ {
		resp, err = client.Get(urlStr)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(delay)
	}
	return nil, errors.New("failed to GET " + urlStr)
}
