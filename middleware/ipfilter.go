package middleware

import (
	"net"
	"net/http"
	"strings"

	"kerio-mirror-go/config"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// IPFilterMiddleware creates a middleware that filters requests based on IP whitelist/blacklist
func IPFilterMiddleware(cfg *config.Config, logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if no IP restrictions configured
			if len(cfg.AllowedIPs) == 0 && len(cfg.BlockedIPs) == 0 {
				return next(c)
			}

			clientIP := getClientIP(c)
			if clientIP == "" {
				logger.Warnf("IP filter: unable to determine client IP")
				return c.String(http.StatusForbidden, "403 Forbidden")
			}

			// Parse client IP
			ip := net.ParseIP(clientIP)
			if ip == nil {
				logger.Warnf("IP filter: invalid client IP: %s", clientIP)
				return c.String(http.StatusForbidden, "403 Forbidden")
			}

			// Check blacklist first (takes priority)
			if len(cfg.BlockedIPs) > 0 {
				if matchesIPList(ip, cfg.BlockedIPs) {
					logger.Warnf("IP filter: blocked IP %s attempting to access %s", clientIP, c.Request().URL.Path)
					return c.String(http.StatusForbidden, "403 Forbidden")
				}
			}

			// Check whitelist
			if len(cfg.AllowedIPs) > 0 {
				if !matchesIPList(ip, cfg.AllowedIPs) {
					logger.Warnf("IP filter: unauthorized IP %s attempting to access %s", clientIP, c.Request().URL.Path)
					return c.String(http.StatusForbidden, "403 Forbidden")
				}
			}

			logger.Debugf("IP filter: allowed IP %s to access %s", clientIP, c.Request().URL.Path)
			return next(c)
		}
	}
}

// getClientIP extracts the real client IP from various headers or RemoteAddr
func getClientIP(c echo.Context) string {
	// Try X-Real-IP header first
	if ip := c.Request().Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Try X-Forwarded-For header
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to RemoteAddr
	ip := c.Request().RemoteAddr
	// Remove port if present
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

// matchesIPList checks if the given IP matches any IP or CIDR in the list
func matchesIPList(ip net.IP, list []string) bool {
	for _, entry := range list {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Check if entry is a CIDR notation
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err != nil {
				// Invalid CIDR, skip
				continue
			}
			if ipNet.Contains(ip) {
				return true
			}
		} else {
			// Single IP address
			entryIP := net.ParseIP(entry)
			if entryIP != nil && entryIP.Equal(ip) {
				return true
			}
		}
	}
	return false
}
