package utils

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

// CreateHTTPClient создаёт HTTP клиент с поддержкой прокси и настраиваемым timeout
func CreateHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	transport := &http.Transport{}
	if proxyURL != "" {
		proxyParsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyParsed)
	}
	// Если timeout не указан, используем значение по умолчанию 5 минут
	if timeout == 0 {
		timeout = 300 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// HttpGetWithRetry performs GET with retries, поддерживает прокси
func HTTPGetWithRetry(urlStr string, retries int, delay time.Duration, proxyURL string) (*http.Response, error) {
	transport := &http.Transport{}
	if proxyURL != "" {
		proxyParsed, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyParsed)
		}
	}
	client := &http.Client{Timeout: 60 * time.Second, Transport: transport}
	var resp *http.Response
	var err error
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
