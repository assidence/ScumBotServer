package CheckIn

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var dbPath = "./db/checkin.db"

// InitDB 打开数据库并创建表
func InitDB() error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// SQLite 并发限制：单连接
	db.SetMaxOpenConns(1)

	schema := `
CREATE TABLE IF NOT EXISTS checkin_records (
    steam_id TEXT PRIMARY KEY,
    last_date TEXT,
    streak_days INTEGER DEFAULT 0,
    total_days INTEGER DEFAULT 0
);
`
	if _, err = db.Exec(schema); err != nil {
		return fmt.Errorf("create table failed: %w", err)
	}

	if abs, e := filepath.Abs(dbPath); e == nil {
		fmt.Printf("[CheckIn-Module] DB initialized: %s\n", abs)
	} else {
		fmt.Printf("[CheckIn-Module] DB initialized (path: %s)\n", dbPath)
	}

	return nil
}

func CloseDB() error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// getRecord: 返回 (lastDate, streak, total, err)
// 若不存在，err == sql.ErrNoRows
func getRecord(steamID string) (string, int, int, error) {
	row := db.QueryRow("SELECT last_date, streak_days, total_days FROM checkin_records WHERE steam_id = ?", steamID)
	var last string
	var streak, total int
	err := row.Scan(&last, &streak, &total)
	return last, streak, total, err
}

func insertRecord(steamID, date string) error {
	_, err := db.Exec("INSERT INTO checkin_records (steam_id, last_date, streak_days, total_days) VALUES (?, ?, 1, 1)", steamID, date)
	return err
}

func updateRecord(steamID, date string, streak, total int) error {
	_, err := db.Exec("UPDATE checkin_records SET last_date=?, streak_days=?, total_days=? WHERE steam_id=?", date, streak, total, steamID)
	return err
}

func todayDate() string {
	return time.Now().Format("2006-01-02")
}
