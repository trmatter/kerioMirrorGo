package mirror

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// DownloadAndUpdateIDS implements the python logic for IDS update discovery and download
func DownloadAndUpdateIDS(conn *sql.DB, cfg *config.Config, logger *logrus.Logger) {
	idsVersions := []string{"1", "2", "3", "4", "5"}
	for _, version := range idsVersions {
		if cfg.LicenseNumber == "" {
			logger.Infof("IDSv%s: passing because license key is not configured", version)
			continue
		}
		url := fmt.Sprintf("https://ids-update.kerio.com/update.php?id=%s&version=%s.0&tag=", cfg.LicenseNumber, version)
		resp, err := utils.HttpGetWithRetry(url, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("IDSv%s: request error: %v", version, err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			logger.Warnf("IDSv%s: bad status: %d", version, resp.StatusCode)
			continue
		}
		lines, err := utils.ReadLines(resp.Body)
		if err != nil {
			logger.Errorf("IDSv%s: read body error: %v", version, err)
			continue
		}
		var remoteVersion int
		var downloadLink string
		parseErr := false
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			kv := utils.SplitKV(line, ':')
			if len(kv) != 2 {
				parseErr = true
				break
			}
			key, value := kv[0], kv[1]
			if key == "0" {
				parts := utils.SplitKV(value, '.')
				if len(parts) != 2 {
					parseErr = true
					break
				}
				remoteVersion = utils.AtoiSafe(parts[1])
			} else if key == "full" {
				downloadLink = value
			} else {
				logger.Warnf("IDSv%s: error: %s", version, line)
				parseErr = true
				break
			}
		}
		if parseErr || downloadLink == "" || remoteVersion == 0 {
			logger.Warnf("IDSv%s: parse error or no update", version)
			continue
		}
		// get current version from DB
		currentVersion := db.GetIDSVersion(conn, version)
		if currentVersion == 0 {
			logger.Infof("IDSv%s: can't get current version from DB, continuing", version)
			continue
		}
		if currentVersion >= remoteVersion {
			logger.Infof("IDSv%s: no new version, current: %d, remote: %d", version, currentVersion, remoteVersion)
			continue
		}
		logger.Infof("IDSv%s: downloading new version: %d", version, remoteVersion)
		os.MkdirAll("mirror/ids", 0755)
		filename := filepath.Base(downloadLink)
		destPath := filepath.Join("mirror/ids", filename)
		if !utils.DownloadFileWithProxy(downloadLink, destPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("IDSv%s: failed to download main file", version)
			continue
		}
		if version == "1" || version == "2" || version == "3" || version == "5" {
			sigPath := destPath + ".sig"
			sigUrl := downloadLink + ".sig"
			if !utils.DownloadFileWithProxy(sigUrl, sigPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
				logger.Errorf("IDSv%s: failed to download signature file", version)
				continue
			}
		}
		err = db.UpdateIDSVersion(conn, version, remoteVersion, filename)
		if err != nil {
			logger.Errorf("IDSv%s: failed to update version in DB: %v", version, err)
			continue
		}
		logger.Infof("IDSv%s: downloaded new version - %d", version, remoteVersion)

		// Cleanup old files for this version
		oldFiles, err := db.GetOldIDSFiles(conn, version)
		if err != nil {
			logger.Errorf("IDSv%s: failed to get old files from DB: %v", version, err)
			continue
		}
		for _, oldFile := range oldFiles {
			oldPath := filepath.Join("mirror/ids", oldFile)
			if err := os.Remove(oldPath); err != nil {
				logger.Warnf("IDSv%s: failed to remove old file %s: %v", version, oldFile, err)
			} else {
				logger.Infof("IDSv%s: removed old file %s", version, oldFile)
			}
			// Also try to remove signature file if it exists
			sigPath := oldPath + ".sig"
			if err := os.Remove(sigPath); err == nil {
				logger.Infof("IDSv%s: removed old signature file %s", version, oldFile+".sig")
			}
		}
	}
}
