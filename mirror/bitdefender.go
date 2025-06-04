package mirror

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// downloadAndStoreBitdefender handles Bitdefender update with backup/rollback logic
func downloadAndStoreBitdefender(conn *sql.DB, urls []string, destDir string, cfg *config.Config, logger *logrus.Logger) {
	// Создаем временную папку для загрузки
	tmpDir := destDir + "_tmp"
	os.RemoveAll(tmpDir)                    // очищаем если осталась от прошлого раза
	defer func() { os.RemoveAll(tmpDir) }() // на случай ошибки

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
	versionsPath := filepath.Join(tmpDir, urlPath)
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
	// decode and close immediately after use
	var info Info
	decodeErr := utils.DecodeXML(versionsFile, &info)
	closeErr := versionsFile.Close()
	if decodeErr != nil {
		logger.Errorf("bitdefender: failed to parse XML: %v", decodeErr)
		return
	}
	if closeErr != nil {
		logger.Errorf("bitdefender: failed to close versions.id: %v", closeErr)
		return
	}
	newVersion := utils.AtoiSafe(info.All.ID.Value)
	currentVersion := db.GetBitdefenderVersion(conn)
	if currentVersion >= newVersion {
		logger.Infof("bitdefender: no new version, current: %d, remote: %d", currentVersion, newVersion)
		return
	}
	logger.Infof("bitdefender: new version detected: %d", newVersion)

	// скачиваем файлы versions.dat
	versionsDatUrl := "https://upgrade.bitdefender.com/av64bit_" + strconv.Itoa(newVersion) + "/versions.dat"
	urlPath = "av64bit_" + strconv.Itoa(newVersion) + "/versions.dat"
	destPath := filepath.Join(tmpDir, urlPath)
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
	destPathSig := filepath.Join(tmpDir, urlPathSig)
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
	destPathGz := filepath.Join(tmpDir, urlPathGz)
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
	destPathId := filepath.Join(tmpDir, urlPathId)
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
			logger.Infof("Stored bitdefender for as-thin-sdk-win-x86_64 -> %s", urlPathId)
		}
	}

	// смотрим id value в xml versions.id для as-thin-sdk-win-x86_64
	asThinSdkVersionsFile, err := os.Open(destPathId)
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
	// Скачиваем versions.dat по id
	thinDatUrl := "https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64_" + thinID + "/versions.dat"
	thinDatPath := filepath.Join(tmpDir, "as-thin-sdk-win-x86_64_"+thinID, "versions.dat")
	os.MkdirAll(filepath.Dir(thinDatPath), 0755)
	respThinDat, err := utils.HttpGetWithRetry(thinDatUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
	if err != nil {
		logger.Errorf("bitdefender: failed to fetch as-thin-sdk-win-x86_64_[id]/versions.dat: %v", err)
	}
	defer respThinDat.Body.Close()
	if err := utils.SaveResponseToFile(respThinDat.Body, thinDatPath); err != nil {
		logger.Errorf("bitdefender: failed to save as-thin-sdk-win-x86_64_[id]/versions.dat: %v", err)
	}
	logger.Infof("Stored bitdefender -> %s", thinDatPath)

	// Скачиваем versions.dat.gz по id и просто сохраняем
	thinDatGzUrl := "https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64_" + thinID + "/versions.dat.gz"
	thinDatGzPath := filepath.Join(tmpDir, "as-thin-sdk-win-x86_64_"+thinID, "versions.dat.gz")
	os.MkdirAll(filepath.Dir(thinDatGzPath), 0755)
	respThinDatGz, err := utils.HttpGetWithRetry(thinDatGzUrl, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
	if err != nil {
		logger.Errorf("bitdefender: failed to fetch as-thin-sdk-win-x86_64_[id]/versions.dat.gz: %v", err)
	} else {
		defer respThinDatGz.Body.Close()
		if err := utils.SaveResponseToFile(respThinDatGz.Body, thinDatGzPath); err != nil {
			logger.Errorf("bitdefender: failed to save as-thin-sdk-win-x86_64_[id]/versions.dat.gz: %v", err)
		} else {
			logger.Infof("Stored bitdefender -> %s", thinDatGzPath)
		}
	}

	// Читаем versions.dat из as-thin-sdk-win-x86_64_[id] (ожидаем список файлов)
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
		fileUrl := "https://upgrade.bitdefender.com/as-thin-sdk-win-x86_64_" + thinID + "/avx/" + filename
		fileDest := filepath.Join(tmpDir, "as-thin-sdk-win-x86_64_"+thinID, filename)
		os.MkdirAll(filepath.Dir(fileDest), 0755)
		if !utils.DownloadFileWithProxy(fileUrl, fileDest, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("bitdefender: failed to download as-thin-sdk-win-x86_64_[id] file %s", fileUrl)
			continue
		}
		logger.Debugf("Stored bitdefender as-thin-sdk-win-x86_64_[id] -> %s", fileDest)

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
		destPath := filepath.Join(tmpDir, filename)
		if !utils.DownloadFileWithProxy(url, destPath, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("bitdefender: failed to download %s archive", arch.name)
			return
		}
		logger.Debugf("bitdefender: downloaded %s archive", arch.name)
	}

	// 3. Распаковываем dat_path архив и читаем JSON
	datArchive := filepath.Join(tmpDir, filepath.Base(info.V3.DatPath))
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
		gzipDest := filepath.Join(tmpDir, fmt.Sprintf("av64bit_%d/avx/%s.gzip", besesVersion, f.LocalPath))
		os.MkdirAll(filepath.Dir(gzipDest), 0755)
		if !utils.DownloadFileWithProxy(gzipUrl, gzipDest, cfg.ProxyURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, logger) {
			logger.Errorf("bitdefender: failed to download gzip %s", gzipUrl)
		} else {
			logger.Debugf("Stored bitdefender gzip -> %s", gzipDest)
		}
	}

	// Если дошли до сюда — все скачано успешно, меняем рабочую папку
	os.RemoveAll(destDir + "_bak")
	// переименовываем старую рабочую папку в _bak
	if err := os.Rename(destDir, destDir+"_bak"); err != nil {
		logger.Errorf("bitdefender: failed to backup old data: %v", err)
	}
	// переименовываем временную папку в рабочую
	if err := os.Rename(tmpDir, destDir); err != nil {
		logger.Errorf("bitdefender: failed to move new data to working dir: %v", err)
		// Откат: пытаемся вернуть старую папку
		if restoreErr := os.Rename(destDir+"_bak", destDir); restoreErr != nil {
			logger.Errorf("bitdefender: rollback failed, could not restore backup: %v", restoreErr)
		} else {
			logger.Warn("bitdefender: rollback successful, old data restored")
		}
		return
	}
	os.RemoveAll(destDir + "_bak") // удаляем бэкап

	err = db.UpdateBitdefenderVersion(conn, newVersion, true, time.Now())
	if err != nil {
		logger.Errorf("bitdefender: failed to update version: %v", err)
		return
	} else {
		logger.Infof("bitdefender: update complete, version %d", newVersion)
	}
	logger.Info(urls)

}

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
type ThinSdkID struct {
	All struct {
		ID struct {
			Value string `xml:"value,attr"`
		} `xml:"id"`
	} `xml:"all"`
}
