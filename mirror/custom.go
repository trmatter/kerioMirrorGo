package mirror

import (
	"kerio-mirror-go/config"
	"kerio-mirror-go/utils"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DownloadCustomFiles скачивает все файлы из CustomDownloadURLs, сохраняя относительный путь
func DownloadCustomFiles(cfg *config.Config, logger *logrus.Logger) {
	customDir := "mirror/custom"
	for _, url := range cfg.CustomDownloadURLs {
		if url == "" {
			continue
		}
		relPath := getRelativePathFromURL(url)
		if relPath == "" {
			logger.Warnf("Cannot determine relative path for custom URL: %s", url)
			continue
		}
		destPath := customDir + "/" + relPath
		ok := utils.DownloadFileWithProxy(url, destPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger)
		if ok {
			logger.Infof("Downloaded custom file: %s", destPath)
		} else {
			logger.Warnf("Failed to download custom file: %s", url)
		}
	}
}

// getRelativePathFromURL extracts the relative path from a URL (without scheme and host)
func getRelativePathFromURL(urlStr string) string {
	// Remove scheme
	parts := strings.SplitN(urlStr, "://", 2)
	if len(parts) == 2 {
		urlStr = parts[1]
	}
	// Remove host
	idx := strings.Index(urlStr, "/")
	if idx >= 0 {
		return urlStr[idx+1:]
	}
	return ""
}
