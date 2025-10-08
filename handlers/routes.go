package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/logging"
	"kerio-mirror-go/mirror"
	"kerio-mirror-go/utils"

	"database/sql"

	"html/template"

	"embed"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// DashboardStatus holds info for the dashboard page
type DashboardStatus struct {
	ServiceName           string
	CurrentTime           string
	Config                *config.Config
	IDSVersions           map[string]int
	IDSSuccess            map[string]bool // успешность по каждой IDS
	BitdefenderVer        int
	BitdefenderSuccess    bool   // успешность Bitdefender
	SnortTemplateSuccess  bool   // успешность Snort Template
	ShieldMatrixVersion   string // версия Shield Matrix
	ShieldMatrixSuccess   bool   // успешность Shield Matrix
	LastUpdate            string
}

func getDashboardStatus(cfg *config.Config) (*DashboardStatus, error) {
	conn, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	idsVersions := make(map[string]int)
	idsSuccess := make(map[string]bool)
	for _, v := range []string{"1", "2", "3", "4", "5"} {
		idsVersions[v] = db.GetIDSVersion(conn, v)
		success, _, _ := db.GetIDSUpdateStatus(conn, v)
		idsSuccess[v] = success
	}
	bitdefenderVer := db.GetBitdefenderVersion(conn)
	bitdefenderSuccess, _, _ := db.GetBitdefenderUpdateStatus(conn)

	// Получаем статус Snort Template
	snortTemplateSuccess, _, _ := db.GetSnortTemplateStatus(conn)

	// Получаем статус Shield Matrix
	shieldMatrixVersion := db.GetShieldMatrixVersion(conn)
	shieldMatrixSuccess, _, _ := db.GetShieldMatrixUpdateStatus(conn)

	// Получаем время последнего обновления из last_update
	lastUpdateStr, _ := db.GetLastUpdate(conn)

	return &DashboardStatus{
		ServiceName:          "Kerio Mirror Go",
		CurrentTime:          time.Now().Format("2006-01-02 15:04:05 MST"),
		Config:               cfg,
		IDSVersions:          idsVersions,
		IDSSuccess:           idsSuccess,
		BitdefenderVer:       bitdefenderVer,
		BitdefenderSuccess:   bitdefenderSuccess,
		SnortTemplateSuccess: snortTemplateSuccess,
		ShieldMatrixVersion:  shieldMatrixVersion,
		ShieldMatrixSuccess:  shieldMatrixSuccess,
		LastUpdate:           lastUpdateStr,
	}, nil
}

func RegisterRoutes(e *echo.Echo, cfg *config.Config, logger *logrus.Logger, embeddedFiles embed.FS) {
	// Dashboard
	e.GET("/", dashboardHandler(embeddedFiles))
	// Settings
	e.GET("/settings", settingsPageHandler(cfg, embeddedFiles))
	e.POST("/settings", settingsPageHandler(cfg, embeddedFiles))
	// Web filter key
	e.GET("/getkey.php", webFilterKeyHandler(cfg))
	// Logs
	e.GET("/logs", serveFileHandler(cfg.LogPath, embeddedFiles))
	e.GET("/logs/raw", serveRawLogHandler(cfg.LogPath))
	e.GET("/logs/full_raw", serveFullRawLogHandler(cfg.LogPath))
	// Start manual update mirror files
	e.GET("/update", updateHandler(cfg, logger))
	// Раздать файлы обновлений
	e.GET("/update.php", updateKerioHandler(cfg, logger))
	// Shield Matrix update check
	e.GET("/check_update/", shieldMatrixCheckUpdateHandler(cfg, logger))
	// Bitdefender or unknown route
	e.GET("/favicon.ico", func(c echo.Context) error {
		data, err := embeddedFiles.ReadFile("favicon.ico")
		if err != nil {
			return c.String(http.StatusNotFound, "favicon.ico not found in embedded files")
		}
		return c.Blob(http.StatusOK, "image/x-icon", data)
	})
	// New handler for serving files from the update_files directory
	e.GET("/control-update/*", controlUpdateHandler(logger))
	// Shield Matrix files
	e.GET("/matrix/*", matrixHandler(logger))
	// Static files from embedded filesystem
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(embeddedFiles)))) // Serve embedded static files
	// other routes
	e.GET("/*", customFilesHandlerOrFallback(cfg, logger))
}

func dashboardHandler(embeddedFiles embed.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger, ok := c.Get("logger").(*logrus.Logger)
		if !ok {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		cfg, ok := c.Get("config").(*config.Config)
		if !ok {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		status, err := getDashboardStatus(cfg)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to load status")
		}
		t, err := template.ParseFS(embeddedFiles, "templates/dashboard.html")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Template file error: "+err.Error())
		}
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		return t.Execute(c.Response(), status)
	}
}

func settingsPageHandler(cfg *config.Config, embeddedFiles embed.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger, ok := c.Get("logger").(*logrus.Logger)
		if !ok {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		if c.Request().Method == http.MethodPost {
			cfg.ScheduleTime = c.FormValue("ScheduleTime")
			cfg.DatabasePath = c.FormValue("DatabasePath")
			cfg.LogPath = c.FormValue("LogPath")
			cfg.ProxyURL = c.FormValue("ProxyURL")
			cfg.LicenseNumber = c.FormValue("LicenseNumber")
			cfg.WebFilterAPI = c.FormValue("WebFilterApi")
			cfg.GeoIP4URL = c.FormValue("GeoIP4Url")
			cfg.GeoIP6URL = c.FormValue("GeoIP6Url")
			cfg.GeoLocURL = c.FormValue("GeoLocUrl")
			cfg.RetryCount, _ = strconv.Atoi(c.FormValue("RetryCount"))
			cfg.RetryDelaySeconds, _ = strconv.Atoi(c.FormValue("RetryDelaySeconds"))
			cfg.LogLevel = c.FormValue("LogLevel")
			cfg.IDSURL = c.FormValue("IDSUrl")
			bitdefUrlsRaw := c.FormValue("BitdefenderUrls")
			cfg.BitdefenderURLs = nil
			for _, line := range strings.Split(bitdefUrlsRaw, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					cfg.BitdefenderURLs = append(cfg.BitdefenderURLs, line)
				}
			}
			// Bitdefender mode (взаимоисключающие опции)
			bitdefenderMode := c.FormValue("BitdefenderMode")
			cfg.EnableBitdefender = bitdefenderMode == "updates"
			cfg.BitdefenderProxyMode = bitdefenderMode == "proxy"
			cfg.BitdefenderProxyBaseURL = c.FormValue("BitdefenderProxyBaseURL")

			customUrlsRaw := c.FormValue("CustomDownloadUrls")
			cfg.CustomDownloadURLs = nil
			for _, line := range strings.Split(customUrlsRaw, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					cfg.CustomDownloadURLs = append(cfg.CustomDownloadURLs, line)
				}
			}
			cfg.EnableIDS1 = c.FormValue("EnableIDS1") == "true"
			cfg.EnableIDS2 = c.FormValue("EnableIDS2") == "true"
			cfg.EnableIDS3 = c.FormValue("EnableIDS3") == "true"
			cfg.EnableIDS4 = c.FormValue("EnableIDS4") == "true"
			cfg.EnableIDS5 = c.FormValue("EnableIDS5") == "true"
			cfg.EnableSnortTemplate = c.FormValue("EnableSnortTemplate") == "true"
			cfg.SnortTemplateURL = c.FormValue("SnortTemplateURL")
			cfg.EnableShieldMatrix = c.FormValue("EnableShieldMatrix") == "true"
			cfg.ShieldMatrixBaseURL = c.FormValue("ShieldMatrixBaseURL")
			msg := "Настройки успешно обновлены!"

			// Get config path from context
			configPath, ok := c.Get("configPath").(string)
			if !ok {
				logger.Error("Config path not found in context")
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}

			// Save the updated config
			if err := config.Save(cfg, configPath); err != nil {
				logger.Errorf("Failed to save config: %v", err)
				return c.String(http.StatusInternalServerError, "Failed to save config")
			}

			// Update logger level if it was changed
			logging.UpdateLogLevel(logger, cfg.LogLevel)

			t, err := template.ParseFS(embeddedFiles, "templates/settings.html")
			if err != nil {
				return c.String(http.StatusInternalServerError, "Template file error: "+err.Error())
			}
			c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
			return t.Execute(c.Response(), map[string]interface{}{
				"Config":  cfg,
				"Message": msg,
			})
		}
		// GET: показать форму
		t, err := template.ParseFS(embeddedFiles, "templates/settings.html")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Template file error: "+err.Error())
		}
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		return t.Execute(c.Response(), map[string]interface{}{
			"Config":  cfg,
			"Message": "",
		})
	}
}

func serveFileHandler(path string, embeddedFiles embed.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger, ok := c.Get("logger").(*logrus.Logger)
		if !ok {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		data, err := os.ReadFile(path)
		if err != nil {
			return c.String(http.StatusNotFound, "File not found")
		}
		t, err := template.ParseFS(embeddedFiles, "templates/logs.html")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Template file error: "+err.Error())
		}
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		return t.Execute(c.Response(), map[string]interface{}{
			"LogContent": string(data),
		})
	}
}

func serveRawLogHandler(path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		const maxLogLines = 1000
		file, err := os.Open(path)
		if err != nil {
			return c.String(http.StatusNotFound, "File not found")
		}
		defer file.Close()

		// Читаем все строки (или только последние 1000)
		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return c.String(http.StatusInternalServerError, "Failed to read log file")
		}

		truncated := false
		if len(lines) > maxLogLines {
			truncated = true
			lines = lines[len(lines)-maxLogLines:]
		}

		c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")
		if truncated {
			return c.String(http.StatusOK, "[ВНИМАНИЕ] Лог слишком большой, показаны только последние 1000 строк.\n\n"+strings.Join(lines, "\n"))
		}
		return c.String(http.StatusOK, strings.Join(lines, "\n"))
	}
}

func serveFullRawLogHandler(path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return c.String(http.StatusNotFound, "File not found")
		}
		c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")
		return c.String(http.StatusOK, string(data))
	}
}

func updateHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		go mirror.Update(cfg, logger)
		return c.Redirect(http.StatusSeeOther, "/logs")
	}
}

func updateKerioHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Debug logging: full request details
		logger.Debugf("=== Update.php Request ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Path: %s", c.Request().URL.Path)
		logger.Debugf("RawQuery: %s", c.Request().URL.RawQuery)
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("RemoteAddr: %s", c.Request().RemoteAddr)
		logger.Debugf("RealIP: %s", c.RealIP())
		logger.Debugf("User-Agent: %s", c.Request().UserAgent())

		version := c.QueryParam("version")
		logger.Debugf("Version param: %s", version)

		if version == "" {
			logger.Errorf("Error processing URL %s in update request", c.Request().URL.String())
			return c.String(http.StatusBadRequest, "")
		}

		logger.Infof("Received update request for version: %s", version)

		// Parse major version number
		parts := strings.Split(version, ".")
		if len(parts) == 0 {
			logger.Errorf("Invalid version format: %s", version)
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		majorVersion, err := strconv.Atoi(parts[0])
		if err != nil {
			logger.Errorf("Invalid version format: %s", version)
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		// Special cases handling
		switch majorVersion {
		case 0:
			return c.String(http.StatusOK, "0:0.0")
		case 6, 7, 8:
			// Shield Matrix для Kerio 9.5+ (версии 6, 7, 8 в update.php)
			// Возвращаем информацию о Shield Matrix
			conn, err := sql.Open("sqlite", cfg.DatabasePath)
			if err != nil {
				logger.Errorf("Failed to open database: %v", err)
				return c.String(http.StatusInternalServerError, "500 Internal Server Error")
			}
			defer conn.Close()

			shieldMatrixVersion := db.GetShieldMatrixVersion(conn)
			if shieldMatrixVersion == "" {
				logger.Warnf("Shield Matrix version not found in database for version %s", version)
				return c.String(http.StatusOK, "0:0.0")
			}
			// Формат ответа для Shield Matrix
			// Kerio Control будет загружать файлы из указанного URL
			response := fmt.Sprintf("0:%s\nmatrix:http://%s/matrix/", shieldMatrixVersion, c.Request().Host)
			logger.Infof("Responding to Shield Matrix request for version %s: %s", version, response)
			return c.String(http.StatusOK, response)
		case 9, 10:
			// Если включен режим прокси Bitdefender, перенаправляем клиента на наш сервер
			if cfg.BitdefenderProxyMode {
				return c.String(http.StatusOK, "THDdir=http://"+c.Request().Host+"/")
			}
			return c.String(http.StatusOK, "THDdir=https://bdupdate.kerio.com/../")
		}

		// Regular versions (1-5)
		if majorVersion >= 1 && majorVersion <= 5 {
			// Get current version from DB
			conn, err := sql.Open("sqlite", cfg.DatabasePath)
			if err != nil {
				logger.Errorf("Failed to open database: %v", err)
				return c.String(http.StatusInternalServerError, "500 Internal Server Error")
			}
			defer conn.Close()

			versionStr := strconv.Itoa(majorVersion)
			currentVersion := db.GetIDSVersion(conn, versionStr)
			if currentVersion == 0 {
				logger.Errorf("Failed to get IDS version %s from database", versionStr)
				return c.String(http.StatusInternalServerError, "500 Internal Server Error")
			}

			// Get filename from DB
			var filename string
			err = conn.QueryRow(`SELECT filename FROM ids_versions WHERE version_id = ?`, "ids"+versionStr).Scan(&filename)
			if err != nil {
				logger.Errorf("Failed to get filename for IDS version %s: %v", versionStr, err)
				return c.String(http.StatusInternalServerError, "500 Internal Server Error")
			}

			response := fmt.Sprintf("0:%s.%d\nfull:http://%s/control-update/%s",
				versionStr, currentVersion, c.Request().Host, filename)
			logger.Infof("Responding to update request for version %s: %s", version, response)
			return c.String(http.StatusOK, response)
		}

		// Unknown version
		logger.Errorf("Received unknown download request: %s", version)
		return c.String(http.StatusNotFound, "404 Not found")
	}
}

func webFilterKeyHandler(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger, ok := c.Get("logger").(*logrus.Logger)
		if !ok {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		if cfg.LicenseNumber == "" {
			return c.String(http.StatusNotFound, "404 Not found")
		}
		conn, err := sql.Open("sqlite", cfg.DatabasePath)
		if err != nil {
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		defer conn.Close()

		key, err := db.GetWebfilterKey(conn, cfg.LicenseNumber)
		if err != nil {
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		if key == "" {
			return c.String(http.StatusNotFound, "404 Not found")
		}
		return c.String(http.StatusOK, key)
	}
}

// Handler for Shield Matrix update check endpoint
func shieldMatrixCheckUpdateHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Debugf("=== Shield Matrix Check Update ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("User-Agent: %s", c.Request().UserAgent())

		logger.Infof("Shield Matrix check update: %s from %s", c.Request().URL.Path, c.RealIP())

		// Получаем параметры запроса
		clientID := c.QueryParam("client-id")
		version := c.QueryParam("version")
		lastUpdate := c.QueryParam("last-update")

		logger.Debugf("Shield Matrix: client-id=%s, version=%s, last-update=%s", clientID, version, lastUpdate)

		// Проверяем что Shield Matrix включен
		if !cfg.EnableShieldMatrix {
			logger.Warn("Shield Matrix: update check received but Shield Matrix is disabled")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"available": false,
			})
		}

		// Открываем соединение с БД
		conn, err := sql.Open("sqlite", cfg.DatabasePath)
		if err != nil {
			logger.Errorf("Shield Matrix: failed to open database: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		defer conn.Close()

		// Получаем текущую версию Shield Matrix из БД
		currentVersion := db.GetShieldMatrixVersion(conn)
		logger.Infof("Shield Matrix: current version in DB: '%s'", currentVersion)

		// Возвращаем текущую версию
		if currentVersion == "" {
			logger.Warn("Shield Matrix: no version in database, returning unavailable")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"available": false,
			})
		}

		logger.Infof("Shield Matrix: responding with version available, url: %s", cfg.ShieldMatrixBaseURL)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"available": true,
			"url":       cfg.ShieldMatrixBaseURL,
		})
	}
}

// Handler for Shield Matrix CloudFront proxy (on-demand download with caching)
func shieldMatrixCloudFrontHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Debugf("=== Shield Matrix CloudFront Handler ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Path: %s", c.Request().URL.Path)
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("User-Agent: %s", c.Request().UserAgent())

		logger.Infof("Shield Matrix CloudFront request: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())

		// Get the requested file path
		requestPath := c.Request().URL.Path
		if requestPath == "" || requestPath == "/" {
			logger.Warnf("Shield Matrix CloudFront: empty path in request")
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		// Extract subpath after version (e.g., /9.5.0/ipv4/threat_data_1.dat -> ipv4/threat_data_1.dat)
		// Remove leading slash
		requestPath = strings.TrimPrefix(requestPath, "/")

		// Split by slash and skip version part (first segment like "9.5.0")
		parts := strings.SplitN(requestPath, "/", 2)
		if len(parts) < 2 {
			logger.Warnf("Shield Matrix CloudFront: invalid path format: %s", requestPath)
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		subpath := parts[1] // e.g., "ipv4/threat_data_1.dat"
		logger.Debugf("Shield Matrix CloudFront: extracted subpath: '%s'", subpath)

		// Validate that this is a threat_data file request or version file
		if subpath == "version" {
			// Special case: version file request
			// Proxy to upstream CloudFront
			logger.Debugf("Shield Matrix CloudFront: version file request, proxying to upstream")

			upstreamURL := fmt.Sprintf("%s/%s", cfg.ShieldMatrixBaseURL, subpath)
			logger.Debugf("Shield Matrix CloudFront: upstream URL: %s", upstreamURL)

			resp, err := utils.HTTPGetWithRetry(upstreamURL, cfg.RetryCount, time.Duration(cfg.RetryDelaySeconds)*time.Second, cfg.ProxyURL)
			if err != nil {
				logger.Errorf("Shield Matrix CloudFront: failed to fetch version: %v", err)
				return c.String(http.StatusNotFound, "404 Not found")
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				logger.Errorf("Shield Matrix CloudFront: bad status code %d for version", resp.StatusCode)
				return c.String(http.StatusNotFound, "404 Not found")
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Errorf("Shield Matrix CloudFront: failed to read version response: %v", err)
				return c.String(http.StatusInternalServerError, "500 Internal Server Error")
			}

			logger.Infof("Shield Matrix CloudFront: serving version file (%d bytes)", len(body))
			return c.String(http.StatusOK, string(body))
		}

		if !strings.Contains(subpath, "ipv4/threat_data") && !strings.Contains(subpath, "ipv6/threat_data") {
			logger.Warnf("Shield Matrix CloudFront: invalid file request (not threat_data or version): %s", subpath)
			return c.String(http.StatusNotFound, "404 Not found")
		}

		// Build local path: mirror/matrix/{ipv4|ipv6}/...
		localPath := filepath.Join("mirror", "matrix", filepath.Clean(subpath))
		logger.Debugf("Shield Matrix CloudFront: local path to check: %s", localPath)

		// Prevent directory traversal attacks
		absBase, err := filepath.Abs("mirror/matrix")
		if err != nil {
			logger.Errorf("Shield Matrix CloudFront: failed to get absolute path for matrix dir: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		absFile, err := filepath.Abs(localPath)
		if err != nil {
			logger.Errorf("Shield Matrix CloudFront: failed to get absolute path for requested file %s: %v", localPath, err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}

		if !strings.HasPrefix(absFile, absBase) {
			logger.Warnf("Shield Matrix CloudFront: path traversal attempt: %s", subpath)
			return c.String(http.StatusForbidden, "403 Forbidden")
		}

		// Check if file exists, if not - download it on-demand
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			logger.Infof("Shield Matrix CloudFront: file not found locally (%s), initiating on-demand download", subpath)
			logger.Debugf("Shield Matrix CloudFront: stat error: %v", err)

			// Download the file
			if err := mirror.DownloadShieldMatrixFile(subpath, cfg, logger); err != nil {
				logger.Errorf("Shield Matrix CloudFront: failed to download file %s: %v", subpath, err)
				return c.String(http.StatusNotFound, "404 Not found")
			}

			// Re-stat the file to get size
			fileInfo, _ = os.Stat(localPath)
		} else {
			logger.Debugf("Shield Matrix CloudFront: file found in cache: %s (%d bytes)", localPath, fileInfo.Size())
		}

		// Serve the file
		var fileSize int64
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}
		logger.Infof("Shield Matrix CloudFront: serving file %s (%d bytes) to %s", subpath, fileSize, c.RealIP())
		return c.File(localPath)
	}
}

// Fallback handler: handles CloudFront (Shield Matrix), Bitdefender, or returns unknown route
func fallbackHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Debug logging for unknown routes
		logger.Debugf("=== Fallback Handler ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Path: %s", c.Request().URL.Path)
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("RemoteAddr: %s", c.Request().RemoteAddr)
		logger.Debugf("RealIP: %s", c.RealIP())

		host := c.Request().Host

		// Check for CloudFront (Shield Matrix) requests
		if strings.Contains(host, "cloudfront.net") || strings.Contains(host, "d2akeya8d016xi") {
			logger.Debugf("Fallback handler: CloudFront host detected, delegating to Shield Matrix handler")
			return shieldMatrixCloudFrontHandler(cfg, logger)(c)
		}

		// Check for Bitdefender requests
		if strings.Contains(host, "bdupdate.kerio.com") || strings.Contains(host, "bda-update.kerio.com") {
			// Если включен режим прокси, используем прокси-обработчик
			if cfg.BitdefenderProxyMode {
				return mirror.BitdefenderProxyHandler(cfg, logger)(c)
			}

			// Иначе работаем в обычном режиме - отдаём только локальные файлы
			filePath := c.Request().URL.Path
			if filePath == "" || filePath == "/" {
				return c.String(http.StatusBadRequest, "400 Bad Request")
			}
			localPath := filepath.Join("mirror/bitdefender", filepath.Clean(filePath))
			absBase, _ := filepath.Abs("mirror/bitdefender")
			absFile, _ := filepath.Abs(localPath)
			if !strings.HasPrefix(absFile, absBase) {
				logger.Warnf("Bitdefender handler: path traversal attempt: %s", filePath)
				return c.String(http.StatusForbidden, "403 Forbidden")
			}
			data, err := os.ReadFile(localPath)
			if err != nil {
				logger.Warnf("Bitdefender handler: file not found: %s", localPath)
				return c.String(http.StatusNotFound, "404 Not found")
			}
			contentType := http.DetectContentType(data)
			// logger.Infof("Serving Bitdefender file: %s", localPath)
			return c.Blob(http.StatusOK, contentType, data)
		}
		// иначе — стандартный unknownRouteHandler
		logger.Errorf("Unknown route: %s", c.Request().URL.Path)
		return c.String(http.StatusNotFound, "Unknown route")
	}
}

// Handler for serving files from the update_files directory
func controlUpdateHandler(logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Debug logging: full request details
		logger.Debugf("=== Control Update Request ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Path: %s", c.Request().URL.Path)
		logger.Debugf("RawQuery: %s", c.Request().URL.RawQuery)
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("RemoteAddr: %s", c.Request().RemoteAddr)
		logger.Debugf("RealIP: %s", c.RealIP())
		logger.Debugf("User-Agent: %s", c.Request().UserAgent())

		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())

		// Get the requested file path after /control-update/
		filePath := c.Param("*")
		logger.Debugf("Param('*'): %s", filePath)

		if filePath == "" {
			logger.Warnf("Control update handler: missing file path in request %s", c.Request().URL.Path)
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		// Try to find file in two locations:
		// 1. mirror/custom/control-update/ (for custom files like snort.tpl)
		// 2. mirror/ (for IDS files)

		customPath := filepath.Join("mirror", "custom", "control-update", filepath.Clean(filePath))
		directPath := filepath.Join("mirror", filepath.Clean(filePath))

		logger.Debugf("Checking custom path: %s", customPath)
		logger.Debugf("Checking direct path: %s", directPath)

		var localPath string
		if _, err := os.Stat(customPath); err == nil {
			localPath = customPath
			logger.Debugf("File found at custom path: %s", customPath)
		} else if _, err := os.Stat(directPath); err == nil {
			localPath = directPath
			logger.Debugf("File found at direct path: %s", directPath)
		} else {
			logger.Warnf("Control update handler: file not found: %s (tried %s and %s)", filePath, customPath, directPath)
			return c.String(http.StatusNotFound, "404 Not found")
		}

		// Prevent directory traversal attacks
		absBase, err := filepath.Abs("mirror")
		if err != nil {
			logger.Errorf("Control update handler: failed to get absolute path for mirror: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		absFile, err := filepath.Abs(localPath)
		if err != nil {
			logger.Errorf("Control update handler: failed to get absolute path for requested file %s: %v", localPath, err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}

		if !strings.HasPrefix(absFile, absBase) {
			logger.Warnf("Control update handler: path traversal attempt: %s", filePath)
			return c.String(http.StatusForbidden, "403 Forbidden")
		}

		// Serve the file
		logger.Infof("Serving file from control-update: %s", localPath)
		return c.File(localPath)
	}
}

// Handler for serving Shield Matrix files (on-demand download)
func matrixHandler(logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Debugf("=== Shield Matrix Handler ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Path: %s", c.Request().URL.Path)
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("User-Agent: %s", c.Request().UserAgent())

		logger.Infof("Shield Matrix request: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())

		// Get the requested file path after /matrix/
		filePath := c.Param("*")
		logger.Debugf("Shield Matrix: requested subpath: '%s'", filePath)

		if filePath == "" {
			logger.Warnf("Shield Matrix handler: missing file path in request %s", c.Request().URL.Path)
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		// Validate that this is a threat_data file request
		if !strings.Contains(filePath, "ipv4/threat_data") && !strings.Contains(filePath, "ipv6/threat_data") {
			logger.Warnf("Shield Matrix handler: invalid file request (not threat_data): %s", filePath)
			return c.String(http.StatusNotFound, "404 Not found")
		}

		// Build local path: mirror/matrix/{ipv4|ipv6}/...
		localPath := filepath.Join("mirror", "matrix", filepath.Clean(filePath))
		logger.Debugf("Shield Matrix: local path to check: %s", localPath)

		// Prevent directory traversal attacks
		absBase, err := filepath.Abs("mirror/matrix")
		if err != nil {
			logger.Errorf("Shield Matrix handler: failed to get absolute path for matrix dir: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		absFile, err := filepath.Abs(localPath)
		if err != nil {
			logger.Errorf("Shield Matrix handler: failed to get absolute path for requested file %s: %v", localPath, err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}

		if !strings.HasPrefix(absFile, absBase) {
			logger.Warnf("Shield Matrix handler: path traversal attempt: %s", filePath)
			return c.String(http.StatusForbidden, "403 Forbidden")
		}

		// Check if file exists, if not - download it on-demand
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			logger.Infof("Shield Matrix: file not found locally (%s), initiating on-demand download", filePath)
			logger.Debugf("Shield Matrix: stat error: %v", err)

			// Get config from context
			cfg, ok := c.Get("config").(*config.Config)
			if !ok {
				logger.Error("Shield Matrix handler: config not found in context")
				return c.String(http.StatusInternalServerError, "500 Internal Server Error")
			}

			// Download the file
			if err := mirror.DownloadShieldMatrixFile(filePath, cfg, logger); err != nil {
				logger.Errorf("Shield Matrix handler: failed to download file %s: %v", filePath, err)
				return c.String(http.StatusNotFound, "404 Not found")
			}

			// Re-stat the file to get size
			fileInfo, _ = os.Stat(localPath)
		} else {
			logger.Debugf("Shield Matrix: file found in cache: %s (%d bytes)", localPath, fileInfo.Size())
		}

		// Serve the file
		var fileSize int64
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}
		logger.Infof("Shield Matrix: serving file %s (%d bytes) to %s", filePath, fileSize, c.RealIP())
		return c.File(localPath)
	}
}

// Handler for serving files from the mirror/custom directory or fallback to bitdefender/unknown
func customFilesHandlerOrFallback(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	fallback := fallbackHandler(cfg, logger)
	return func(c echo.Context) error {
		// Debug logging
		logger.Debugf("=== Custom Files Handler ===")
		logger.Debugf("Method: %s", c.Request().Method)
		logger.Debugf("Full URL: %s", c.Request().URL.String())
		logger.Debugf("Path: %s", c.Request().URL.Path)
		logger.Debugf("Host: %s", c.Request().Host)
		logger.Debugf("RemoteAddr: %s", c.Request().RemoteAddr)
		logger.Debugf("RealIP: %s", c.RealIP())

		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		urlPath := c.Request().URL.Path
		if urlPath == "" || urlPath == "/" {
			return fallback(c)
		}
		localPath := filepath.Join("mirror/custom", filepath.Clean(urlPath))
		absBase, err := filepath.Abs("mirror/custom")
		if err != nil {
			logger.Errorf("Custom files handler: failed to get absolute path for custom dir: %v", err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		absFile, err := filepath.Abs(localPath)
		if err != nil {
			logger.Errorf("Custom files handler: failed to get absolute path for requested file %s: %v", localPath, err)
			return c.String(http.StatusInternalServerError, "500 Internal Server Error")
		}
		if !strings.HasPrefix(absFile, absBase) {
			logger.Warnf("Custom files handler: path traversal attempt: %s", urlPath)
			return c.String(http.StatusForbidden, "403 Forbidden")
		}
		if _, err := os.Stat(localPath); err == nil {
			logger.Infof("Serving custom file: %s", localPath)
			return c.File(localPath)
		}
		// Если файл не найден — fallback
		return fallback(c)
	}
}
