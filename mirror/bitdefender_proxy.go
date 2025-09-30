package mirror

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"kerio-mirror-go/config"
	"kerio-mirror-go/utils"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Список файлов, которые НЕ должны кэшироваться (всегда запрашиваются с сервера)
var nonCacheableFiles = []string{
	"versions.id",    // Информация о версиях баз
	"version.txt",    // Альтернативный файл версий
	"cumulative.txt", // Кумулятивная информация
}

// shouldCache проверяет, должен ли файл кэшироваться
func shouldCache(filePath string) bool {
	// Используем path.Base() для URL путей (не filepath.Base для файловых путей)
	fileName := path.Base(filePath)
	for _, nonCacheable := range nonCacheableFiles {
		if strings.EqualFold(fileName, nonCacheable) {
			return false
		}
	}
	return true
}

// shouldCacheWithLog проверяет, должен ли файл кэшироваться (с логированием для отладки)
func shouldCacheWithLog(filePath string, logger *logrus.Logger) bool {
	// Используем path.Base() для URL путей (не filepath.Base для файловых путей)
	fileName := path.Base(filePath)
	logger.Debugf("Checking if file should be cached: %s (base: %s)", filePath, fileName)
	for _, nonCacheable := range nonCacheableFiles {
		if strings.EqualFold(fileName, nonCacheable) {
			logger.Debugf("File %s matches non-cacheable pattern: %s", fileName, nonCacheable)
			return false
		}
	}
	logger.Debugf("File %s is cacheable", fileName)
	return true
}

// BitdefenderProxyHandler обрабатывает запросы к Bitdefender в режиме прокси с кэшированием
func BitdefenderProxyHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Получаем путь запроса
		requestPath := c.Request().URL.Path
		if requestPath == "" || requestPath == "/" {
			logger.Warnf("Bitdefender proxy: empty request path")
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		// Формируем путь к локальному кэшированному файлу
		localPath := filepath.Join("mirror/bitdefender", filepath.Clean(requestPath))

		// Проверка на path traversal
		absBase, err := filepath.Abs("mirror/bitdefender")
		if err != nil {
			logger.Errorf("Bitdefender proxy: failed to get absolute path for base dir: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		absFile, err := filepath.Abs(localPath)
		if err != nil {
			logger.Errorf("Bitdefender proxy: failed to get absolute path for file %s: %v", localPath, err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		if !strings.HasPrefix(absFile, absBase) {
			logger.Warnf("Bitdefender proxy: path traversal attempt: %s", requestPath)
			return c.String(http.StatusForbidden, "403 Forbidden")
		}

		// Проверяем, должен ли файл кэшироваться
		cacheable := shouldCacheWithLog(requestPath, logger)
		logger.Infof("Bitdefender proxy: file %s cacheable=%v", path.Base(requestPath), cacheable)

		// Если файл некэшируемый, но существует в кэше - удаляем его
		if !cacheable {
			if _, err := os.Stat(localPath); err == nil {
				logger.Infof("Bitdefender proxy: deleting old cached non-cacheable file: %s", localPath)
				if err := os.Remove(localPath); err != nil {
					logger.Errorf("Bitdefender proxy: failed to delete cached file: %v", err)
				}
			}
			logger.Infof("Bitdefender proxy: file %s is non-cacheable, always fetching from remote", path.Base(requestPath))
		} else {
			// Если файл кэшируемый, проверяем наличие в кэше
			if _, err := os.Stat(localPath); err == nil {
				// Файл уже закэширован, отдаём его
				logger.Infof("Bitdefender proxy: serving cached file: %s", localPath)
				return c.File(localPath)
			}
		}

		// Файл не найден в кэше, запрашиваем с удалённого сервера
		// Проверяем и корректируем базовый URL
		baseURL := cfg.BitdefenderProxyBaseURL
		if baseURL == "" {
			baseURL = "https://upgrade.bitdefender.com"
			logger.Warnf("Bitdefender proxy: BitdefenderProxyBaseURL is empty, using default: %s", baseURL)
		}
		// Убираем двойные слеши при конкатенации URL
		baseURL = strings.TrimSuffix(baseURL, "/")
		remoteURL := baseURL + requestPath
		logger.Infof("Bitdefender proxy: fetching from remote: %s (using proxy: %s)", remoteURL, cfg.ProxyURL)

		// Создаём HTTP клиент с поддержкой прокси
		client, err := utils.CreateHTTPClient(cfg.ProxyURL)
		if err != nil {
			logger.Errorf("Bitdefender proxy: failed to create HTTP client: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}

		if cfg.ProxyURL != "" {
			logger.Debugf("Bitdefender proxy: using HTTP proxy: %s", cfg.ProxyURL)
		}

		// Выполняем запрос к удалённому серверу
		resp, err := client.Get(remoteURL)
		if err != nil {
			logger.Errorf("Bitdefender proxy: failed to fetch from remote: %v", err)
			return c.String(http.StatusBadGateway, "502 Bad Gateway")
		}
		defer resp.Body.Close()

		// Проверяем статус ответа
		if resp.StatusCode != http.StatusOK {
			logger.Warnf("Bitdefender proxy: remote server returned status %d for %s", resp.StatusCode, remoteURL)
			return c.String(resp.StatusCode, http.StatusText(resp.StatusCode))
		}

		// Если файл не должен кэшироваться, просто проксируем его клиенту
		if !cacheable {
			c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			c.Response().WriteHeader(resp.StatusCode)
			_, err := io.Copy(c.Response().Writer, resp.Body)
			if err != nil {
				logger.Errorf("Bitdefender proxy: failed to proxy non-cacheable file to client: %v", err)
			} else {
				logger.Infof("Bitdefender proxy: proxied non-cacheable file: %s", path.Base(requestPath))
			}
			return nil
		}

		// Создаём директорию для кэша, если её нет
		localDir := filepath.Dir(localPath)
		if err := os.MkdirAll(localDir, 0755); err != nil {
			logger.Errorf("Bitdefender proxy: failed to create cache directory %s: %v", localDir, err)
			// Всё равно отдаём файл клиенту, просто не кэшируем
			c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			c.Response().WriteHeader(resp.StatusCode)
			_, copyErr := io.Copy(c.Response().Writer, resp.Body)
			if copyErr != nil {
				logger.Errorf("Bitdefender proxy: failed to copy response to client: %v", copyErr)
			}
			return nil
		}

		// Создаём временный файл для сохранения
		tempFile, err := os.CreateTemp(localDir, "bitdefender_*.tmp")
		if err != nil {
			logger.Errorf("Bitdefender proxy: failed to create temp file: %v", err)
			// Отдаём файл без кэширования
			c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			c.Response().WriteHeader(resp.StatusCode)
			_, copyErr := io.Copy(c.Response().Writer, resp.Body)
			if copyErr != nil {
				logger.Errorf("Bitdefender proxy: failed to copy response to client: %v", copyErr)
			}
			return nil
		}
		tempFilePath := tempFile.Name()
		defer os.Remove(tempFilePath) // Удалим временный файл в случае ошибки

		// Одновременно записываем в файл и отправляем клиенту
		c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		c.Response().WriteHeader(resp.StatusCode)

		// Используем MultiWriter для записи одновременно в файл и ответ клиенту
		multiWriter := io.MultiWriter(tempFile, c.Response().Writer)
		_, err = io.Copy(multiWriter, resp.Body)
		tempFile.Close()

		if err != nil {
			logger.Errorf("Bitdefender proxy: failed to save and send file: %v", err)
			return nil // Ответ уже начали отправлять
		}

		// Переименовываем временный файл в целевой
		if err := os.Rename(tempFilePath, localPath); err != nil {
			logger.Errorf("Bitdefender proxy: failed to rename temp file to %s: %v", localPath, err)
			return nil // Файл уже отправлен клиенту
		}

		logger.Infof("Bitdefender proxy: cached file: %s", localPath)
		return nil
	}
}