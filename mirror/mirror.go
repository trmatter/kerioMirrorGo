package mirror

import (
	"database/sql"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"

	"github.com/sirupsen/logrus"
)

func Update(cfg *config.Config, logger *logrus.Logger) {
	start := time.Now()
	logger.Info("MirrorUpdate started")

	// open DB
	conn, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		logger.Errorf("DB open error: %v", err)
		return
	}
	defer conn.Close()

	// Загрузка баз IDS
	DownloadAndUpdateIDS(conn, cfg, logger)

	// Загрузка баз GeoIP
	if cfg.GeoIP4URL != "" && cfg.GeoIP6URL != "" {
		// Call the new function from geo.go to handle GeoIP update
		UpdateGeoIPDatabases(conn, cfg, logger)
	}

	// Download locations file if configured
	if cfg.GeoLocURL != "" {
		DownloadGeoLocations(cfg, logger)
	}

	// Загрузка баз WebFilter
	UpdateWebFilterKey(conn, cfg, logger)
	// Загрузка баз Bitdefender
	if cfg.EnableBitdefender {
		downloadAndStoreBitdefender(conn, cfg.BitdefenderURLs, "mirror/bitdefender", cfg, logger)
	} else {
		logger.Infof("Bitdefender update is disabled by config.")
	}

	// Загрузка Shield Matrix
	UpdateShieldMatrix(conn, cfg, logger)

	// --- Custom Download URLs ---
	DownloadCustomFiles(cfg, logger)
	// --- END Custom Download URLs ---

	duration := time.Since(start)
	logger.Infof("MirrorUpdate completed in %s", duration)

	// Сохраняем время последнего обновления
	err = saveLastUpdate(conn)
	if err != nil {
		logger.Errorf("Failed to save last update time: %v", err)
	}
}

func StartScheduler(cfg *config.Config, logger *logrus.Logger) {
	for {
		now := time.Now()
		target, err := time.ParseInLocation("15:04", cfg.ScheduleTime, now.Location())
		if err != nil {
			logger.Errorf("Invalid ScheduleTime format: %v", err)
			return
		}
		targetTime := time.Date(now.Year(), now.Month(), now.Day(), target.Hour(), target.Minute(), 0, 0, now.Location())
		if now.After(targetTime) {
			targetTime = targetTime.Add(24 * time.Hour)
		}
		dur := targetTime.Sub(now)
		logger.Infof("Next scheduled update at %s (in %s)", targetTime.Format("2006-01-02 15:04:05"), dur)
		time.Sleep(dur)
		Update(cfg, logger)
	}
}

// contains is a helper for substring search
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (contains(s[1:], substr) || contains(s[:len(s)-1], substr)))) || (len(s) < len(substr) && false)
}

// saveLastUpdate сохраняет текущее время в таблицу last_update
func saveLastUpdate(conn *sql.DB) error {
	return db.SetLastUpdate(conn, time.Now())
}
