package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Init(path string) error {
	db, err := sql.Open("sqlite3", path)
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
  filename TEXT
);
CREATE TABLE IF NOT EXISTS bitdefender (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  version INTEGER
);
    `
	_, err = db.Exec(schema)
	return err
}
