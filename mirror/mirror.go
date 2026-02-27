package mirror

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/telegram"

	"github.com/sirupsen/logrus"
)

func Update(cfg *config.Config, logger *logrus.Logger) {
	start := time.Now()
	logger.Info("MirrorUpdate started")

	notifier := telegram.New(cfg)
	if err := notifier.NotifyStart("&#128260; <b>Kerio Mirror</b>: scheduled update started"); err != nil {
		logger.Warnf("Telegram notify start: %v", err)
	}

	// open DB
	conn, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		logger.Errorf("DB open error: %v", err)
		if err2 := notifier.NotifyError(fmt.Sprintf("&#10060; <b>Kerio Mirror</b>: failed to open database: %v", err)); err2 != nil {
			logger.Warnf("Telegram notify error: %v", err2)
		}
		return
	}
	defer conn.Close()

	// Загрузка баз IDS (включая GeoIP как IDS4)
	DownloadAndUpdateIDS(conn, cfg, logger)

	// Download locations file if configured (part of GeoIP/IDS4)
	if cfg.EnableIDS4 && cfg.GeoLocURL != "" {
		DownloadGeoLocations(cfg, logger)
	}

	// Загрузка баз WebFilter
	UpdateWebFilterKey(conn, cfg, logger)
	// Загрузка баз Bitdefender
	if cfg.BitdefenderMode == "mirror" {
		downloadAndStoreBitdefender(conn, cfg.BitdefenderURLs, "mirror/bitdefender", cfg, logger)
	} else if cfg.BitdefenderMode == "proxy" {
		// В proxy mode выполняем только очистку старых версий
		currentVersion := db.GetBitdefenderVersion(conn)
		if currentVersion > 0 {
			cleanupOldBitdefenderVersions("mirror/bitdefender", currentVersion, cfg.BitdefenderKeepVersions, logger)
		} else {
			logger.Info("Bitdefender proxy mode: no current version in DB, skipping cleanup")
		}
	} else {
		logger.Infof("Bitdefender is disabled by config (current mode: %s).", cfg.BitdefenderMode)
	}

	// Загрузка Shield Matrix
	UpdateShieldMatrix(conn, cfg, logger)

	// --- Custom Download URLs ---
	DownloadCustomFiles(cfg, logger)
	// --- END Custom Download URLs ---

	duration := time.Since(start)
	logger.Infof("MirrorUpdate completed in %s", duration)

	// Send Telegram summary notification
	sendUpdateSummary(notifier, conn, cfg, duration, logger)

	// Сохраняем время последнего обновления
	err = saveLastUpdate(conn)
	if err != nil {
		logger.Errorf("Failed to save last update time: %v", err)
	}
}

// sendUpdateSummary checks component statuses in DB and sends a Telegram notification.
func sendUpdateSummary(notifier *telegram.Notifier, conn *sql.DB, cfg *config.Config, duration time.Duration, logger *logrus.Logger) {
	if !notifier.Enabled() {
		return
	}

	var failed []string
	var ok []string

	for _, v := range []string{"1", "2", "3", "4", "5"} {
		enabled := false
		switch v {
		case "1":
			enabled = cfg.EnableIDS1
		case "2":
			enabled = cfg.EnableIDS2
		case "3":
			enabled = cfg.EnableIDS3
		case "4":
			enabled = cfg.EnableIDS4
		case "5":
			enabled = cfg.EnableIDS5
		}
		if !enabled {
			continue
		}
		success, _, err := db.GetIDSUpdateStatus(conn, v)
		if err != nil {
			continue // no data yet
		}
		if success {
			ok = append(ok, "IDS "+v)
		} else {
			failed = append(failed, "IDS "+v)
		}
	}

	if cfg.BitdefenderMode == "mirror" {
		success, _, err := db.GetBitdefenderUpdateStatus(conn)
		if err == nil {
			if success {
				ok = append(ok, "Bitdefender")
			} else {
				failed = append(failed, "Bitdefender")
			}
		}
	}

	if cfg.EnableShieldMatrix {
		success, _, err := db.GetShieldMatrixUpdateStatus(conn)
		if err == nil {
			if success {
				ok = append(ok, "Shield Matrix")
			} else {
				failed = append(failed, "Shield Matrix")
			}
		}
	}

	durationStr := duration.Round(time.Second).String()

	if len(failed) > 0 {
		msg := fmt.Sprintf("&#10060; <b>Kerio Mirror</b>: update finished with errors\n\n<b>Failed:</b> %s\n<b>Duration:</b> %s",
			strings.Join(failed, ", "), durationStr)
		if len(ok) > 0 {
			msg += fmt.Sprintf("\n<b>OK:</b> %s", strings.Join(ok, ", "))
		}
		if err := notifier.NotifyError(msg); err != nil {
			logger.Warnf("Telegram notify error: %v", err)
		}
		return
	}

	if len(ok) > 0 {
		msg := fmt.Sprintf("&#9989; <b>Kerio Mirror</b>: update completed\n\n<b>OK:</b> %s\n<b>Duration:</b> %s",
			strings.Join(ok, ", "), durationStr)
		if err := notifier.NotifySuccess(msg); err != nil {
			logger.Warnf("Telegram notify success: %v", err)
		}
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
