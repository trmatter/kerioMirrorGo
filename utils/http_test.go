package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateHTTPClient_WithoutProxy(t *testing.T) {
	client, err := CreateHTTPClient("", 60*time.Second)
	if err != nil {
		t.Fatalf("CreateHTTPClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", client.Timeout)
	}
}

func TestCreateHTTPClient_WithProxy(t *testing.T) {
	proxyURL := "http://proxy.example.com:8080"
	client, err := CreateHTTPClient(proxyURL, 60*time.Second)
	if err != nil {
		t.Fatalf("CreateHTTPClient with proxy failed: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport")
	}

	if transport.Proxy == nil {
		t.Error("Expected proxy to be configured")
	}
}

func TestCreateHTTPClient_InvalidProxy(t *testing.T) {
	proxyURL := "://invalid-url"
	_, err := CreateHTTPClient(proxyURL, 60*time.Second)
	if err == nil {
		t.Error("Expected error for invalid proxy URL")
	}
}

func TestHTTPGetWithRetry_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Test successful request
	resp, err := HTTPGetWithRetry(server.URL, 3, 100*time.Millisecond, "")
	if err != nil {
		t.Fatalf("HTTPGetWithRetry failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHTTPGetWithRetry_FailureThenSuccess(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Should succeed on second attempt
	resp, err := HTTPGetWithRetry(server.URL, 3, 10*time.Millisecond, "")
	if err != nil {
		t.Fatalf("HTTPGetWithRetry failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestHTTPGetWithRetry_AllFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Should fail after all retries
	_, err := HTTPGetWithRetry(server.URL, 2, 10*time.Millisecond, "")
	if err == nil {
		t.Error("Expected error after all retries failed")
	}
}

func TestHTTPGetWithRetry_InvalidURL(t *testing.T) {
	_, err := HTTPGetWithRetry("http://invalid-domain-that-does-not-exist-12345.com", 1, 10*time.Millisecond, "")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func BenchmarkCreateHTTPClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CreateHTTPClient("", 60*time.Second)
	}
}

func BenchmarkHTTPGetWithRetry(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := HTTPGetWithRetry(server.URL, 1, 10*time.Millisecond, "")
		if resp != nil {
			resp.Body.Close()
		}
	}
}
