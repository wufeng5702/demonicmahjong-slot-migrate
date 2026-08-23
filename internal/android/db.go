// Package android provides read access to the on-device SQLite save database
// and wraps adb for pulling the file from a connected device.
package android

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

// SlotRow mirrors one record of the `slot` table.
// SourceDB is the local file path the row was read from, so the UI can
// disambiguate rows coming from multiple game.db files (e.g. when the user
// has both a standard install and a TapTap install).
type SlotRow struct {
	ID          int64  `json:"id"`
	SlotIndex   int    `json:"slotIndex"`
	UserAccount string `json:"userAccount"`
	JSONString  string `json:"jsonString"`
	JSONSize    int    `json:"jsonSize"`
	JSONPreview string `json:"jsonPreview"`
	SourceDB    string `json:"sourceDb"`
}

// PreviewLimit is the maximum number of runes included in JSONPreview.
const PreviewLimit = 200

// ErrNotADB is returned when the file is not a valid SQLite database.
var ErrNotADB = errors.New("not a valid SQLite database or missing 'slot' table")

// ReadSlots opens the SQLite database in read-only mode and returns every row
// of the `slot` table ordered by (slotIndex, userAccount).
func ReadSlots(dbPath string) ([]SlotRow, error) {
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	// Forward slashes in the DSN are required even on Windows by modernc.org/sqlite.
	dsn := "file:" + filepath.ToSlash(abs) + "?mode=ro&_query_only=1&_pragma=busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	rows, err := db.Query(`SELECT id, slotIndex, userAccount, jsonString FROM slot ORDER BY slotIndex, userAccount`)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotADB, err)
	}
	defer rows.Close()

	var out []SlotRow
	for rows.Next() {
		var r SlotRow
		if err := rows.Scan(&r.ID, &r.SlotIndex, &r.UserAccount, &r.JSONString); err != nil {
			return nil, err
		}
		r.JSONSize = len(r.JSONString)
		r.JSONPreview = preview(r.JSONString)
		r.SourceDB = dbPath
		out = append(out, r)
	}
	return out, rows.Err()
}

func preview(s string) string {
	if utf8.RuneCountInString(s) <= PreviewLimit {
		return s
	}
	runes := make([]rune, 0, PreviewLimit)
	n := 0
	for _, r := range s {
		if n >= PreviewLimit {
			break
		}
		runes = append(runes, r)
		n++
	}
	return string(runes) + "..."
}
