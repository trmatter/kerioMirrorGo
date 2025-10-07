package mirror

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// downloadSnortTemplate downloads Snort template files for IPS updates (Kerio 9.5)
// This is called internally as part of IDS5 update process
func downloadSnortTemplate(conn *sql.DB, cfg *config.Config, logger *logrus.Logger) bool {
	if !cfg.EnableSnortTemplate {
		logger.Info("IDSv5/Snort: template update is disabled by config")
		return true // Not an error, just disabled
	}

	if cfg.SnortTemplateURL == "" {
		logger.Warn("IDSv5/Snort: template URL is not configured")
		return false
	}

	logger.Info("IDSv5/Snort: downloading template files")

	// Создаём директорию для Snort файлов
	destDir := filepath.Join("mirror", "custom", "control-update", "config", "v1")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		logger.Errorf("IDSv5/Snort: failed to create directory %s: %v", destDir, err)
		db.UpdateSnortTemplateStatus(conn, false, time.Now())
		return false
	}

	// Скачиваем snort.tpl
	snortTplPath := filepath.Join(destDir, "snort.tpl")
	if !utils.DownloadFileWithProxy(cfg.SnortTemplateURL, snortTplPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
		logger.Error("IDSv5/Snort: failed to download snort.tpl")
		db.UpdateSnortTemplateStatus(conn, false, time.Now())
		return false
	}
	logger.Info("IDSv5/Snort: snort.tpl downloaded successfully")

	// Скачиваем snort.tpl.md5
	md5URL := cfg.SnortTemplateURL + ".md5"
	md5Path := snortTplPath + ".md5"
	if !utils.DownloadFileWithProxy(md5URL, md5Path, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
		logger.Warn("IDSv5/Snort: failed to download snort.tpl.md5 (non-critical)")
		// Не фейлим обновление, если MD5 не загрузился
	} else {
		logger.Info("IDSv5/Snort: snort.tpl.md5 downloaded successfully")
	}

	// Обновляем статус в БД
	if err := db.UpdateSnortTemplateStatus(conn, true, time.Now()); err != nil {
		logger.Errorf("IDSv5/Snort: failed to update status in DB: %v", err)
		return false
	}

	logger.Info("IDSv5/Snort: template update completed successfully")
	return true
}
