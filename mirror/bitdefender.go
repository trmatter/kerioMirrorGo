package mirror

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// downloadAndStoreBitdefender handles Bitdefender update with backup/rollback logic
func downloadAndStoreBitdefender(conn *sql.DB, urls []string, destDir string, cfg *config.Config, logger *logrus.Logger) {
	startBitdefenderHeartbeat(logger)

	tmpDir := destDir + "_tmp"
	os.RemoveAll(tmpDir)
	defer func() { os.RemoveAll(tmpDir) }()

	newVersion, info, err := fetchAndParseBitdefenderVersion(tmpDir, cfg, logger)
	if err != nil {
		logger.Errorf("bitdefender: %v", err)
		return
	}
	currentVersion := db.GetBitdefenderVersion(conn)
	if currentVersion >= newVersion {
		logger.Infof("bitdefender: no new version, current: %d, remote: %d", currentVersion, newVersion)
		return
	}
	logger.Infof("bitdefender: new version detected: %d", newVersion)

	downloadBitdefenderMetaFiles(tmpDir, newVersion, cfg, logger)
	handleThinSdkFiles(tmpDir, cfg, logger)
	downloadV3Archives(tmpDir, info, cfg, logger)
	dat, err := extractAndParseDatJSON(tmpDir, info, logger)
	if err != nil {
		logger.Errorf("bitdefender: %v", err)
		return
	}
	downloadDatFiles(tmpDir, newVersion, dat, cfg, logger)

	if !replaceBitdefenderDirs(destDir, tmpDir, logger) {
		return
	}

	err = db.UpdateBitdefenderVersion(conn, newVersion, true, time.Now())
	if err != nil {
		logger.Errorf("bitdefender: failed to update version: %v", err)
		return
	}
	logger.Infof("bitdefender: update complete, version %d", newVersion)
	logger.Info(urls)
}

// Heartbeat goroutine
func startBitdefenderHeartbeat(logger *logrus.Logger) {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logger.Infof("bitdefender: update in progress...")
			case <-done:
				return
			}
		}
	}()
	defer close(done)
}

func fetchAndParseBitdefenderVersion(tmpDir string, cfg *config.Config, logger *logrus.Logger) (int, Info, error) {
	const versionURL = "https://upgrade.bitdefender.com/av64bit/versions.id"
	resp, err := utils.HTTPGetWithRetry(versionURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
	if err != nil {
		return 0, Info{}, fmt.Errorf("failed to fetch versions.id: %w", err)
	}
	defer resp.Body.Close()
	urlPath := "av64bit/versions.id"
	versionsPath := filepath.Join(tmpDir, urlPath)
	if err := os.MkdirAll(filepath.Dir(versionsPath), 0755); err != nil {
		return 0, Info{}, fmt.Errorf("failed to create directory: %w", err)
	}
	if err := utils.SaveResponseToFile(resp.Body, versionsPath); err != nil {
		return 0, Info{}, fmt.Errorf("failed to save versions.id: %w", err)
	}
	logger.Infof("Stored bitdefender -> %s", urlPath)
	versionsFile, err := os.Open(versionsPath)
	if err != nil {
		return 0, Info{}, fmt.Errorf("failed to reopen versions.id: %w", err)
	}
	var info Info
	decodeErr := utils.DecodeXML(versionsFile, &info)
	closeErr := versionsFile.Close()
	if decodeErr != nil {
		return 0, Info{}, fmt.Errorf("failed to parse XML: %w", decodeErr)
	}
	if closeErr != nil {
		logger.Errorf("bitdefender: failed to close versions.id: %v", closeErr)
	}
	newVersion := utils.AtoiSafe(info.All.ID.Value)
	return newVersion, info, nil
}

func downloadBitdefenderMetaFiles(tmpDir string, newVersion int, cfg *config.Config, logger *logrus.Logger) {
	downloadAndLog := func(urlPath, url string) {
		destPath := filepath.Join(tmpDir, urlPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			logger.Errorf("bitdefender: failed to create directory for %s: %v", urlPath, err)
			return
		}
		resp, err := utils.HTTPGetWithRetry(url, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch %s: %v", urlPath, err)
			return
		}
		defer resp.Body.Close()
		if err := utils.SaveResponseToFile(resp.Body, destPath); err != nil {
			logger.Errorf("bitdefender: failed to save %s: %v", urlPath, err)
		} else {
			logger.Infof("Stored bitdefender -> %s", urlPath)
		}
	}
	downloadAndLog(fmt.Sprintf("av64bit_%d/versions.dat", newVersion), fmt.Sprintf("https://upgrade.bitdefender.com/av64bit_%d/versions.dat", newVersion))
	downloadAndLog(fmt.Sprintf("av64bit_%d/versions.sig", newVersion), fmt.Sprintf("https://upgrade.bitdefender.com/av64bit_%d/versions.sig", newVersion))
	downloadAndLog(fmt.Sprintf("av64bit_%d/versions.dat.gz", newVersion), fmt.Sprintf("https://upgrade.bitdefender.com/av64bit_%d/versions.dat.gz", newVersion))
}

func handleThinSdkFiles(tmpDir string, cfg *config.Config, logger *logrus.Logger) {
	idURL := "https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64/versions.id"
	idPath := "as-thin-sdk-win-x86_64/versions.id"
	destPathID := filepath.Join(tmpDir, idPath)
	if err := os.MkdirAll(filepath.Dir(destPathID), 0755); err != nil {
		logger.Errorf("bitdefender: failed to create directory for as-thin-sdk-win-x86_64: %v", err)
		return
	}
	respID, err := utils.HTTPGetWithRetry(idURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
	if err != nil {
		logger.Errorf("bitdefender: failed to fetch versions.id for as-thin-sdk-win-x86_64: %v", err)
		return
	}
	defer respID.Body.Close()
	if err := utils.SaveResponseToFile(respID.Body, destPathID); err != nil {
		logger.Errorf("bitdefender: failed to save versions.id for as-thin-sdk-win-x86_64: %v", err)
		return
	}
	logger.Infof("Stored bitdefender for as-thin-sdk-win-x86_64 -> %s", idPath)
	asThinSdkVersionsFile, err := os.Open(destPathID)
	if err != nil {
		logger.Errorf("bitdefender: failed to open as-thin-sdk-win-x86_64/versions.id: %v", err)
		return
	}
	var thinSdkID ThinSdkID
	decodeErr2 := utils.DecodeXML(asThinSdkVersionsFile, &thinSdkID)
	closeErr2 := asThinSdkVersionsFile.Close()
	if decodeErr2 != nil {
		logger.Errorf("bitdefender: failed to parse as-thin-sdk-win-x86_64/versions.id XML: %v", decodeErr2)
	}
	if closeErr2 != nil {
		logger.Errorf("bitdefender: failed to close as-thin-sdk-win-x86_64/versions.id: %v", closeErr2)
	}
	thinID := thinSdkID.All.ID.Value
	if thinID == "" {
		logger.Errorf("bitdefender: no id value found in as-thin-sdk-win-x86_64/versions.id")
	}
	// Скачиваем versions.dat и .gz по id
	for _, ext := range []string{"versions.dat", "versions.dat.gz"} {
		url := fmt.Sprintf("https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64_%s/%s", thinID, ext)
		path := filepath.Join(tmpDir, fmt.Sprintf("as-thin-sdk-win-x86_64_%s/%s", thinID, ext))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			logger.Errorf("bitdefender: failed to create directory for as-thin-sdk-win-x86_64_[id]: %v", err)
			continue
		}
		resp, err := utils.HTTPGetWithRetry(url, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch as-thin-sdk-win-x86_64_[id]/%s: %v", ext, err)
			continue
		}
		defer resp.Body.Close()
		if err := utils.SaveResponseToFile(resp.Body, path); err != nil {
			logger.Errorf("bitdefender: failed to save as-thin-sdk-win-x86_64_[id]/%s: %v", ext, err)
		} else {
			logger.Infof("Stored bitdefender -> %s", path)
		}
	}
	// Читаем versions.dat и скачиваем файлы
	thinDatPath := filepath.Join(tmpDir, fmt.Sprintf("as-thin-sdk-win-x86_64_%s/versions.dat", thinID))
	thinDatBytes, err := os.ReadFile(thinDatPath)
	if err != nil {
		logger.Errorf("bitdefender: failed to read as-thin-sdk-win-x86_64_[id]/versions.dat: %v", err)
	}
	lines := strings.Split(string(thinDatBytes), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		filename := parts[2] + ".gzip"
		fileURL := fmt.Sprintf("https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64_%s/avx/%s", thinID, filename)
		fileDest := filepath.Join(tmpDir, fmt.Sprintf("as-thin-sdk-win-x86_64_%s/avx/%s", thinID, filename))
		if err := os.MkdirAll(filepath.Dir(fileDest), 0755); err != nil {
			logger.Errorf("bitdefender: failed to create directory for as-thin-sdk-win-x86_64_[id] file: %v", err)
			continue
		}
		if !utils.DownloadFileWithProxy(fileURL, fileDest, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("bitdefender: failed to download as-thin-sdk-win-x86_64_[id] file %s", fileURL)
			continue
		}
		logger.Debugf("Stored bitdefender as-thin-sdk-win-x86_64_[id] -> %s", fileDest)
	}
}

func downloadV3Archives(tmpDir string, info Info, cfg *config.Config, logger *logrus.Logger) {
	archiveUrls := []struct{ path, name string }{
		{info.V3.IDPath, "id"},
		{info.V3.DatPath, "dat"},
		{info.V3.SigPath, "sig"},
	}
	for _, arch := range archiveUrls {
		if arch.path == "" {
			continue
		}
		url := "https://upgrade.bitdefender.com/" + arch.path
		filename := filepath.Base(arch.path)
		destPath := filepath.Join(tmpDir, filename)
		if !utils.DownloadFileWithProxy(url, destPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("bitdefender: failed to download %s archive", arch.name)
			return
		}
		logger.Debugf("bitdefender: downloaded %s archive", arch.name)
	}
}

func extractAndParseDatJSON(tmpDir string, info Info, logger *logrus.Logger) (BitdefenderDat, error) {
	datArchive := filepath.Join(tmpDir, filepath.Base(info.V3.DatPath))
	jsonData, err := utils.ExtractFirstFileFromGzip(datArchive)
	if err != nil {
		return BitdefenderDat{}, fmt.Errorf("failed to extract dat archive: %w", err)
	}
	var dat BitdefenderDat
	if err := utils.DecodeJSON(jsonData, &dat); err != nil {
		return BitdefenderDat{}, fmt.Errorf("failed to parse dat JSON: %w", err)
	}
	return dat, nil
}

type BitdefenderFile struct {
	LocalPath string `json:"local_path"`
	URL       string `json:"url"`
}
type BitdefenderDat struct {
	Files []BitdefenderFile `json:"files"`
}

func downloadDatFiles(tmpDir string, newVersion int, dat BitdefenderDat, cfg *config.Config, logger *logrus.Logger) {
	totalFiles := len(dat.Files)
	for i, f := range dat.Files {
		if f.URL == "" || f.LocalPath == "" {
			continue
		}
		gzipURL := fmt.Sprintf("https://upgrade.bitdefender.com/av64bit_%d/avx/%s.gzip", newVersion, f.LocalPath)
		gzipDest := filepath.Join(tmpDir, fmt.Sprintf("av64bit_%d/avx/%s.gzip", newVersion, f.LocalPath))
		if err := os.MkdirAll(filepath.Dir(gzipDest), 0755); err != nil {
			logger.Errorf("bitdefender: failed to create directory for gzip file: %v", err)
			continue
		}
		if !utils.DownloadFileWithProxy(gzipURL, gzipDest, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("bitdefender: failed to download gzip %s", gzipURL)
		} else {
			logger.Debugf("Stored bitdefender gzip -> %s", gzipDest)
		}
		if (i+1)%10 == 0 || i == totalFiles-1 {
			percent := float64(i+1) / float64(totalFiles) * 100
			logger.Infof("bitdefender: update in progress... %.1f%% (%d/%d)", percent, i+1, totalFiles)
		}
	}
}

func replaceBitdefenderDirs(destDir, tmpDir string, logger *logrus.Logger) bool {
	os.RemoveAll(destDir + "_bak")
	if err := os.Rename(destDir, destDir+"_bak"); err != nil {
		logger.Errorf("bitdefender: failed to backup old data: %v", err)
	}
	if err := os.Rename(tmpDir, destDir); err != nil {
		logger.Errorf("bitdefender: failed to move new data to working dir: %v", err)
		if restoreErr := os.Rename(destDir+"_bak", destDir); restoreErr != nil {
			logger.Errorf("bitdefender: rollback failed, could not restore backup: %v", restoreErr)
		} else {
			logger.Warn("bitdefender: rollback successful, old data restored")
		}
		return false
	}
	os.RemoveAll(destDir + "_bak")
	return true
}

type V3 struct {
	IDPath  string `xml:"id_path,attr"`
	DatPath string `xml:"dat_path,attr"`
	SigPath string `xml:"sig_path,attr"`
}
type All struct {
	ID struct {
		Value string `xml:"value,attr"`
	} `xml:"id"`
}
type Info struct {
	All All `xml:"all"`
	V3  V3  `xml:"v3"`
}
type ThinSdkID struct {
	All struct {
		ID struct {
			Value string `xml:"value,attr"`
		} `xml:"id"`
	} `xml:"all"`
}
