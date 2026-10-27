package upgrade

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// CheckLegacyFingerprint accepts only known PocketBase-backed SubFlow shapes.
// Alpha versions intentionally resolve to the generic "legacy" source label;
// no version is inferred from incomplete or mutable schema clues.
func CheckLegacyFingerprint(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// SQLite URI paths on Windows need a leading slash before the drive letter
	// (file:///C:/...), while Unix absolute paths already have one.
	uriPath := "/" + strings.TrimPrefix(filepath.ToSlash(absolute), "/")
	uri := (&url.URL{Scheme: "file", Path: uriPath}).String() + "?mode=ro"
	database, err := sql.Open("sqlite", uri)
	if err != nil {
		return ErrUnknownSchema
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return ErrUnknownSchema
	}
	rows, err := database.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return ErrUnknownSchema
	}
	defer rows.Close()
	tables := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return ErrUnknownSchema
		}
		tables[name] = true
	}
	if err := rows.Err(); err != nil {
		return ErrUnknownSchema
	}
	if !tables["_collections"] || !tables["users"] || !tables["groups"] {
		return ErrUnknownSchema
	}
	knownBusinessTables := 0
	for _, name := range []string{"expenses", "incomes", "subscriptions", "audit_logs"} {
		if tables[name] {
			knownBusinessTables++
		}
	}
	if knownBusinessTables == 0 {
		return fmt.Errorf("%w: SubFlow domain collections are missing", ErrUnknownSchema)
	}
	return nil
}
