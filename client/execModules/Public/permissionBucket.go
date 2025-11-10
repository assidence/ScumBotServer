package Public

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// 全局信道：外部命令模块把 map[string]map[string]interface{} 发到这里
//var CommandConfigChan = make(chan map[string]map[string]interface{})

// CommandConfig 强类型表示单条命令配置（从 interface{} map 转换而来）
type CommandConfig struct {
	// 我把你的 ini 字段名映射到这里
	CoolDown   time.Duration // seconds
	DailyLimit int
	TotalLimit int
	// 其它字段按需扩展（PrefixRequire 等）
	PrefixRequire string

	// (Command字符串等可以忽略)
}

// Bucket 表示一个玩家针对某命令的使用状态
type Bucket struct {
	Tokens     int       // 可用于令牌模型，如果你用冷却而非计数可省略
	LastUsed   time.Time // 最近一次使用时间（用于 CoolDown 逻辑）
	DailyCount int
	TotalCount int
	mu         sync.Mutex
}

// Manager 负责合并配置、维护 buckets，并持久化到 sqlite
type Manager struct {
	mu                sync.RWMutex
	configs           map[string]*CommandConfig     // commandName -> config
	buckets           map[string]map[string]*Bucket // playerID -> commandName -> Bucket
	db                *sql.DB
	quit              chan struct{}
	wg                sync.WaitGroup
	CommandConfigChan chan map[string]map[string]interface{}
	//TitleManager      *Prefix.TitleManager
}

// NewManager 创建并初始化 sqlite 表
func NewManager(sqlitePath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return nil, err
	}
	// usage 表保存每个玩家每条命令的状态
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS usage (
		player_id TEXT NOT NULL,
		command TEXT NOT NULL,
		last_used INTEGER,
		daily_count INTEGER,
		total_count INTEGER,
		PRIMARY KEY (player_id, command)
	);`)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		configs:           make(map[string]*CommandConfig),
		buckets:           make(map[string]map[string]*Bucket),
		db:                db,
		quit:              make(chan struct{}),
		CommandConfigChan: make(chan map[string]map[string]interface{}),
	}
	// 在创建时从 DB 载入现有 usage 到内存（可选）
	if err := m.loadAllBucketsFromDB(); err != nil {
		return nil, err
	}
	// 启动监听信道和每日重置/持久化协程
	m.wg.Add(1)
	go m.signalListener()
	// 启动每日凌晨重置 daily count
	m.wg.Add(1)
	go m.dailyResetLoop()
	return m, nil
}

// Close 停止后台并关闭 db
func (m *Manager) Close() error {
	close(m.quit)
	m.wg.Wait()
	return m.db.Close()
}

// signalListener 从全局信道读取配置并合并到 m.configs
func (m *Manager) signalListener() {
	defer m.wg.Done()
	for {
		//fmt.Println("[PmBucket] signalListener Running")
		select {
		case newCfg, ok := <-m.CommandConfigChan:
			//fmt.Println("CommandConfigChan Reviced Data!")
			if !ok {
				return
			}
			// newCfg: map[commandName]map[string]interface{}
			m.mergeConfig(newCfg)
		case <-m.quit:
			//fmt.Println("[PmBucket] signalListener Quit")
			return
		}
	}
}

// mergeConfig 把外部上报的 map 合并进 manager.configs
func (m *Manager) mergeConfig(in map[string]map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for cmd, kv := range in {
		if cmd == "DEFAULT" {
			continue
		}
		// 建立或更新 CommandConfig，按需读取字段（类型断言）
		cfg := &CommandConfig{}
		// CoolDown (秒)
		if v, ok := kv["CoolDown"]; ok {
			switch t := v.(type) {
			case int:
				cfg.CoolDown = time.Duration(t) * time.Second
			case int64:
				cfg.CoolDown = time.Duration(t) * time.Second
			case float64:
				cfg.CoolDown = time.Duration(int(t)) * time.Second
			case string:
				// 尝试 parse int
				if parsed, err := strconv.Atoi(t); err == nil {
					cfg.CoolDown = time.Duration(parsed) * time.Second
				}
			}
		}
		// DailyLimit
		if v, ok := kv["DailyLimit"]; ok {
			switch t := v.(type) {
			case int:
				cfg.DailyLimit = t
			case int64:
				cfg.DailyLimit = int(t)
			case float64:
				cfg.DailyLimit = int(t)
			case string:
				if parsed, err := strconv.Atoi(t); err == nil {
					cfg.DailyLimit = parsed
				}
			}
		}
		// TotalLimit
		if v, ok := kv["TotalLimit"]; ok {
			switch t := v.(type) {
			case int:
				cfg.TotalLimit = t
			case int64:
				cfg.TotalLimit = int(t)
			case float64:
				cfg.TotalLimit = int(t)
			case string:
				if parsed, err := strconv.Atoi(t); err == nil {
					cfg.TotalLimit = parsed
				}
			}
		}
		// PrefixRequire
		if v, ok := kv["PrefixRequire"]; ok {
			switch t := v.(type) {
			case string:
				cfg.PrefixRequire = t
			}
		}
		// 如果某些字段没提供，保留之前的（如果已存在）
		if old, exists := m.configs[cmd]; exists {
			// only replace fields that are non-zero in cfg
			if cfg.CoolDown == 0 {
				cfg.CoolDown = old.CoolDown
			}
			if cfg.DailyLimit == 0 {
				cfg.DailyLimit = old.DailyLimit
			}
			if cfg.TotalLimit == 0 {
				cfg.TotalLimit = old.TotalLimit
			}
			// PrefixRequire: false default might be intentional; override only if provided:
			// we choose to override if incoming map has the key
			if _, provided := kv["PrefixRequire"]; !provided {
				cfg.PrefixRequire = old.PrefixRequire
			}
		}
		m.configs[cmd] = cfg
		fmt.Printf("[PmBucket] merged config for %s -> %+v\n", cmd, cfg)
	}
}

// getOrCreateBucket 获取内存中的 bucket（若不存在则尝试从 DB 载入或创建）
func (m *Manager) getOrCreateBucket(playerID, command string) *Bucket {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.buckets[playerID]; !ok {
		m.buckets[playerID] = make(map[string]*Bucket)
	}
	if b, ok := m.buckets[playerID][command]; ok {
		return b
	}
	// 试从 DB 载入
	var lastUsedUnix int64
	var dailyCount, totalCount int
	err := m.db.QueryRow("SELECT last_used, daily_count, total_count FROM usage WHERE player_id=? AND command=?", playerID, command).
		Scan(&lastUsedUnix, &dailyCount, &totalCount)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("DB load error:", err)
	}
	b := &Bucket{}
	if err == nil {
		b.LastUsed = time.Unix(lastUsedUnix, 0)
		b.DailyCount = dailyCount
		b.TotalCount = totalCount
	} else {
		// new bucket
		b.LastUsed = time.Time{} // zero -> allow immediate use unless CoolDown prohibits
		b.DailyCount = 0
		b.TotalCount = 0
	}
	m.buckets[playerID][command] = b
	return b
}

// saveBucket persist single bucket to DB (last_used as unix seconds)
func (m *Manager) saveBucket(playerID, command string, b *Bucket) {
	_, err := m.db.Exec(`INSERT OR REPLACE INTO usage (player_id, command, last_used, daily_count, total_count) VALUES (?, ?, ?, ?, ?)`,
		playerID, command, b.LastUsed.Unix(), b.DailyCount, b.TotalCount)
	if err != nil {
		fmt.Println("DB save error:", err)
	}
}

// loadAllBucketsFromDB 将 DB 中现有 usage 读到内存（启动时调用）
func (m *Manager) loadAllBucketsFromDB() error {
	rows, err := m.db.Query("SELECT player_id, command, last_used, daily_count, total_count FROM usage")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var playerID, cmd string
		var lastUsedUnix int64
		var dailyCount, totalCount int
		if err := rows.Scan(&playerID, &cmd, &lastUsedUnix, &dailyCount, &totalCount); err != nil {
			return err
		}
		if _, ok := m.buckets[playerID]; !ok {
			m.buckets[playerID] = make(map[string]*Bucket)
		}
		m.buckets[playerID][cmd] = &Bucket{
			LastUsed:   time.Unix(lastUsedUnix, 0),
			DailyCount: dailyCount,
			TotalCount: totalCount,
		}
	}
	return nil
}

// CanExecute 检查玩家是否可执行某命令（基于 CoolDown / DailyLimit / TotalLimit）
func (m *Manager) CanExecute(playerID, command string) (bool, string) {
	//TitleManager := Public.TitleManager
	m.mu.RLock()
	//fmt.Println("PermissionConfigs:")
	//fmt.Println(m.configs[command])
	cfg, exists := m.configs[command]
	m.mu.RUnlock()
	if !exists {
		return false, "[Permission] 未知命令"
	}
	b := m.getOrCreateBucket(playerID, command)
	b.mu.Lock()
	defer b.mu.Unlock()
	// cooldown
	if cfg.CoolDown > 0 && !b.LastUsed.IsZero() {
		remaining := cfg.CoolDown - time.Since(b.LastUsed)
		if remaining > 0 {
			return false, fmt.Sprintf("冷却中，剩余 %d 秒", int(remaining.Seconds()))
		}
	}
	// daily limit
	if cfg.DailyLimit > 0 && b.DailyCount >= cfg.DailyLimit {
		return false, fmt.Sprintf("今日已达上限 %d/%d", b.DailyCount, cfg.DailyLimit)
	}
	// total limit
	if cfg.TotalLimit >= 0 && b.TotalCount >= cfg.TotalLimit {
		return false, fmt.Sprintf("总次数已达上限 %d/%d", b.TotalCount, cfg.TotalLimit)
	}
	// Prefix limit
	if cfg.PrefixRequire != "" {
		ok, _ := TitleInterface.PrefixHasTitle(playerID, cfg.PrefixRequire)
		if !ok {
			return false, fmt.Sprintf("执行此命令需要称号【%s】", cfg.PrefixRequire)
		}
	}
	//b.mu.Unlock()
	m.Consume(playerID, command)
	return true, "允许执行"
}

// Consume 在允许执行后调用：记录使用并保存
func (m *Manager) Consume(playerID, command string) {
	b := m.getOrCreateBucket(playerID, command)
	//b.mu.Lock()
	//defer b.mu.Unlock()
	b.LastUsed = time.Now()
	b.DailyCount++
	b.TotalCount++
	// persist
	m.saveBucket(playerID, command, b)
}

// dailyResetLoop 每日午夜重置 daily_count
func (m *Manager) dailyResetLoop() {
	defer m.wg.Done()
	for {
		now := time.Now()
		// 下一次午夜
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		select {
		case <-time.After(time.Until(next)):
			// reset
			m.mu.Lock()
			for _, cmds := range m.buckets {
				for _, b := range cmds {
					b.mu.Lock()
					b.DailyCount = 0
					b.mu.Unlock()
				}
			}
			_, err := m.db.Exec("UPDATE usage SET daily_count = 0")
			if err != nil {
				fmt.Println("DB daily reset error:", err)
			}
			m.mu.Unlock()
		case <-m.quit:
			return
		}
	}
}
