package config

import (
	"errors"
	"fmt" // Import fmt for error handling
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ScheduleTime            string // время запуска в формате HH:MM
	IDSURL                  string
	WebFilterAPI            string
	BitdefenderURLs         []string
	EnableBitdefender       bool // Включить обновление Bitdefender
	DatabasePath            string
	LogPath                 string
	RetryCount              int
	RetryDelaySeconds       int
	ProxyURL                string // URL прокси-сервера, если требуется
	GeoIP4URL               string
	GeoIP6URL               string
	GeoLocURL               string
	LicenseNumber           string
	LogLevel                string   // уровень логирования: debug, info, warn, error
	CustomDownloadURLs      []string // Пользовательские URL для скачивания
	EnableIDS1              bool     // Включить обновление IDS1
	EnableIDS2              bool     // Включить обновление IDS2
	EnableIDS3              bool     // Включить обновление IDS3
	EnableIDS4              bool     // Включить обновление IDS4
	EnableIDS5              bool     // Включить обновление IDS5
	BitdefenderProxyMode    bool     // Режим прокси для Bitdefender (запросы передаются на сервер и кэшируются)
	BitdefenderProxyBaseURL string   // Базовый URL для прокси Bitdefender
	EnableSnortTemplate  bool   // Включить обновление шаблона Snort для IPS
	SnortTemplateURL     string // URL для скачивания snort.tpl
	EnableShieldMatrix   bool   // Включить обновление Shield Matrix (Kerio 9.5+)
	ShieldMatrixBaseURL  string // Базовый URL для Shield Matrix (без /version)
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)

	// Set config type explicitly if file extension is missing or not supported for writing
	ext := filepath.Ext(path)
	if ext == "" || (ext != ".json" && ext != ".yaml" && ext != ".yml" && ext != ".toml" && ext != ".ini") {
		return nil, fmt.Errorf("unsupported config file type: %s. Please use .json, .yaml, .toml, or .ini", ext)
	}

	viper.SetDefault("SCHEDULE_TIME", "03:00")
	viper.SetDefault("IDS_URL", "https://ids-update.kerio.com/update.php?id=%s&version=%s.0&tag=")
	viper.SetDefault("WEBFILTER_API", "https://updates.kerio.com/webfilter/key")
	viper.SetDefault("DATABASE_PATH", "./mirror.db")
	viper.SetDefault("LOG_PATH", "./logs/mirror.log")
	viper.SetDefault("RETRY_COUNT", 3)
	viper.SetDefault("RETRY_DELAY_SECONDS", 10)
	viper.SetDefault("PROXY_URL", "")
	viper.SetDefault("GEOIP4_URL", "https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv4.csv")
	viper.SetDefault("GEOIP6_URL", "https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv6.csv")
	viper.SetDefault("GEOLOC_URL", "https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Locations-en.csv")
	viper.SetDefault("LICENSE_NUMBER", "")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("CUSTOM_DOWNLOAD_URLS", []string{})
	viper.SetDefault("ENABLE_BITDEFENDER", true)
	viper.SetDefault("ENABLE_IDS1", true)
	viper.SetDefault("ENABLE_IDS2", true)
	viper.SetDefault("ENABLE_IDS3", true)
	viper.SetDefault("ENABLE_IDS4", true)
	viper.SetDefault("ENABLE_IDS5", true)
	viper.SetDefault("BITDEFENDER_PROXY_MODE", false)
	viper.SetDefault("BITDEFENDER_PROXY_BASE_URL", "https://upgrade.bitdefender.com")
	viper.SetDefault("ENABLE_SNORT_TEMPLATE", true)
	viper.SetDefault("SNORT_TEMPLATE_URL", "http://download.kerio.com/control-update/config/v1/snort.tpl")
	viper.SetDefault("ENABLE_SHIELD_MATRIX", true)
	viper.SetDefault("SHIELD_MATRIX_BASE_URL", "https://d2akeya8d016xi.cloudfront.net/9.5.0")

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		// ignore if not found, use defaults
	}

	return &Config{
		ScheduleTime:            viper.GetString("SCHEDULE_TIME"),
		IDSURL:                  viper.GetString("IDS_URL"),
		WebFilterAPI:            viper.GetString("WEBFILTER_API"),
		EnableBitdefender:       viper.GetBool("ENABLE_BITDEFENDER"),
		DatabasePath:            viper.GetString("DATABASE_PATH"),
		LogPath:                 viper.GetString("LOG_PATH"),
		RetryCount:              viper.GetInt("RETRY_COUNT"),
		RetryDelaySeconds:       viper.GetInt("RETRY_DELAY_SECONDS"),
		ProxyURL:                viper.GetString("PROXY_URL"),
		GeoIP4URL:               viper.GetString("GEOIP4_URL"),
		GeoIP6URL:               viper.GetString("GEOIP6_URL"),
		GeoLocURL:               viper.GetString("GEOLOC_URL"),
		LicenseNumber:           viper.GetString("LICENSE_NUMBER"),
		LogLevel:                viper.GetString("LOG_LEVEL"),
		CustomDownloadURLs:      viper.GetStringSlice("CUSTOM_DOWNLOAD_URLS"),
		EnableIDS1:              viper.GetBool("ENABLE_IDS1"),
		EnableIDS2:              viper.GetBool("ENABLE_IDS2"),
		EnableIDS3:              viper.GetBool("ENABLE_IDS3"),
		EnableIDS4:              viper.GetBool("ENABLE_IDS4"),
		EnableIDS5:              viper.GetBool("ENABLE_IDS5"),
		BitdefenderProxyMode:    viper.GetBool("BITDEFENDER_PROXY_MODE"),
		BitdefenderProxyBaseURL: viper.GetString("BITDEFENDER_PROXY_BASE_URL"),
		EnableSnortTemplate:     viper.GetBool("ENABLE_SNORT_TEMPLATE"),
		SnortTemplateURL:        viper.GetString("SNORT_TEMPLATE_URL"),
		EnableShieldMatrix:      viper.GetBool("ENABLE_SHIELD_MATRIX"),
		ShieldMatrixBaseURL:     viper.GetString("SHIELD_MATRIX_BASE_URL"),
	}, nil
}

func Save(cfg *Config, path string) error {
	// Set the values in viper from the config struct
	viper.Set("SCHEDULE_TIME", cfg.ScheduleTime)
	viper.Set("IDS_URL", cfg.IDSURL)
	viper.Set("WEBFILTER_API", cfg.WebFilterAPI)
	viper.Set("DATABASE_PATH", cfg.DatabasePath)
	viper.Set("LOG_PATH", cfg.LogPath)
	viper.Set("RETRY_COUNT", cfg.RetryCount)
	viper.Set("RETRY_DELAY_SECONDS", cfg.RetryDelaySeconds)
	viper.Set("PROXY_URL", cfg.ProxyURL)
	viper.Set("GEOIP4_URL", cfg.GeoIP4URL)
	viper.Set("GEOIP6_URL", cfg.GeoIP6URL)
	viper.Set("GEOLOC_URL", cfg.GeoLocURL)
	viper.Set("LICENSE_NUMBER", cfg.LicenseNumber)
	viper.Set("LOG_LEVEL", cfg.LogLevel)
	viper.Set("CUSTOM_DOWNLOAD_URLS", cfg.CustomDownloadURLs)
	viper.Set("ENABLE_BITDEFENDER", cfg.EnableBitdefender)
	viper.Set("ENABLE_IDS1", cfg.EnableIDS1)
	viper.Set("ENABLE_IDS2", cfg.EnableIDS2)
	viper.Set("ENABLE_IDS3", cfg.EnableIDS3)
	viper.Set("ENABLE_IDS4", cfg.EnableIDS4)
	viper.Set("ENABLE_IDS5", cfg.EnableIDS5)
	viper.Set("BITDEFENDER_PROXY_MODE", cfg.BitdefenderProxyMode)
	viper.Set("BITDEFENDER_PROXY_BASE_URL", cfg.BitdefenderProxyBaseURL)
	viper.Set("ENABLE_SNORT_TEMPLATE", cfg.EnableSnortTemplate)
	viper.Set("SNORT_TEMPLATE_URL", cfg.SnortTemplateURL)
	viper.Set("ENABLE_SHIELD_MATRIX", cfg.EnableShieldMatrix)
	viper.Set("SHIELD_MATRIX_BASE_URL", cfg.ShieldMatrixBaseURL)

	// Set config type explicitly if file extension is missing or not supported for writing
	ext := filepath.Ext(path)
	if ext == "" || (ext != ".json" && ext != ".yaml" && ext != ".yml" && ext != ".toml" && ext != ".ini") {
		return fmt.Errorf("unsupported config file type: %s. Please use .json, .yaml, .toml, or .ini", ext)
	}

	viper.SetConfigFile(path)

	// Write the config to file
	if err := viper.WriteConfig(); err != nil {
		// If the config file doesn't exist, try writing it as a new file
		var notFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &notFoundErr) {
			err = viper.WriteConfigAs(path)
			if err != nil {
				return fmt.Errorf("failed to write config file as %s: %w", path, err)
			}
		} else {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	return nil
}
