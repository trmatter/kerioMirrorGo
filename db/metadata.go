package db

import (
	"database/sql"
	"time"
)

func GetExistingFiles(db *sql.DB, resourceType string) ([]string, error) {
	rows, err := db.Query(`SELECT filename FROM files WHERE file_type = ?`, resourceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []string
	for rows.Next() {
		var fname string
		if err := rows.Scan(&fname); err == nil {
			files = append(files, fname)
		}
	}
	return files, nil
}

func InsertFileRecord(db *sql.DB, resourceType, filename, url, checksum string) error {
	_, err := db.Exec(`INSERT INTO files(file_type, filename, url, downloaded, checksum) VALUES(?,?,?,?,?)`,
		resourceType, filename, url, time.Now(), checksum)
	return err
}

// GetIDSVersion returns current version for IDS type from DB
func GetIDSVersion(db *sql.DB, version string) int {
	var v int
	err := db.QueryRow(`SELECT version FROM ids_versions WHERE version_id = ?`, "ids"+version).Scan(&v)
	if err != nil {
		return 0
	}
	return v
}

// UpdateIDSVersion updates version and filename for IDS type in DB
func UpdateIDSVersion(db *sql.DB, version string, newVersion int, filename string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO ids_versions(version_id, version, filename) VALUES(?,?,?)`, "ids"+version, newVersion, filename)
	return err
}

// GetWebfilterKey returns the webfilter key for a given license number
func GetWebfilterKey(db *sql.DB, licNumber string) (string, error) {
	var key string
	err := db.QueryRow(`SELECT key FROM webfilter WHERE lic_number = ?`, licNumber).Scan(&key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return key, nil
}

// AddWebfilterKey inserts a new webfilter key for a license number
func AddWebfilterKey(db *sql.DB, licNumber, key string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO webfilter (lic_number, key) VALUES (?, ?)`, licNumber, key)
	return err
}

// GetBitdefenderVersion returns current version for Bitdefender from DB
func GetBitdefenderVersion(db *sql.DB) int {
	var v int
	err := db.QueryRow(`SELECT version FROM bitdefender`).Scan(&v)
	if err != nil {
		return 0
	}
	return v
}

// UpdateBitdefenderVersion updates version for Bitdefender in DB
func UpdateBitdefenderVersion(db *sql.DB, newVersion int) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO bitdefender(id, version) VALUES(1, ?)`, newVersion)
	return err
}

// GetOldIDSFiles returns a list of old IDS files for a specific version
func GetOldIDSFiles(db *sql.DB, version string) ([]string, error) {
	var files []string
	rows, err := db.Query(`SELECT filename FROM ids_versions WHERE version_id = ? AND filename != (SELECT filename FROM ids_versions WHERE version_id = ? ORDER BY version DESC LIMIT 1)`, "ids"+version, "ids"+version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		files = append(files, filename)
	}
	return files, nil
}
