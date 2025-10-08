package main

import (
	"embed"
	"flag"
	"log"
	"strings"
	"sync"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/handlers"
	"kerio-mirror-go/logging"
	"kerio-mirror-go/middleware"
	"kerio-mirror-go/mirror"

	"github.com/labstack/echo/v4"
)

//go:embed templates static favicon.ico
var embeddedFiles embed.FS

func main() {
	// Parse config path
	cfgPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Init logger
	logger := logging.NewLogger(cfg.LogPath, cfg.LogLevel)
	logger.Info("Starting kerio-mirror-go")

	// Init DB
	if err := db.Init(cfg.DatabasePath); err != nil {
		logger.Fatalf("DB init error: %v", err)
	}

	// Start scheduled mirror
	go mirror.StartScheduler(cfg, logger)

	// Setup HTTP server
	e := echo.New()
	// Inject config and logger into context for all handlers
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("config", cfg)
			c.Set("logger", logger)
			c.Set("configPath", *cfgPath)
			return next(c)
		}
	})
	// Add IP filter middleware
	e.Use(middleware.IPFilterMiddleware(cfg, logger))
	handlers.RegisterRoutes(e, cfg, logger, embeddedFiles)

	// Start servers in goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Channel to capture critical errors
	errChan := make(chan error, 2)

	// Start HTTP server (port 80)
	go func() {
		defer wg.Done()
		if err := e.Start(":80"); err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				logger.Error("========================================")
				logger.Error("PORT 80 IS ALREADY IN USE")
				logger.Error("========================================")
				logger.Error("Another application is using port 80.")
				logger.Error("")
				logger.Error("To find which process is using port 80:")
				logger.Error("  Windows: netstat -ano | findstr :80")
				logger.Error("           tasklist /FI \"PID eq <PID>\"")
				logger.Error("  Linux:   sudo lsof -i :80")
				logger.Error("           sudo netstat -tulpn | grep :80")
				logger.Error("")
				logger.Error("To stop the process:")
				logger.Error("  Windows: taskkill /PID <PID> /F")
				logger.Error("  Linux:   sudo kill <PID>")
				logger.Error("========================================")
				errChan <- err
			} else {
				logger.Errorf("HTTP server error: %v", err)
			}
		}
	}()

	// Start HTTPS server (port 443)
	go func() {
		defer wg.Done()
		if err := e.StartTLS(":443", "cert.pem", "key.pem"); err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				logger.Error("========================================")
				logger.Error("PORT 443 IS ALREADY IN USE")
				logger.Error("========================================")
				logger.Error("Another application is using port 443.")
				logger.Error("")
				logger.Error("To find which process is using port 443:")
				logger.Error("  Windows: netstat -ano | findstr :443")
				logger.Error("           tasklist /FI \"PID eq <PID>\"")
				logger.Error("  Linux:   sudo lsof -i :443")
				logger.Error("           sudo netstat -tulpn | grep :443")
				logger.Error("")
				logger.Error("To stop the process:")
				logger.Error("  Windows: taskkill /PID <PID> /F")
				logger.Error("  Linux:   sudo kill <PID>")
				logger.Error("========================================")
				errChan <- err
			} else {
				logger.Errorf("HTTPS server error: %v", err)
			}
		}
	}()

	// Monitor for critical errors
	go func() {
		if err := <-errChan; err != nil {
			logger.Fatal("Application cannot start due to port conflict. Please resolve the issue and try again.")
		}
	}()

	// Wait for both servers
	wg.Wait()
}
