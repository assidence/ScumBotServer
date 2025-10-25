package Archievement

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/CommandSelecter"
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/Prefix"
	"ScumBotServer/client/execModules/permissionBucket"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type BehaviorRecorder struct {
	db *sql.DB
}

// 初始化配置
func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/Archievement.ini")
	if err != nil {
		fmt.Println("[ERROR-Archievement]->Error:", err)
		return &execModules.Config{}
	}
	var commandList []string
	for section, secMap := range cfg.Data {
		if section == "DEFAULT" {
			continue
		}
		commandFilePart := secMap["Command"].(string)
		commandList, err = execModules.CommandFileReadLines(commandFilePart)
		if err != nil {
			fmt.Println("[ERROR-Archievement]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}

// 权限管理器
func createPermissionBucket() *permissionBucket.Manager {
	PmBucket, err := permissionBucket.NewManager("./db/Archievement-Perm.db")
	if err != nil {
		panic(err)
	}
	return PmBucket
}

// 初始化玩家行为记录器
func newBehaviorRecorder() *BehaviorRecorder {
	db, err := sql.Open("sqlite3", "./db/Archievement.db")
	if err != nil {
		panic(err)
	}
	createTable := `
	CREATE TABLE IF NOT EXISTS player_behaviors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		steam_id TEXT NOT NULL,
		action_type TEXT NOT NULL,
		target TEXT,
		quantity INTEGER DEFAULT 1,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTable); err != nil {
		panic(err)
	}
	return &BehaviorRecorder{db: db}
}

// ✅ 通用记录函数（支持购买、出售、击杀、死亡）
func (r *BehaviorRecorder) RecordBehaviorDetail(steamID, actionType, target string, quantity int) {
	if quantity <= 0 {
		quantity = 1
	}

	// 对于击杀/死亡类行为，我们尝试累加而不是重复插入
	if actionType == "Kill" || actionType == "Death" {
		var exists int
		err := r.db.QueryRow(`SELECT COUNT(*) FROM player_behaviors WHERE steam_id = ? AND action_type = ? AND target = ?`, steamID, actionType, target).Scan(&exists)
		if err == nil && exists > 0 {
			_, err = r.db.Exec(`UPDATE player_behaviors SET quantity = quantity + ? WHERE steam_id = ? AND action_type = ? AND target = ?`,
				quantity, steamID, actionType, target)
			if err != nil {
				fmt.Println("[ERROR-Archievement]->RecordBehaviorDetail(UPDATE):", err)
			}
			return
		}
	}

	// 其他情况直接插入
	_, err := r.db.Exec(`INSERT INTO player_behaviors (steam_id, action_type, target, quantity, timestamp)
		VALUES (?, ?, ?, ?, ?)`, steamID, actionType, target, quantity, time.Now())
	if err != nil {
		fmt.Println("[ERROR-Archievement]->RecordBehaviorDetail(INSERT):", err)
	}
}

// ✅ 查询玩家行为统计
func (r *BehaviorRecorder) GetBehaviorStats(steamID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT action_type, target, quantity, timestamp FROM player_behaviors WHERE steam_id = ? ORDER BY id DESC LIMIT 50`, steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var aType, target string
		var qty int
		var ts string
		rows.Scan(&aType, &target, &qty, &ts)
		stats = append(stats, map[string]interface{}{
			"Action":    aType,
			"Target":    target,
			"Quantity":  qty,
			"Timestamp": ts,
		})
	}
	return stats, nil
}

// 关闭数据库
func (r *BehaviorRecorder) Close() {
	if r.db != nil {
		r.db.Close()
	}
}

// 注册命令
func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string
	for section := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["Archievement"] = commandList
}

// 分流行为记录器
func recordSelecter(steamID string, action string, targetQuantity string, recorder *BehaviorRecorder) {
	switch action {
	case "Kill":
	case "Death":
	case "purchased":

		// 正则匹配 "名字 (x数量)"
		re := regexp.MustCompile(`^(.*?)\s*\(\s*x(\d+)\s*\)$`)
		match := re.FindStringSubmatch(targetQuantity)
		if len(match) < 2 {
			fmt.Errorf("[Archievement-Error]no quantity found")
		}
		// 规范格式
		name := strings.TrimSpace(match[1])
		qty, _ := strconv.Atoi(match[2])
		// 玩家购买物品
		recorder.RecordBehaviorDetail(steamID, action, name, qty)
	case "sold":
		// 匹配括号前的名字
		re := regexp.MustCompile(`^(.*?)\s*\(`)
		match := re.FindStringSubmatch(targetQuantity)
		if len(match) < 2 {
			// 没有括号，直接返回原字符串去空格
			fmt.Errorf("[Archievement-Error]no sold Item found")
		}
		name := strings.TrimSpace(match[1])
		qty := 1
		recorder.RecordBehaviorDetail(steamID, action, name, qty)
	case "deposit":
	case "destroyed_card":
	}

	// 玩家出售物品
	//recorder.RecordBehaviorDetail(steamID, "Sell", "Ammo 7.62mm", 30)

	// 玩家击杀
	//recorder.RecordBehaviorDetail(steamID, "Kill", "Zombie", 1)

	// 玩家死亡
	//recorder.RecordBehaviorDetail(steamID, "Death", "Explosion", 1)

	//行为记录查询
	records, _ := recorder.GetBehaviorStats("76561198012345678")
	for _, rec := range records {
		fmt.Printf("[%s]%s %s ×%d\n", rec["Timestamp"], rec["Action"], rec["Target"], rec["Quantity"])
	}
}

// 主命令处理
func CommandHandler(ArchievementChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string, lw *LogWacher.LogWatcher, recorder *BehaviorRecorder) {
	var commandLines []string
	for command := range ArchievementChan {
		steamID := command["steamID"].(string)
		cmdName := command["command"].(string)
		commandArgs := strings.Split(command["commandArgs"].(string), "-")
		//nick := command["nickName"].(string)

		ok, msg := PMbucket.CanExecute(steamID, cmdName)
		if !ok {
			fmt.Println("[ERROR-Archievement]->Error:", msg)
			continue
		}

		recordSelecter(commandArgs[0], cmdName, commandArgs[1], recorder)

		commandLines = cfg.Data[cmdName]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfglines := CommandSelecter.Selecter(steamID, cfgCommand, lw)
			for _, lines := range cfglines {
				chatChan <- lines
				fmt.Println("[Archievement-Module]:" + lines)
			}
		}
	}
	defer PMbucket.Close()
}

// 模块入口
func Archievement(regCommand *map[string][]string, ArchievementChan chan map[string]interface{}, chatChan chan string, lw *LogWacher.LogWatcher, TitleManager *Prefix.TitleManager, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	PmBucket.TitleManager = TitleManager
	CommandRegister(cfg, regCommand)

	recorder := newBehaviorRecorder()

	go func() {
		CommandHandler(ArchievementChan, cfg, PmBucket, chatChan, lw, recorder)
		recorder.Close()
	}()
	close(initChan)
}
