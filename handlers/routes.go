package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"os"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/mirror"

	"database/sql"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func RegisterRoutes(e *echo.Echo, cfg *config.Config, logger *logrus.Logger) {
	// Dashboard
	e.GET("/", dashboardHandler)
	// Settings
	e.GET("/settings", settingsGetHandler(cfg))
	e.POST("/settings", settingsPostHandler(cfg))
	// Web filter key
	e.GET("/getkey.php", webFilterKeyHandler(cfg))
	// Logs
	e.GET("/logs", serveFileHandler(cfg.LogPath))
	// Start manual update mirror files
	e.GET("/update", updateHandler(cfg, logger))
	// Раздать файлы обновлений
	e.GET("/update.php", updateKerioHandler(cfg, logger))
	// Bitdefender or unknown route
	e.GET("/*", bitdefenderOrUnknownHandler(cfg, logger))
}

func dashboardHandler(c echo.Context) error {
	return c.String(http.StatusOK, "<h1>Kerio Mirror Dashboard</h1><p>Use /update, /logs, /download</p>")
}

func settingsGetHandler(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, cfg)
	}
}

func settingsPostHandler(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		// For simplicity, not persisted; real impl would write .env and reload
		var newCfg config.Config
		if err := c.Bind(&newCfg); err != nil {
			return c.String(http.StatusBadRequest, "Invalid payload")
		}
		*cfg = newCfg
		return c.String(http.StatusOK, "Settings updated")
	}
}

func serveFileHandler(path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return c.String(http.StatusNotFound, "File not found")
		}
		return c.Blob(http.StatusOK, "text/plain", data)
	}
}

func updateHandler(cfg *config.Config, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		go mirror.MirrorUpdate(cfg, logger)
		return c.String(http.StatusAccepted, "Mirror update started")
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
			return c.String(http.StatusOK, response)
		}

		// Unknown version
		logger.Errorf("Received unknown download request: %s", version)
		return c.String(http.StatusNotFound, "404 Not found")
	}
}

func webFilterKeyHandler(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
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
