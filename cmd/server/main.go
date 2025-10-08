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
			errMsg := err.Error()
			if strings.Contains(errMsg, "address already in use") || strings.Contains(errMsg, "Only one usage of each socket address") {
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
			errMsg := err.Error()
			if strings.Contains(errMsg, "Only one usage of each socket address") {
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
			} else if strings.Contains(errMsg, "The system cannot find the file specified") {
				logger.Error("========================================")
				logger.Error("SSL CERTIFICATE FILES NOT FOUND")
				logger.Error("========================================")
				logger.Error("The HTTPS server requires cert.pem and key.pem files.")
				logger.Error("")
				logger.Error("Option 1: Generate self-signed certificate in Kerio Control interface:")
				logger.Error("  Go to  https://control.local/admin/#sslCertificates")
				logger.Error("  Click on 'Add' -> 'New Certificate'")
				logger.Error("  Enter the following information:")
				logger.Error("    Name: KerioMirror (or any other name)")
				logger.Error("    Hostname: KerioMirror (or any other name)")
				logger.Error("    Alternative Hostnames: All domains that you use for updates. See readme.md")
				logger.Error("    Click on 'OK'")
				logger.Error("  Click right mouse button on the certificate and select 'Export' -> 'Export Certificate in PEM' and 'Export Private Key in PEM'")
				logger.Error("  Save the certificate and key files to the working directory")
				logger.Error("  Rename the certificate file to cert.pem and the key file to key.pem")
				logger.Error("  Save the certificate and key files to the working directory")
				logger.Error("")
				logger.Error("Option 2: Generate self-signed certificate (for testing/development):")
				logger.Error("  openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj \"/CN=localhost\"")
				logger.Error("")
				logger.Error("Option 3: Use existing certificates:")
				logger.Error("  Copy your cert.pem and key.pem files to the working directory")
				logger.Error("")
				logger.Error("Note: HTTP server on port 80 will continue to work without HTTPS")
				logger.Error("========================================")
			} else if strings.Contains(errMsg, "private key does not match public key") {
				logger.Error("========================================")
				logger.Error("SSL CERTIFICATE AND KEY MISMATCH")
				logger.Error("========================================")
				logger.Error("The cert.pem and key.pem files do not match.")
				logger.Error("")
				logger.Error("Possible causes:")
				logger.Error("  - The certificate and private key are from different pairs")
				logger.Error("  - Files were corrupted during copy/export")
				logger.Error("  - Wrong files were renamed to cert.pem/key.pem")
				logger.Error("")
				logger.Error("Solution:")
				logger.Error("  1. Delete the current cert.pem and key.pem files")
				logger.Error("  2. Export both certificate and key from the same source")
				logger.Error("  3. Make sure to use matching pair of files")
				logger.Error("")
				logger.Error("Note: HTTP server on port 80 will continue to work without HTTPS")
				logger.Error("========================================")
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
