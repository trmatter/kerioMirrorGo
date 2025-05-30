package mirror

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// downloadAndStoreBitdefender handles Bitdefender update with backup/rollback logic
func downloadAndStoreBitdefender(conn *sql.DB, resType string, urls []string, destDir string, cfg *config.Config, logger *logrus.Logger) {
	if resType == "bitdefender" {
		// --- Bitdefender update with backup/rollback logic ---
		// bakDir := destDir + ".bak"
		// 1. Backup existing bitdefender dir
		// if _, err := os.Stat(destDir); err == nil {
		// 	os.RemoveAll(bakDir)
		// 	if err := os.Rename(destDir, bakDir); err != nil {
		// 		logger.Errorf("bitdefender: failed to backup old dir: %v", err)
		// 		return
		// 	}
		// }
		// os.MkdirAll(destDir, 0755)
		// success := false
		// defer func() {
		// 	if !success {
		// 		logger.Warn("bitdefender: update failed, restoring backup")
		// 		os.RemoveAll(destDir)
		// 		if _, err := os.Stat(bakDir); err == nil {
		// 			_ = os.Rename(bakDir, destDir)
		// 		}
		// 	} else {
		// 		os.RemoveAll(bakDir)
		// 	}
		// }()

		// 1. Получаем XML с информацией о версиях
		const versionUrl = "https://upgrade.bitdefender.com/av64bit/versions.id"
		resp, err := utils.HttpGetWithRetry(versionUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch versions.id: %v", err)
			return
		}
		defer resp.Body.Close()

		// Сохраняем versions.id с сохранением структуры директорий
		urlPath := "/av64bit/versions.id"
		if len(urlPath) > 0 && urlPath[0] == '/' {
			urlPath = urlPath[1:]
		}
		versionsPath := filepath.Join(destDir, urlPath)
		os.MkdirAll(filepath.Dir(versionsPath), 0755)
		if err := utils.SaveResponseToFile(resp.Body, versionsPath); err != nil {
			logger.Errorf("bitdefender: failed to save versions.id: %v", err)
			return
		}
		logger.Infof("Stored bitdefender -> %s", urlPath)

		// Reopen the file for XML parsing
		versionsFile, err := os.Open(versionsPath)
		if err != nil {
			logger.Errorf("bitdefender: failed to reopen versions.id: %v", err)
			return
		}
		defer versionsFile.Close()

		type V3 struct {
			IdPath  string `xml:"id_path,attr"`
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
		var info Info
		if err := utils.DecodeXML(versionsFile, &info); err != nil {
			logger.Errorf("bitdefender: failed to parse XML: %v", err)
			return
		}
		newVersion := utils.AtoiSafe(info.All.ID.Value)
		currentVersion := db.GetBitdefenderVersion(conn)
		if currentVersion >= newVersion {
			logger.Infof("bitdefender: no new version, current: %d, remote: %d", currentVersion, newVersion)
			// success = true
			return
		}
		logger.Infof("bitdefender: new version detected: %d", newVersion)

		// скачиваем файлы versions.dat
		versionsDatUrl := "https://upgrade.bitdefender.com/av64bit_" + strconv.Itoa(newVersion) + "/versions.dat"
		urlPath = "av64bit_" + strconv.Itoa(newVersion) + "/versions.dat"
		destPath := filepath.Join(destDir, urlPath)
		os.MkdirAll(filepath.Dir(destPath), 0755)
		respDat, err := utils.HttpGetWithRetry(versionsDatUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch versions.dat: %v", err)
		} else {
			if err := utils.SaveResponseToFile(respDat.Body, destPath); err != nil {
				logger.Errorf("bitdefender: failed to save versions.dat: %v", err)
			} else {
				logger.Infof("Stored bitdefender -> %s", urlPath)
			}
		}

		// скачиваем versions.sig
		versionsSigUrl := "https://upgrade.bitdefender.com/av64bit_" + strconv.Itoa(newVersion) + "/versions.sig"
		urlPathSig := "av64bit_" + strconv.Itoa(newVersion) + "/versions.sig"
		destPathSig := filepath.Join(destDir, urlPathSig)
		os.MkdirAll(filepath.Dir(destPathSig), 0755)
		respSig, err := utils.HttpGetWithRetry(versionsSigUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch versions.sig: %v", err)
		} else {
			if err := utils.SaveResponseToFile(respSig.Body, destPathSig); err != nil {
				logger.Errorf("bitdefender: failed to save versions.sig: %v", err)
			} else {
				logger.Infof("Stored bitdefender -> %s", urlPathSig)
			}
		}

		// скачиваем versions.dat.gz
		versionsDatGzUrl := "https://upgrade.bitdefender.com/av64bit_" + strconv.Itoa(newVersion) + "/versions.dat.gz"
		urlPathGz := "av64bit_" + strconv.Itoa(newVersion) + "/versions.dat.gz"
		destPathGz := filepath.Join(destDir, urlPathGz)
		os.MkdirAll(filepath.Dir(destPathGz), 0755)
		respDatGz, err := utils.HttpGetWithRetry(versionsDatGzUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch versions.dat.gz: %v", err)
		} else {
			if err := utils.SaveResponseToFile(respDatGz.Body, destPathGz); err != nil {
				logger.Errorf("bitdefender: failed to save versions.dat.gz: %v", err)
			} else {
				logger.Infof("Stored bitdefender -> %s", urlPathGz)
			}
		}

		// скачиваем versions.id по пути https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64/versions.id
		versionsIdUrl := "https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64/versions.id"
		urlPathId := "as-thin-sdk-win-x86_64/versions.id"
		destPathId := filepath.Join(destDir, urlPathId)
		os.MkdirAll(filepath.Dir(destPathId), 0755)
		respId, err := utils.HttpGetWithRetry(versionsIdUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("bitdefender: failed to fetch versions.id for as-thin-sdk-win-x86_64: %v", err)
			return
		} else {
			if err := utils.SaveResponseToFile(respId.Body, destPathId); err != nil {
				logger.Errorf("bitdefender: failed to save versions.id for as-thin-sdk-win-x86_64: %v", err)
				return
			} else {
				logger.Infof("Stored bitdefender -> %s", urlPathId)
			}
		}

		// 2. Скачиваем архивы id_path, dat_path, sig_path
		archiveUrls := []struct{ path, name string }{
			{info.V3.IdPath, "id"},
			{info.V3.DatPath, "dat"},
			{info.V3.SigPath, "sig"},
		}
		for _, arch := range archiveUrls {
			if arch.path == "" {
				continue
			}
			url := "https://upgrade.bitdefender.com/" + arch.path
			filename := filepath.Base(arch.path)
			destPath := filepath.Join(destDir, filename)
			if !utils.DownloadFileWithProxy(url, destPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
				logger.Errorf("bitdefender: failed to download %s archive", arch.name)
				return
			}
		}

		// 3. Распаковываем dat_path архив и читаем JSON
		datArchive := filepath.Join(destDir, filepath.Base(info.V3.DatPath))
		jsonData, err := utils.ExtractFirstFileFromGzip(datArchive)
		if err != nil {
			logger.Errorf("bitdefender: failed to extract dat archive: %v", err)
			return
		}
		type BitdefenderFile struct {
			LocalPath string `json:"local_path"`
			URL       string `json:"url"`
		}
		type BitdefenderDat struct {
			Files []BitdefenderFile `json:"files"`
		}
		var dat BitdefenderDat
		if err := utils.DecodeJSON(jsonData, &dat); err != nil {
			logger.Errorf("bitdefender: failed to parse dat JSON: %v", err)
			return
		}

		// 4. Скачиваем все файлы из dat
		for _, f := range dat.Files {
			if f.URL == "" || f.LocalPath == "" {
				continue
			}
			besesVersion := newVersion
			gzipUrl := fmt.Sprintf("https://upgrade.bitdefender.com/av64bit_%d/avx/%s.gzip", besesVersion, f.LocalPath)
			gzipDest := filepath.Join(destDir, fmt.Sprintf("av64bit_%d/avx/%s.gzip", besesVersion, f.LocalPath))
			os.MkdirAll(filepath.Dir(gzipDest), 0755)
			if !utils.DownloadFileWithProxy(gzipUrl, gzipDest, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
				logger.Errorf("bitdefender: failed to download gzip %s", gzipUrl)
			} else {
				logger.Infof("Stored bitdefender gzip -> %s", gzipDest)
			}
		}

		err = db.UpdateBitdefenderVersion(conn, newVersion)
		if err != nil {
			logger.Errorf("bitdefender: failed to update version: %v", err)
			return
		} else {
			logger.Infof("bitdefender: update complete, version %d", newVersion)
		}
		// success = true
		return
	}
	for _, url := range urls {
		filename := filepath.Base(url)
		destPath := filepath.Join(destDir, filename)
		resp, err := utils.HttpGetWithRetry(url, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
		if err != nil {
			logger.Errorf("%s download error: %v", resType, err)
			continue
		}
		if err := utils.SaveResponseToFile(resp.Body, destPath); err != nil {
			logger.Errorf("Save %s file error: %v", resType, err)
			continue
		}
		checksum := calcChecksum(destPath)
		db.InsertFileRecord(conn, resType, filename, url, checksum)
		logger.Infof("Stored %s -> %s", resType, filename)
	}
}
