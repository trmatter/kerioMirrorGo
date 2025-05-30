package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	TimeZone                string
	ScheduleInterval        int
	IDSUrls                 []string
	WebFilterApi            string
	BitdefenderUrls         []string
	DatabasePath            string
	LogPath                 string
	ListenAddress           string
	RetryCount              int
	RetryDelaySeconds       int
	ProxyURL                string // URL прокси-сервера, если требуется
	GeoIP4Url               string
	GeoIP6Url               string
	GeoLocUrl               string
	LicenseNumber           string
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetDefault("TIME_ZONE", "Europe/Amsterdam")
	viper.SetDefault("SCHEDULE_INTERVAL", 6)
	viper.SetDefault("IDS_URLS", []string{"https://updates.kerio.com/gateway/ids/signatures.zip"})
	viper.SetDefault("WEBFILTER_API", "https://updates.kerio.com/webfilter/key")
	viper.SetDefault("BITDEFENDER_URLS", []string{"https://updates.bitdefender.com/resources/product/antivirus/vdmp/latest.vdmp"})
	viper.SetDefault("DATABASE_PATH", "./mirror.db")
	viper.SetDefault("LOG_PATH", "./logs/mirror.log")
	viper.SetDefault("LISTEN_ADDRESS", ":80")
	viper.SetDefault("RETRY_COUNT", 3)
	viper.SetDefault("RETRY_DELAY_SECONDS", 10)
	viper.SetDefault("PROXY_URL", "")
	viper.SetDefault("GEOIP4_URL", "https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv4.csv")
	viper.SetDefault("GEOIP6_URL", "https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv6.csv")
	viper.SetDefault("GEOLOC_URL", "https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Locations-en.csv")
	viper.SetDefault("LICENSE_NUMBER", "")

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		// ignore if not found, use defaults
	}

	return &Config{
		TimeZone:                viper.GetString("TIME_ZONE"),
		ScheduleInterval:        viper.GetInt("SCHEDULE_INTERVAL"),
		IDSUrls:                 viper.GetStringSlice("IDS_URLS"),
		WebFilterApi:            viper.GetString("WEBFILTER_API"),
		BitdefenderUrls:         viper.GetStringSlice("BITDEFENDER_URLS"),
		DatabasePath:            viper.GetString("DATABASE_PATH"),
		LogPath:                 viper.GetString("LOG_PATH"),
		ListenAddress:           viper.GetString("LISTEN_ADDRESS"),
		RetryCount:              viper.GetInt("RETRY_COUNT"),
		RetryDelaySeconds:       viper.GetInt("RETRY_DELAY_SECONDS"),
		ProxyURL:                viper.GetString("PROXY_URL"),
		GeoIP4Url:               viper.GetString("GEOIP4_URL"),
		GeoIP6Url:               viper.GetString("GEOIP6_URL"),
		GeoLocUrl:               viper.GetString("GEOLOC_URL"),
		LicenseNumber:           viper.GetString("LICENSE_NUMBER"),
	}, nil
}
