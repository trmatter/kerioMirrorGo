package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/mirror"

	"database/sql"

	"html/template"

	"embed"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// DashboardStatus holds info for the dashboard page
type DashboardStatus struct {
	ServiceName        string
	CurrentTime        string
	Config             *config.Config
	IDSVersions        map[string]int
	IDSSuccess         map[string]bool // успешность по каждой IDS
	BitdefenderVer     int
	BitdefenderSuccess bool // успешность Bitdefender
	LastUpdate         string
}

func getDashboardStatus(cfg *config.Config) (*DashboardStatus, error) {
	conn, err := sql.Open("sqlite3", cfg.DatabasePath)
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

	// Получаем время последнего обновления из last_update
	lastUpdateStr, _ := db.GetLastUpdate(conn)

	return &DashboardStatus{
		ServiceName:        "Kerio Mirror Go",
		CurrentTime:        time.Now().Format("2006-01-02 15:04:05 MST"),
		Config:             cfg,
		IDSVersions:        idsVersions,
		IDSSuccess:         idsSuccess,
		BitdefenderVer:     bitdefenderVer,
		BitdefenderSuccess: bitdefenderSuccess,
		LastUpdate:         lastUpdateStr,
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
	// Start manual update mirror files
	e.GET("/update", updateHandler(cfg, logger))
	// Раздать файлы обновлений
	e.GET("/update.php", updateKerioHandler(cfg, logger))
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
	// Static files from embedded filesystem
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(embeddedFiles)))) // Serve embedded static files
	// other routes
	e.GET("/*", bitdefenderOrUnknownHandler(cfg, logger))

}

func dashboardHandler(embeddedFiles embed.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := c.Get("logger").(*logrus.Logger)
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		cfg := c.Get("config").(*config.Config)
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
		logger := c.Get("logger").(*logrus.Logger)
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		if c.Request().Method == http.MethodPost {
			cfg.ScheduleTime = c.FormValue("ScheduleTime")
			cfg.ScheduleInterval, _ = strconv.Atoi(c.FormValue("ScheduleInterval"))
			cfg.DatabasePath = c.FormValue("DatabasePath")
			cfg.LogPath = c.FormValue("LogPath")
			cfg.ProxyURL = c.FormValue("ProxyURL")
			cfg.LicenseNumber = c.FormValue("LicenseNumber")
			cfg.WebFilterApi = c.FormValue("WebFilterApi")
			cfg.GeoIP4Url = c.FormValue("GeoIP4Url")
			cfg.GeoIP6Url = c.FormValue("GeoIP6Url")
			cfg.GeoLocUrl = c.FormValue("GeoLocUrl")
			cfg.RetryCount, _ = strconv.Atoi(c.FormValue("RetryCount"))
			cfg.RetryDelaySeconds, _ = strconv.Atoi(c.FormValue("RetryDelaySeconds"))

			cfg.IDSUrl = c.FormValue("IDSUrls")
			bitdefUrlsRaw := c.FormValue("BitdefenderUrls")
			cfg.BitdefenderUrls = nil
			for _, line := range strings.Split(bitdefUrlsRaw, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					cfg.BitdefenderUrls = append(cfg.BitdefenderUrls, line)
				}
			}
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
		logger := c.Get("logger").(*logrus.Logger)
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
		go mirror.MirrorUpdate(cfg, logger)
		return c.Redirect(http.StatusSeeOther, "/logs")
	}
}

func updateKerioHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		version := c.QueryParam("version")
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
		if majorVersion == 0 {
			return c.String(http.StatusOK, "0:0.0")
		} else if majorVersion == 9 || majorVersion == 10 {
			return c.String(http.StatusOK, "THDdir=https://bdupdate.kerio.com/../")
		}

		// Regular versions (1-5)
		if majorVersion >= 1 && majorVersion <= 5 {
			// Get current version from DB
			conn, err := sql.Open("sqlite3", cfg.DatabasePath)
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
		logger := c.Get("logger").(*logrus.Logger)
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		if cfg.LicenseNumber == "" {
			return c.String(http.StatusNotFound, "404 Not found")
		}
		conn, err := sql.Open("sqlite3", cfg.DatabasePath)
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

// Универсальный обработчик: если host Bitdefender — раздаём базу, иначе unknown
func bitdefenderOrUnknownHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		host := c.Request().Host
		if strings.Contains(host, "bdupdate.kerio.com") || strings.Contains(host, "bda-update.kerio.com") {
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
		logger.Infof("Web access: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())

		// Get the requested file path after /control-update/
		filePath := c.Param("*")
		if filePath == "" {
			logger.Warnf("Control update handler: missing file path in request %s", c.Request().URL.Path)
			return c.String(http.StatusBadRequest, "400 Bad Request")
		}

		// Construct the full path to the file within the mirror directory
		localPath := filepath.Join("mirror", filepath.Clean(filePath))

		// Prevent directory traversal attacks
		absBase, err := filepath.Abs("mirror")
		if err != nil {
			logger.Errorf("Control update handler: failed to get absolute path for update_files: %v", err)
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
		logger.Infof("Serving file from update_files: %s", localPath)
		return c.File(localPath)
	}
}
