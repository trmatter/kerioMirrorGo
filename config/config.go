package config

import (
	"fmt" // Import fmt for error handling
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ScheduleInterval  int
	ScheduleTime      string // время запуска в формате HH:MM
	IDSUrl            string
	WebFilterApi      string
	BitdefenderUrls   []string
	DatabasePath      string
	LogPath           string
	RetryCount        int
	RetryDelaySeconds int
	ProxyURL          string // URL прокси-сервера, если требуется
	GeoIP4Url         string
	GeoIP6Url         string
	GeoLocUrl         string
	LicenseNumber     string
	LogLevel          string // уровень логирования: debug, info, warn, error
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
	viper.SetDefault("BITDEFENDER_URLS", []string{""})
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

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		// ignore if not found, use defaults
	}

	return &Config{
		ScheduleTime:      viper.GetString("SCHEDULE_TIME"),
		IDSUrl:            viper.GetString("IDS_URL"),
		WebFilterApi:      viper.GetString("WEBFILTER_API"),
		BitdefenderUrls:   viper.GetStringSlice("BITDEFENDER_URLS"),
		DatabasePath:      viper.GetString("DATABASE_PATH"),
		LogPath:           viper.GetString("LOG_PATH"),
		RetryCount:        viper.GetInt("RETRY_COUNT"),
		RetryDelaySeconds: viper.GetInt("RETRY_DELAY_SECONDS"),
		ProxyURL:          viper.GetString("PROXY_URL"),
		GeoIP4Url:         viper.GetString("GEOIP4_URL"),
		GeoIP6Url:         viper.GetString("GEOIP6_URL"),
		GeoLocUrl:         viper.GetString("GEOLOC_URL"),
		LicenseNumber:     viper.GetString("LICENSE_NUMBER"),
		LogLevel:          viper.GetString("LOG_LEVEL"),
	}, nil
}

func Save(cfg *Config, path string) error {
	// Set the values in viper from the config struct
	viper.Set("SCHEDULE_TIME", cfg.ScheduleTime)
	viper.Set("IDS_URL", cfg.IDSUrl)
	viper.Set("WEBFILTER_API", cfg.WebFilterApi)
	viper.Set("BITDEFENDER_URLS", cfg.BitdefenderUrls)
	viper.Set("DATABASE_PATH", cfg.DatabasePath)
	viper.Set("LOG_PATH", cfg.LogPath)
	viper.Set("RETRY_COUNT", cfg.RetryCount)
	viper.Set("RETRY_DELAY_SECONDS", cfg.RetryDelaySeconds)
	viper.Set("PROXY_URL", cfg.ProxyURL)
	viper.Set("GEOIP4_URL", cfg.GeoIP4Url)
	viper.Set("GEOIP6_URL", cfg.GeoIP6Url)
	viper.Set("GEOLOC_URL", cfg.GeoLocUrl)
	viper.Set("LICENSE_NUMBER", cfg.LicenseNumber)
	viper.Set("LOG_LEVEL", cfg.LogLevel)

	// Set config type explicitly if file extension is missing or not supported for writing
	ext := filepath.Ext(path)
	if ext == "" || (ext != ".json" && ext != ".yaml" && ext != ".yml" && ext != ".toml" && ext != ".ini") {
		return fmt.Errorf("unsupported config file type: %s. Please use .json, .yaml, .toml, or .ini", ext)
	}

	viper.SetConfigFile(path)

	// Write the config to file
	if err := viper.WriteConfig(); err != nil {
		// If the config file doesn't exist, try writing it as a new file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
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
