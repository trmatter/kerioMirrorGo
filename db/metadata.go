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

// UpdateIDSVersion обновляет версию, имя файла и статус обновления для IDS
func UpdateIDSVersion(db *sql.DB, version string, newVersion int, filename string, success bool, lastSuccessAt time.Time) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO ids_versions(version_id, version, filename, last_update_success, last_success_update_at) VALUES(?,?,?,?,?)`, "ids"+version, newVersion, filename, success, lastSuccessAt)
	return err
}

// UpdateIDSVersionLegacy оставлена для обратной совместимости
func UpdateIDSVersionLegacy(db *sql.DB, version string, newVersion int, filename string) error {
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

// UpdateBitdefenderVersion обновляет версию и статус обновления для Bitdefender
func UpdateBitdefenderVersion(db *sql.DB, newVersion int, success bool, lastSuccessAt time.Time) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO bitdefender(id, version, last_update_success, last_success_update_at) VALUES(1, ?, ?, ?)`, newVersion, success, lastSuccessAt)
	return err
}

// UpdateBitdefenderVersionLegacy оставлена для обратной совместимости
func UpdateBitdefenderVersionLegacy(db *sql.DB, newVersion int) error {
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

// SetLastUpdate сохраняет текущее время как время последнего обновления
func SetLastUpdate(db *sql.DB, t time.Time) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO last_update (id, updated_at) VALUES (1, ?)`, t)
	return err
}

// GetLastUpdate возвращает время последнего обновления
func GetLastUpdate(db *sql.DB) (string, error) {
	var updatedAt sql.NullString
	err := db.QueryRow(`SELECT updated_at FROM last_update WHERE id = 1`).Scan(&updatedAt)
	if err != nil || !updatedAt.Valid {
		return "-", err
	}
	return updatedAt.String, nil
}

// GetIDSUpdateStatus возвращает статус последнего обновления и дату последнего удачного обновления для IDS
func GetIDSUpdateStatus(db *sql.DB, version string) (bool, string, error) {
	var success bool
	var lastSuccessAt sql.NullString
	err := db.QueryRow(`SELECT last_update_success, last_success_update_at FROM ids_versions WHERE version_id = ?`, "ids"+version).Scan(&success, &lastSuccessAt)
	if err != nil {
		return false, "", err
	}
	return success, lastSuccessAt.String, nil
}

// UpdateIDSUpdateStatus обновляет статус последнего обновления IDS
func UpdateIDSUpdateStatus(db *sql.DB, version string, success bool, lastSuccessAt time.Time) error {
	_, err := db.Exec(`UPDATE ids_versions SET last_update_success = ?, last_success_update_at = ? WHERE version_id = ?`, success, lastSuccessAt, "ids"+version)
	return err
}

// GetBitdefenderUpdateStatus возвращает статус последнего обновления и дату последнего удачного обновления для Bitdefender
func GetBitdefenderUpdateStatus(db *sql.DB) (bool, string, error) {
	var success bool
	var lastSuccessAt sql.NullString
	err := db.QueryRow(`SELECT last_update_success, last_success_update_at FROM bitdefender WHERE id = 1`).Scan(&success, &lastSuccessAt)
	if err != nil {
		return false, "", err
	}
	return success, lastSuccessAt.String, nil
}

// UpdateBitdefenderUpdateStatus обновляет статус последнего обновления Bitdefender
func UpdateBitdefenderUpdateStatus(db *sql.DB, success bool, lastSuccessAt time.Time) error {
	_, err := db.Exec(`UPDATE bitdefender SET last_update_success = ?, last_success_update_at = ? WHERE id = 1`, success, lastSuccessAt)
	return err
}
