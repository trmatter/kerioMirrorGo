package mirror

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"time"

	"kerio-mirror-go/config"

	"github.com/sirupsen/logrus"
)

func MirrorUpdate(cfg *config.Config, logger *logrus.Logger) {
	start := time.Now()
	logger.Info("MirrorUpdate start")

	// open DB
	conn, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		logger.Errorf("DB open error: %v", err)
		return
	}
	defer conn.Close()

	// Загрузка баз IDS
	DownloadAndUpdateIDS(conn, cfg, logger)

	// Загрузка баз GeoIP
	if cfg.GeoIP4Url != "" && cfg.GeoIP6Url != "" {
		// Call the new function from geo.go to handle GeoIP update
		UpdateGeoIPDatabases(conn, cfg, logger)
	}

	// Download locations file if configured
	if cfg.GeoLocUrl != "" {
		DownloadGeoLocations(cfg, logger)
	}

	// Загрузка баз WebFilter
	UpdateWebFilterKey(conn, cfg, logger)
	// Загрузка баз Bitdefender
	downloadAndStoreBitdefender(conn, "bitdefender", cfg.BitdefenderUrls, "mirror/bitdefender", cfg, logger)

	duration := time.Since(start)
	logger.Infof("MirrorUpdate completed in %s", duration)
}

func calcChecksum(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func StartScheduler(cfg *config.Config, logger *logrus.Logger) {
	ticker := time.NewTicker(time.Duration(cfg.ScheduleInterval) * time.Hour)
	for range ticker.C {
		MirrorUpdate(cfg, logger)
	}
}

// contains is a helper for substring search
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (contains(s[1:], substr) || contains(s[:len(s)-1], substr)))) || (len(s) < len(substr) && false)
}
