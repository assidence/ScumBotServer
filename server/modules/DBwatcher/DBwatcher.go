package DBwatcher

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DBWatcher struct {
	DBPath         string
	CheckInterval  time.Duration
	OnUpdate       func(db *sql.DB)
	db             *sql.DB
	lastModTime    time.Time
	lastWalModTime time.Time
	isWALMode      bool
	walFile        string
	stopChan       chan struct{}
}

// New 创建一个新的数据库监控器
func New(dbPath string, interval time.Duration, onUpdate func(db *sql.DB)) (*DBWatcher, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", dbPath))
	if err != nil {
		return nil, err
	}

	// 检查是否是 WAL 模式
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		return nil, err
	}

	isWAL := (journalMode == "wal")
	watcher := &DBWatcher{
		DBPath:        dbPath,
		CheckInterval: interval,
		OnUpdate:      onUpdate,
		db:            db,
		isWALMode:     isWAL,
		walFile:       dbPath + "-wal",
		stopChan:      make(chan struct{}),
	}

	log.Printf("[dbwatcher] 监控启动: %s (模式: %s)\n", dbPath, journalMode)
	return watcher, nil
}

// Start 开始监控
func (w *DBWatcher) Start() {
	for {
		select {
		case <-w.stopChan:
			log.Println("[dbwatcher] 已停止监控")
			return
		default:
			w.checkChanges()
			time.Sleep(w.CheckInterval)
		}
	}
}

// Stop 停止监控
func (w *DBWatcher) Stop() {
	close(w.stopChan)
	if w.db != nil {
		w.db.Close()
	}
}

func (w *DBWatcher) checkChanges() {
	changed := false

	// 检查主数据库文件
	if info, err := os.Stat(w.DBPath); err == nil {
		if info.ModTime() != w.lastModTime {
			w.lastModTime = info.ModTime()
			changed = true
		}
	}

	// 如果是 WAL 模式，再检查 wal 文件
	if w.isWALMode {
		if info, err := os.Stat(w.walFile); err == nil {
			if info.ModTime() != w.lastWalModTime {
				w.lastWalModTime = info.ModTime()
				changed = true
			}
		}
	}

	if changed {
		log.Println("[dbwatcher] 检测到数据库变更，准备触发回调...")
		time.Sleep(1 * time.Second) // 给主程序写入一点时间完成 checkpoint
		w.OnUpdate(w.db)
	}
}

// PrintTables 示例函数：打印数据库中所有表名
func PrintTables(db *sql.DB) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;`)
	if err != nil {
		log.Println("读取表失败:", err)
		return
	}
	defer rows.Close()

	var name string
	for rows.Next() {
		rows.Scan(&name)
		fmt.Println("📂 表名:", name)
	}
}
