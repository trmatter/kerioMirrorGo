package mirror

import (
	"database/sql"
	"fmt"
	"time"
	"io"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// UpdateWebFilterKey implements the python logic for fetching and storing the Web Filter key
func UpdateWebFilterKey(conn *sql.DB, cfg *config.Config, logger *logrus.Logger) {
	if cfg.LicenseNumber == "" {
		logger.Infof("Web Filter: passing because license key is not configured")
		return
	}

	key, err := db.GetWebfilterKey(conn, cfg.LicenseNumber)
	if err != nil {
		logger.Errorf("Web Filter: DB error: %v", err)
		return
	}
	if key != "" {
		logger.Infof("Web Filter: database already contains an actual Web Filter key")
		return
	}

	logger.Info("Fetching new Web Filter key from wf-activation.kerio.com server")
	url := "https://wf-activation.kerio.com/getkey.php?id=" + cfg.LicenseNumber + "&tag="

	// Try direct, then proxy if set
	attempts := []struct {
		desc  string
		proxy string
	}{
		{"without proxy", ""},
	}
	if cfg.ProxyURL != "" {
		attempts = append(attempts, struct{ desc, proxy string }{"with proxy", cfg.ProxyURL})
	}

	for _, att := range attempts {
		resp, err := utils.HttpGetWithRetry(url, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, att.proxy)
		if err != nil {
			logger.Warnf("Error fetching Web Filter key %s: %v", att.desc, err)
			continue
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("Web Filter: read body error: %v", err)
			continue
		}
		text := string(data)
		if resp.StatusCode != 200 {
			logger.Warnf("Web Filter: bad status: %d", resp.StatusCode)
			continue
		}
		if contains(text, "Invalid product license") {
			msg := fmt.Sprintf("Web Filter: invalid license key. %s", cfg.LicenseNumber)
			logger.Warn(msg)
			cfg.LicenseNumber = ""
			return
		}
		if contains(text, "Product Software Maintenance expired") {
			msg := fmt.Sprintf("Web Filter: license key expired. %s", cfg.LicenseNumber)
			logger.Warn(msg)
			cfg.LicenseNumber = ""
			return
		}
		if text != "" {
			err = db.AddWebfilterKey(conn, cfg.LicenseNumber, text)
			if err != nil {
				logger.Errorf("Web Filter: failed to save key: %v", err)
				return
			}
			msg := fmt.Sprintf("Web Filter: received new key - %s", text)
			logger.Info(msg)
			return
		}
	}
	logger.Error("Web Filter: error fetching Web Filter key")
}
