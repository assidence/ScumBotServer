package CheckInRewardManager

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/Public"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Debug 控制（手动开启即可，不依赖 ini）
var DebugEnabled bool = false

func debugLog(msg string) {
	if DebugEnabled {
		fmt.Println("[DEBUG-CheckInRewardManager] " + msg)
	}
}

// 玩家信息结构体
type PlayerInfo struct {
	SteamID       string
	TotalLogin    int
	LastTier      string
	LastLoginDate string
}

// INI 加载
func iniLoader() *execModules.Config {
	debugLog("加载 CheckInRewardManager.ini ...")

	cfg, err := execModules.NewConfig("./ini/CheckInRewardManager.ini")
	if err != nil {
		fmt.Println("[ERROR-CheckInRewardManager]->Error:", err)
		return &execModules.Config{}
	}

	for section, secMap := range cfg.Data {
		if section == "DEFAULT" {
			continue
		}

		commandFilePart := secMap["Command"].(string)
		cmdList, err := execModules.CommandFileReadLines(commandFilePart)
		if err != nil {
			fmt.Println("[ERROR-CheckInRewardManager]->Error reading command file:", err)
			continue
		}
		cfg.Data[section]["Command"] = cmdList

		debugLog("加载命令组: " + section)
	}

	return cfg
}

func loadRewardLevels() *execModules.Config {
	debugLog("加载 CheckInRewardLevels.ini ...")

	cfg, err := execModules.NewConfig("./ini/CheckInRewardLevels.ini")
	if err != nil {
		fmt.Println("[ERROR-CheckInReward]->Error loading CheckInRewardLevels.ini:", err)
		return &execModules.Config{Data: make(map[string]map[string]interface{})}
	}

	for tier, secMap := range cfg.Data {
		cmdPath := secMap["Command"].(string)
		cmdList, err := execModules.CommandFileReadLines(cmdPath)
		if err != nil {
			fmt.Println("[ERROR-CheckInReward]->Error reading command file:", err)
			continue
		}
		cfg.SetValue(tier, "Command", cmdList)

		debugLog("Reward Tier 加载成功: " + tier)
	}

	return cfg
}

// 数据库操作
func createDB() *sql.DB {
	dbDir := "./db"
	dbPath := filepath.Join(dbDir, "CheckInReward.db")

	debugLog("初始化数据库: " + dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(fmt.Sprintf("无法打开数据库: %v", err))
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS players_tier (
		steam_id TEXT PRIMARY KEY,
		last_login TEXT,
		total_login_days INTEGER,
		last_reward_tier TEXT
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		panic(fmt.Sprintf("无法创建表: %v", err))
	}

	debugLog("数据库结构检查完毕")
	return db
}

func getOrCreatePlayer(db *sql.DB, steamID string) (*PlayerInfo, error) {
	debugLog("读取玩家数据: " + steamID)

	var p PlayerInfo
	row := db.QueryRow("SELECT total_login_days, last_reward_tier, last_login FROM players_tier WHERE steam_id=?", steamID)
	err := row.Scan(&p.TotalLogin, &p.LastTier, &p.LastLoginDate)
	p.SteamID = steamID

	if err == sql.ErrNoRows {
		today := time.Now().Format("2006-01-02")
		_, err := db.Exec(
			"INSERT INTO players_tier(steam_id, last_login, total_login_days, last_reward_tier) VALUES (?, ?, ?, ?)",
			steamID, today, 1, "",
		)
		if err != nil {
			return nil, err
		}

		debugLog("玩家首次记录创建: " + steamID)
		p.TotalLogin = 1
		p.LastTier = ""
		p.LastLoginDate = today
		return &p, nil
	}

	if err != nil {
		return nil, err
	}

	debugLog(fmt.Sprintf("玩家记录读取成功: %s, 连续 %d 天", steamID, p.TotalLogin))
	return &p, nil
}

func updateDailyLogin(db *sql.DB, p *PlayerInfo) error {
	today := time.Now().Format("2006-01-02")
	if p.LastLoginDate != today {
		p.TotalLogin++
		p.LastLoginDate = today
		debugLog(fmt.Sprintf("玩家 %s 今日首次登录，累计: %d", p.SteamID, p.TotalLogin))

		_, err := db.Exec(
			"UPDATE players_tier SET total_login_days=?, last_login=? WHERE steam_id=?",
			p.TotalLogin, today, p.SteamID,
		)
		return err
	}

	debugLog("玩家今日已登录: " + p.SteamID)
	return nil
}

func updatePlayerTier(db *sql.DB, steamID string, tier string) error {
	debugLog(fmt.Sprintf("更新玩家 %s Tier 为 %s", steamID, tier))

	_, err := db.Exec(
		"UPDATE players_tier SET last_reward_tier=? WHERE steam_id=?",
		tier, steamID,
	)
	return err
}

// PMbucket & 命令注册
func createPermissionBucket() *Public.Manager {
	debugLog("初始化 PermissionBucket ...")

	PmBucket, err := Public.NewManager("./db/CheckInRewardManager.db")
	if err != nil {
		panic(err)
	}
	return PmBucket
}

func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string
	for section := range cfg.Data {
		commandList = append(commandList, section)
	}

	debugLog("注册命令组: CheckInRewardManager")

	(*regCommand)["CheckInRewardManager"] = commandList
}

// CommandHandler
func CommandHandler(cmdChan chan map[string]interface{}, levelCfg *execModules.Config, PMbucket *Public.Manager, db *sql.DB, chatChan chan string) {

	for Command := range cmdChan {
		steamID := Command["steamID"].(string)
		nickName := Command["nickName"].(string)
		command := Command["command"].(string)

		debugLog(fmt.Sprintf("接收到玩家命令 [%s] from %s", command, steamID))

		// 如果玩家调用 @签到，直接返回累计签到天数
		if command == "@签到" {
			player, err := getOrCreatePlayer(db, steamID)
			if err != nil {
				fmt.Println("[ERROR-CheckInReward]->DB Error:", err)
				continue
			}
			chatChan <- fmt.Sprintf("玩家: %s 已签到了 %d 天", nickName, player.TotalLogin)
			debugLog(fmt.Sprintf("玩家 %s 查询签到天数: %d", steamID, player.TotalLogin))
			continue
		}

		// 权限验证
		ok, msg := PMbucket.CanExecute(steamID, command)
		if !ok {
			if command == "checkIn" {
				debugLog("[CheckInReward]玩家重复尝试签到: " + steamID)
			} else {
				chatChan <- fmt.Sprintf("玩家%s：%s", nickName, msg)
			}
			continue
		}

		// 获取或创建玩家信息
		player, err := getOrCreatePlayer(db, steamID)
		if err != nil {
			fmt.Println("[ERROR-CheckInReward]->DB Error:", err)
			continue
		}

		// 检查是否首次登录
		if err := updateDailyLogin(db, player); err != nil {
			fmt.Println("[ERROR-CheckInReward]->DB Update DailyLogin Error:", err)
		}

		// 查找符合的最高 Tier
		var targetTier string
		highestDays := int64(-1)
		for tier := range levelCfg.Data {
			requiredDays := levelCfg.GetInt(tier, "Days")
			if int64(player.TotalLogin) >= requiredDays && player.LastTier != tier && requiredDays > highestDays {
				targetTier = tier
				highestDays = requiredDays
			}
		}

		if targetTier == "" {
			debugLog("玩家没有达到新的奖励 Tier")
			continue
		}

		debugLog(fmt.Sprintf("玩家 %s 达成 Tier: %s", steamID, targetTier))

		// 执行奖励命令
		cmdList, ok := levelCfg.GetValue(targetTier, "Command").([]string)
		if !ok {
			fmt.Println("[ERROR-CheckInReward]->Command invalid type:", targetTier)
			continue
		}

		for _, line := range cmdList {
			lines := Public.CommandSelecterInterface.Selecter(steamID, line)
			for _, l := range lines {
				chatChan <- l
				fmt.Println("[CheckInRewardManager]:" + l)
			}
		}

		// 扣除权限
		PMbucket.Consume(steamID, command)

		// 更新玩家 Tier
		if err := updatePlayerTier(db, steamID, targetTier); err != nil {
			fmt.Println("[ERROR-CheckInReward]->DB Update Tier Error:", err)
		}
	}
}

// 模块启动
func CheckInRewardManager(regCommand *map[string][]string, cmdChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {

	debugLog("初始化 CheckInRewardManager 模块")

	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data

	CommandRegister(cfg, regCommand)
	levelCfg := loadRewardLevels()

	db := createDB()
	defer db.Close()

	go CommandHandler(cmdChan, levelCfg, PmBucket, db, chatChan)

	debugLog("CheckInRewardManager 初始化完毕")

	close(initChan)
}
