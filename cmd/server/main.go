package main

import (
	"embed"
	"flag"
	"log"
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

	// Start HTTP server (port 80)
	go func() {
		defer wg.Done()
		if err := e.Start(":80"); err != nil {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// Start HTTPS server (port 443)
	go func() {
		defer wg.Done()
		if err := e.StartTLS(":443", "cert.pem", "key.pem"); err != nil {
			logger.Errorf("HTTPS server error: %v", err)
		}
	}()

	// Wait for both servers
	wg.Wait()
}
