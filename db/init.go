package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Init(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
CREATE TABLE IF NOT EXISTS files (
  id INTEGER PRIMARY KEY,
  file_type TEXT,
  filename TEXT,
  url TEXT,
  downloaded DATETIME,
  checksum TEXT,
  local_path TEXT
);
CREATE TABLE IF NOT EXISTS webfilter (
  lic_number TEXT PRIMARY KEY,
  key TEXT
);
CREATE TABLE IF NOT EXISTS ids_versions (
  version_id TEXT PRIMARY KEY,
  version INTEGER,
  filename TEXT,
  last_update_success BOOLEAN DEFAULT 0,
  last_success_update_at DATETIME
);
CREATE TABLE IF NOT EXISTS bitdefender (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  version INTEGER,
  last_update_success BOOLEAN DEFAULT 0,
  last_success_update_at DATETIME
);
CREATE TABLE IF NOT EXISTS last_update (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  updated_at DATETIME
);
    `
	_, err = db.Exec(schema)
	if err != nil {
		return err
	}

	// Миграция: добавление новых полей, если их нет
	_, _ = db.Exec(`ALTER TABLE ids_versions ADD COLUMN last_update_success BOOLEAN DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE ids_versions ADD COLUMN last_success_update_at DATETIME`)
	_, _ = db.Exec(`ALTER TABLE bitdefender ADD COLUMN last_update_success BOOLEAN DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE bitdefender ADD COLUMN last_success_update_at DATETIME`)

	return nil
}
