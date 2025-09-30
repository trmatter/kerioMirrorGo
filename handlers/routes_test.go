package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kerio-mirror-go/config"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func TestUpdateKerioHandler_Version0(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/update.php?version=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := &config.Config{
		DatabasePath: ":memory:",
	}
	logger := logrus.New()

	handler := updateKerioHandler(cfg, logger)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "0:0.0" {
		t.Errorf("Expected body '0:0.0', got '%s'", rec.Body.String())
	}
}

func TestUpdateKerioHandler_Version9_ProxyDisabled(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/update.php?version=9", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := &config.Config{
		DatabasePath:         ":memory:",
		BitdefenderProxyMode: false,
	}
	logger := logrus.New()

	handler := updateKerioHandler(cfg, logger)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	expected := "THDdir=https://bdupdate.kerio.com/../"
	if rec.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rec.Body.String())
	}
}

func TestUpdateKerioHandler_Version9_ProxyEnabled(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/update.php?version=9", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := &config.Config{
		DatabasePath:         ":memory:",
		BitdefenderProxyMode: true,
	}
	logger := logrus.New()

	handler := updateKerioHandler(cfg, logger)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	expected := "THDdir=http://localhost:8080/"
	if rec.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rec.Body.String())
	}
}

func TestUpdateKerioHandler_Version10_ProxyEnabled(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/update.php?version=10", nil)
	req.Host = "192.168.1.1"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := &config.Config{
		DatabasePath:         ":memory:",
		BitdefenderProxyMode: true,
	}
	logger := logrus.New()

	handler := updateKerioHandler(cfg, logger)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	expected := "THDdir=http://192.168.1.1/"
	if rec.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, rec.Body.String())
	}
}

func TestUpdateKerioHandler_EmptyVersion(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/update.php", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := &config.Config{
		DatabasePath: ":memory:",
	}
	logger := logrus.New()

	handler := updateKerioHandler(cfg, logger)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestUpdateKerioHandler_InvalidVersion(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/update.php?version=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := &config.Config{
		DatabasePath: ":memory:",
	}
	logger := logrus.New()

	handler := updateKerioHandler(cfg, logger)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestWebFilterKeyHandler_NoLicense(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/getkey.php", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("logger", logrus.New())

	cfg := &config.Config{
		DatabasePath:  ":memory:",
		LicenseNumber: "",
	}

	handler := webFilterKeyHandler(cfg)

	// Execute
	err := handler(c)

	// Assert
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestParseMajorVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected int
		hasError bool
	}{
		{"1", 1, false},
		{"9", 9, false},
		{"10", 10, false},
		{"1.2.3", 1, false},
		{"9.4.1", 9, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			parts := strings.Split(tt.version, ".")
			if len(parts) == 0 {
				if !tt.hasError {
					t.Error("Expected error for empty version")
				}
				return
			}

			var majorVersion int
			_, err := fmt.Sscanf(parts[0], "%d", &majorVersion)

			if tt.hasError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.hasError && majorVersion != tt.expected {
				t.Errorf("Expected major version %d, got %d", tt.expected, majorVersion)
			}
		})
	}
}