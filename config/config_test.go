package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test config
	configContent := `
schedule_time: "02:00"
database_path: "./test.db"
log_path: "./test.log"
proxy_url: "http://proxy.test:8080"
license_number: "TEST-1234"
enable_bitdefender: true
bitdefender_proxy_mode: true
bitdefender_proxy_base_url: "https://test.bitdefender.com"
`
	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load config
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.ScheduleTime != "02:00" {
		t.Errorf("Expected ScheduleTime '02:00', got '%s'", cfg.ScheduleTime)
	}
	if cfg.DatabasePath != "./test.db" {
		t.Errorf("Expected DatabasePath './test.db', got '%s'", cfg.DatabasePath)
	}
	if cfg.ProxyURL != "http://proxy.test:8080" {
		t.Errorf("Expected ProxyURL 'http://proxy.test:8080', got '%s'", cfg.ProxyURL)
	}
	if !cfg.EnableBitdefender {
		t.Error("Expected EnableBitdefender to be true")
	}
	if !cfg.BitdefenderProxyMode {
		t.Error("Expected BitdefenderProxyMode to be true")
	}
	if cfg.BitdefenderProxyBaseURL != "https://test.bitdefender.com" {
		t.Errorf("Expected BitdefenderProxyBaseURL 'https://test.bitdefender.com', got '%s'", cfg.BitdefenderProxyBaseURL)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Create empty config file
	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Load config with defaults
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults
	if cfg.ScheduleTime != "03:00" {
		t.Errorf("Expected default ScheduleTime '03:00', got '%s'", cfg.ScheduleTime)
	}
	if cfg.RetryCount != 3 {
		t.Errorf("Expected default RetryCount 3, got %d", cfg.RetryCount)
	}
	if cfg.BitdefenderProxyBaseURL != "https://upgrade.bitdefender.com" {
		t.Errorf("Expected default BitdefenderProxyBaseURL 'https://upgrade.bitdefender.com', got '%s'", cfg.BitdefenderProxyBaseURL)
	}
	if cfg.BitdefenderProxyMode != false {
		t.Error("Expected default BitdefenderProxyMode to be false")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create config
	cfg := &Config{
		ScheduleTime:            "04:00",
		DatabasePath:            "./save_test.db",
		LogPath:                 "./save_test.log",
		ProxyURL:                "http://save.test:3128",
		LicenseNumber:           "SAVE-TEST-5678",
		EnableBitdefender:       true,
		BitdefenderProxyMode:    true,
		BitdefenderProxyBaseURL: "https://save.bitdefender.com",
		RetryCount:              5,
		EnableIDS1:              true,
		EnableIDS2:              false,
	}

	// Save config
	if err := Save(cfg, tmpFile.Name()); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config back
	loadedCfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify values match
	if loadedCfg.ScheduleTime != cfg.ScheduleTime {
		t.Errorf("ScheduleTime mismatch: expected '%s', got '%s'", cfg.ScheduleTime, loadedCfg.ScheduleTime)
	}
	if loadedCfg.ProxyURL != cfg.ProxyURL {
		t.Errorf("ProxyURL mismatch: expected '%s', got '%s'", cfg.ProxyURL, loadedCfg.ProxyURL)
	}
	if loadedCfg.BitdefenderProxyMode != cfg.BitdefenderProxyMode {
		t.Errorf("BitdefenderProxyMode mismatch: expected %v, got %v", cfg.BitdefenderProxyMode, loadedCfg.BitdefenderProxyMode)
	}
	if loadedCfg.EnableIDS1 != cfg.EnableIDS1 {
		t.Errorf("EnableIDS1 mismatch: expected %v, got %v", cfg.EnableIDS1, loadedCfg.EnableIDS1)
	}
	if loadedCfg.EnableIDS2 != cfg.EnableIDS2 {
		t.Errorf("EnableIDS2 mismatch: expected %v, got %v", cfg.EnableIDS2, loadedCfg.EnableIDS2)
	}
}

func TestLoadUnsupportedFormat(t *testing.T) {
	// Try to load unsupported format
	_, err := Load("config.txt")
	if err == nil {
		t.Error("Expected error for unsupported config format, got nil")
	}
}