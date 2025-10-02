package mirror

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"kerio-mirror-go/config"
	"kerio-mirror-go/db"
	"kerio-mirror-go/utils"
)

// DownloadAndProcessGeo downloads a CSV file, processes its content, and saves the result.
func DownloadAndProcessGeo(url, outputFilename string, modify bool, logger func(string, ...any)) (string, error) {
	saveDir := "mirror/geo"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	outputPath := filepath.Join(saveDir, outputFilename)

	logger("Downloading file: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error downloading: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	if modify {
		// Process data in memory without creating a temporary file
		reader := csv.NewReader(resp.Body)
		header, err := reader.Read()
		if err != nil {
			return "", fmt.Errorf("error reading header: %w", err)
		}

		var rows [][]string
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("error reading row: %w", err)
			}
			if len(row) >= 3 {
				if row[1] != "" {
					row[2] = row[1]
				} else if row[2] != "" {
					row[1] = row[2]
				}
			}
			rows = append(rows, row)
		}

		// Write processed data to the output file
		f, err := os.Create(outputPath)
		if err != nil {
			return "", fmt.Errorf("error creating output file: %w", err)
		}
		defer f.Close()

		w := csv.NewWriter(f)
		if err := w.Write(header); err != nil {
			return "", fmt.Errorf("error writing header: %w", err)
		}
		if err := w.WriteAll(rows); err != nil {
			return "", fmt.Errorf("error writing rows: %w", err)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return "", fmt.Errorf("error flushing writer: %w", err)
		}
	} else {
		// Save file without modifications using streaming
		f, err := os.Create(outputPath)
		if err != nil {
			return "", fmt.Errorf("error creating output file: %w", err)
		}
		defer f.Close()

		// Use buffered copy for better performance
		buf := make([]byte, 32*1024) // 32KB buffer
		_, err = io.CopyBuffer(f, resp.Body, buf)
		if err != nil {
			return "", fmt.Errorf("error copying data: %w", err)
		}
	}

	logger("File downloaded, processed and saved at %s", outputPath)
	return outputPath, nil
}

// CombineAndCompressGeoFiles combines two geo CSVs, extracts first two columns, and gzips the result.
func CombineAndCompressGeoFiles(v4Filename, v6Filename string, logger func(string, ...any)) (string, error) {
	saveDir := "mirror/geo"
	v4Path := filepath.Join(saveDir, v4Filename)
	v6Path := filepath.Join(saveDir, v6Filename)
	fileVersion := time.Now().Format("20060102")
	outputGzPath := filepath.Join(saveDir, fmt.Sprintf("full-4-%s.gz", fileVersion))

	try := func() error {
		gzf, err := os.Create(outputGzPath)
		if err != nil {
			return err
		}
		defer gzf.Close()
		gw := gzip.NewWriter(gzf)
		w := csv.NewWriter(gw)
		for _, path := range []string{v4Path, v6Path} {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			r := csv.NewReader(bufio.NewReader(f))
			_, err = r.Read() // skip header
			if err != nil {
				f.Close()
				return err
			}
			for {
				row, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					f.Close()
					return err
				}
				if len(row) >= 2 {
					if err := w.Write(row[:2]); err != nil {
						f.Close()
						return err
					}
				}
			}
			f.Close()
		}
		w.Flush()
		gw.Close()
		return nil
	}

	maxAttempts := 5
	delay := 2 * time.Second
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := try(); err == nil {
			fi, err := os.Stat(outputGzPath)
			if err == nil && fi.Size() > 0 {
				logger("File created successfully. Size: %d bytes", fi.Size())
				return outputGzPath, nil
			}
			lastErr = errors.New("file not created or empty")
		} else {
			lastErr = err
		}
		logger("Attempt %d/%d: error during file processing: %v", attempt, maxAttempts, lastErr)
		if attempt < maxAttempts {
			logger("Pausing for %d seconds before next attempt...", int(delay.Seconds()))
			time.Sleep(delay)
		}
	}
	return "", lastErr
}

// UpdateGeoIPDatabases handles downloading, processing, combining, and DB update for GeoIP databases.
func UpdateGeoIPDatabases(conn *sql.DB, cfg *config.Config, logger *logrus.Logger) {
	v4Path, err := DownloadAndProcessGeo(cfg.GeoIP4URL, "v4.csv", true, logger.Infof)
	if err != nil {
		logger.Errorf("GeoIP4 download error: %v", err)
	}
	v6Path, err := DownloadAndProcessGeo(cfg.GeoIP6URL, "v6.csv", true, logger.Infof)
	if err != nil {
		logger.Errorf("GeoIP6 download error: %v", err)
	}
	if v4Path != "" && v6Path != "" {
		outputPath, err := CombineAndCompressGeoFiles("v4.csv", "v6.csv", logger.Infof)
		if err != nil {
			logger.Errorf("GeoIP combine error: %v", err)
		} else if outputPath != "" {
			fileVersion := time.Now().Format("20060102")
			version := utils.AtoiSafe(fileVersion)
			filename := filepath.Base(outputPath)
			if updateErr := db.UpdateIDSVersion(conn, "4", version, filename, true, time.Now()); updateErr != nil {
				logger.Errorf("Failed to update GeoIP version in DB: %v", updateErr)
			} else {
				logger.Infof("GeoIP update complete, version 4.%s", fileVersion)
			}
		}
	}
}

// DownloadGeoLocations downloads and processes the locations file if configured.
func DownloadGeoLocations(cfg *config.Config, logger *logrus.Logger) {
	_, err := DownloadAndProcessGeo(cfg.GeoLocURL, "locations.csv", false, logger.Infof)
	if err != nil {
		logger.Errorf("GeoLoc download error: %v", err)
	}
}
