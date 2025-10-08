package mirror

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"

	"github.com/sirupsen/logrus"
)

// UpdateShieldMatrix проверяет и обновляет Shield Matrix (Kerio 9.5+)
func UpdateShieldMatrix(conn *sql.DB, cfg *config.Config, logger *logrus.Logger) {
	logger.Debug("Shield Matrix: starting update check...")

	if !cfg.EnableShieldMatrix {
		logger.Info("Shield Matrix: update is disabled by config")
		return
	}

	if cfg.ShieldMatrixBaseURL == "" {
		logger.Warn("Shield Matrix: base URL is not configured")
		return
	}

	logger.Infof("Shield Matrix: checking for updates (base URL: %s)", cfg.ShieldMatrixBaseURL)

	// Получаем текущую версию из БД
	currentVersion := db.GetShieldMatrixVersion(conn)
	logger.Infof("Shield Matrix: current version in DB: '%s'", currentVersion)

	// Формируем URL для проверки версии
	versionURL := fmt.Sprintf("%s/version", cfg.ShieldMatrixBaseURL)
	logger.Debugf("Shield Matrix: requesting version from: %s", versionURL)

	// Запрашиваем версию с сервера
	resp, err := utils.HTTPGetWithRetry(versionURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
	if err != nil {
		logger.Errorf("Shield Matrix: failed to check version: %v", err)
		db.UpdateShieldMatrixVersion(conn, currentVersion, false, time.Now())
		return
	}
	defer resp.Body.Close()

	logger.Debugf("Shield Matrix: version check response status: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		logger.Errorf("Shield Matrix: bad status code: %d", resp.StatusCode)
		db.UpdateShieldMatrixVersion(conn, currentVersion, false, time.Now())
		return
	}

	// Читаем версию
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Shield Matrix: failed to read version response: %v", err)
		db.UpdateShieldMatrixVersion(conn, currentVersion, false, time.Now())
		return
	}

	remoteVersion := strings.TrimSpace(string(body))
	logger.Infof("Shield Matrix: remote version: '%s' (current: '%s')", remoteVersion, currentVersion)

	// Проверяем версию и наличие файлов (если включена предзагрузка)
	if remoteVersion == currentVersion {
		// Версия актуальная
		if cfg.ShieldMatrixPreloadFiles {
			// При включенной предзагрузке проверяем наличие файлов
			if checkShieldMatrixFilesExist(logger) {
				logger.Info("Shield Matrix: already up to date, all files exist")
				db.UpdateShieldMatrixVersion(conn, currentVersion, true, time.Now())
				return
			}
			// Файлы отсутствуют, нужно загрузить
			logger.Warn("Shield Matrix: version is up to date but files are missing, re-downloading...")
		} else {
			// При on-demand режиме файлы не нужны
			logger.Info("Shield Matrix: already up to date, no changes needed")
			db.UpdateShieldMatrixVersion(conn, currentVersion, true, time.Now())
			return
		}
	} else {
		// Новая версия доступна
		logger.Infof("Shield Matrix: new version available: %s -> %s", currentVersion, remoteVersion)
	}

	// Shield Matrix использует модель "загрузка по запросу" (on-demand)
	// Файлы не скачиваются заранее, а загружаются только когда Kerio Control их запрашивает
	// Поэтому здесь мы только создаём директории и очищаем старые данные

	// Создаём директории
	matrixDir := "mirror/matrix"
	ipv4Dir := filepath.Join(matrixDir, "ipv4")
	ipv6Dir := filepath.Join(matrixDir, "ipv6")

	// Очищаем старые данные
	logger.Infof("Shield Matrix: cleaning old data directories: %s, %s", ipv4Dir, ipv6Dir)
	if err := os.RemoveAll(ipv4Dir); err != nil {
		logger.Warnf("Shield Matrix: error removing ipv4 dir: %v", err)
	}
	if err := os.RemoveAll(ipv6Dir); err != nil {
		logger.Warnf("Shield Matrix: error removing ipv6 dir: %v", err)
	}

	logger.Debugf("Shield Matrix: creating directory: %s", ipv4Dir)
	if err := os.MkdirAll(ipv4Dir, 0755); err != nil {
		logger.Errorf("Shield Matrix: failed to create ipv4 directory: %v", err)
		db.UpdateShieldMatrixVersion(conn, currentVersion, false, time.Now())
		return
	}

	logger.Debugf("Shield Matrix: creating directory: %s", ipv6Dir)
	if err := os.MkdirAll(ipv6Dir, 0755); err != nil {
		logger.Errorf("Shield Matrix: failed to create ipv6 directory: %v", err)
		db.UpdateShieldMatrixVersion(conn, currentVersion, false, time.Now())
		return
	}

	// Проверяем, нужно ли предзагружать файлы
	if cfg.ShieldMatrixPreloadFiles {
		logger.Info("Shield Matrix: preload mode enabled, downloading all files...")
		PreloadShieldMatrixFiles(cfg, logger)
	} else {
		logger.Info("Shield Matrix: directories prepared, files will be downloaded on-demand when requested by Kerio Control")
	}

	// Обновляем версию в БД
	logger.Debugf("Shield Matrix: updating version in DB: %s -> %s", currentVersion, remoteVersion)
	if err := db.UpdateShieldMatrixVersion(conn, remoteVersion, true, time.Now()); err != nil {
		logger.Errorf("Shield Matrix: failed to update version in DB: %v", err)
		return
	}

	if cfg.ShieldMatrixPreloadFiles {
		logger.Infof("Shield Matrix: successfully updated to version %s (DB updated, all files preloaded)", remoteVersion)
	} else {
		logger.Infof("Shield Matrix: successfully updated to version %s (DB updated, directories ready)", remoteVersion)
	}
}

// DownloadShieldMatrixFile загружает один файл Shield Matrix по запросу
// Используется в HTTP обработчике когда Kerio Control запрашивает файл
func DownloadShieldMatrixFile(subpath string, cfg *config.Config, logger *logrus.Logger) error {
	// Формируем URL для загрузки
	// Базовый URL из конфига: https://d2akeya8d016xi.cloudfront.net/9.5.0
	downloadURL := fmt.Sprintf("%s/%s", cfg.ShieldMatrixBaseURL, subpath)

	logger.Infof("Shield Matrix: initiating on-demand download for: %s", subpath)
	logger.Debugf("Shield Matrix: download URL: %s", downloadURL)

	resp, err := utils.HTTPGetWithRetry(downloadURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
	if err != nil {
		logger.Errorf("Shield Matrix: download failed for %s: %v", subpath, err)
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	logger.Debugf("Shield Matrix: download response status: %d for %s", resp.StatusCode, subpath)

	if resp.StatusCode != 200 {
		logger.Errorf("Shield Matrix: bad status code %d for %s", resp.StatusCode, subpath)
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// Определяем путь для сохранения
	savePath := filepath.Join("mirror", "matrix", subpath)
	logger.Debugf("Shield Matrix: saving to: %s", savePath)

	// Создаём директорию если нужно
	dirPath := filepath.Dir(savePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		logger.Errorf("Shield Matrix: failed to create directory %s: %v", dirPath, err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Сохраняем файл
	out, err := os.Create(savePath)
	if err != nil {
		logger.Errorf("Shield Matrix: failed to create file %s: %v", savePath, err)
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		logger.Errorf("Shield Matrix: failed to save file %s: %v", savePath, err)
		return fmt.Errorf("failed to save file: %w", err)
	}

	logger.Infof("Shield Matrix: successfully downloaded %s (%d bytes) -> %s", subpath, written, savePath)
	return nil
}

// checkShieldMatrixFilesExist проверяет существование всех необходимых файлов Shield Matrix
// Возвращает true если все файлы threat_data_1.dat до threat_data_5.dat существуют для IPv4 и IPv6
func checkShieldMatrixFilesExist(logger *logrus.Logger) bool {
	logger.Debug("Shield Matrix: checking if all files exist...")

	allFilesExist := true
	missingFiles := []string{}

	// Проверка IPv4 файлов
	for i := 1; i <= 5; i++ {
		filePath := filepath.Join("mirror", "matrix", fmt.Sprintf("ipv4/threat_data_%d.dat", i))
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, filePath)
			allFilesExist = false
		}
	}

	// Проверка IPv6 файлов
	for i := 1; i <= 5; i++ {
		filePath := filepath.Join("mirror", "matrix", fmt.Sprintf("ipv6/threat_data_%d.dat", i))
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, filePath)
			allFilesExist = false
		}
	}

	if !allFilesExist {
		logger.Warnf("Shield Matrix: missing files detected: %v", missingFiles)
		return false
	}

	logger.Debug("Shield Matrix: all files exist")
	return true
}

// PreloadShieldMatrixFiles загружает все файлы Shield Matrix заранее (по расписанию)
// Скачивает файлы threat_data_1.dat до threat_data_5.dat для IPv4 и IPv6
func PreloadShieldMatrixFiles(cfg *config.Config, logger *logrus.Logger) {
	logger.Info("Shield Matrix: starting preload of all files...")

	totalFiles := 0
	ipv4Files := 0
	ipv6Files := 0

	// Загрузка IPv4 файлов
	logger.Debug("Shield Matrix: preloading IPv4 threat data files...")
	for i := 1; i <= 5; i++ { // Shield Matrix использует файлы threat_data_1.dat до threat_data_5.dat
		subpath := fmt.Sprintf("ipv4/threat_data_%d.dat", i)
		err := DownloadShieldMatrixFile(subpath, cfg, logger)
		if err != nil {
			// Если получили ошибку (скорее всего 404), прекращаем загрузку IPv4
			logger.Debugf("Shield Matrix: stopped IPv4 preload at file %d (error: %v)", i, err)
			break
		}
		ipv4Files++
		totalFiles++
	}

	// Загрузка IPv6 файлов
	logger.Debug("Shield Matrix: preloading IPv6 threat data files...")
	for i := 1; i <= 5; i++ { // Shield Matrix использует файлы threat_data_1.dat до threat_data_5.dat
		subpath := fmt.Sprintf("ipv6/threat_data_%d.dat", i)
		err := DownloadShieldMatrixFile(subpath, cfg, logger)
		if err != nil {
			// Если получили ошибку (скорее всего 404), прекращаем загрузку IPv6
			logger.Debugf("Shield Matrix: stopped IPv6 preload at file %d (error: %v)", i, err)
			break
		}
		ipv6Files++
		totalFiles++
	}

	logger.Infof("Shield Matrix: preload completed - %d files total (IPv4: %d, IPv6: %d)", totalFiles, ipv4Files, ipv6Files)
}
