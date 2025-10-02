package utils

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SaveResponseToFile saves HTTP response body to the given path
func SaveResponseToFile(body io.ReadCloser, destPath string) error {
	defer body.Close()
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
		return err
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, body)
	return err
}

// CleanupOldFiles removes files older than maxAgeDays or exceeding maxFiles per subdir
func CleanupOldFiles(rootDir string, maxAgeDays, maxFiles int) error {
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	// iterate subdirectories
	subdirs, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}
	for _, sd := range subdirs {
		dirPath := filepath.Join(rootDir, sd.Name())
		if !sd.IsDir() {
			continue
		}
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		// collect files with info
		type fileInfo struct {
			name string
			mod  time.Time
		}
		var files []fileInfo
		for _, f := range entries {
			fi, err := f.Info()
			if err == nil {
				files = append(files, fileInfo{f.Name(), fi.ModTime()})
			}
		}
		// sort by mod time desc
		sort.Slice(files, func(i, j int) bool {
			return files[i].mod.After(files[j].mod)
		})
		// remove beyond maxFiles
		for i, file := range files {
			full := filepath.Join(dirPath, file.name)
			if i >= maxFiles || file.mod.Before(cutoff) {
				_ = os.Remove(full) // ignore error, file may already be deleted
			}
		}
	}
	return nil
}

// ReadLines reads all lines from an io.Reader
func ReadLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// SplitKV splits a string by the first sep and trims spaces
func SplitKV(s string, sep rune) []string {
	parts := strings.SplitN(s, string(sep), 2)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// AtoiSafe converts string to int, returns 0 on error
func AtoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// DownloadFileWithProxy downloads a file with optional proxy and retry
func DownloadFileWithProxy(url, destPath, proxyURL string, retries int, delay time.Duration, logger *logrus.Logger) bool {
	resp, err := HTTPGetWithRetry(url, retries, delay, proxyURL)
	if err != nil {
		logger.Errorf("Download error: %v", err)
		return false
	}
	defer resp.Body.Close()
	err = SaveResponseToFile(resp.Body, destPath)
	if err != nil {
		logger.Errorf("Save file error: %v", err)
		return false
	}
	return true
}

// DecodeXML decodes XML from reader into v
func DecodeXML(r io.Reader, v interface{}) error {
	return xml.NewDecoder(r).Decode(v)
}

// DecodeJSON decodes JSON from bytes into v
func DecodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ExtractFirstFileFromGzip extracts the first file from a gzip archive and returns its bytes
func ExtractFirstFileFromGzip(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, gz)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
