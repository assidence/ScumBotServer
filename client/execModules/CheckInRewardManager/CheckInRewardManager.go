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

	cfg, err := execModules.NewConfig("./ini/CheckInRewardManager/CheckInRewardManager.ini")
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

	cfg, err := execModules.NewConfig("./ini/CheckInRewardManager/CheckInRewardLevels.ini")
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
		// 首次创建时默认 Tier1
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

	debugLog(fmt.Sprintf("玩家记录读取成功: %s, 连续 %d 天, 当前Tier: %s", steamID, p.TotalLogin, p.LastTier))
	return &p, nil
}

// 更新玩家签到与当前 Tier
func updateDailyLogin(db *sql.DB, p *PlayerInfo, levelCfg *execModules.Config) error {
	today := time.Now().Format("2006-01-02")
	firstLoginToday := p.LastLoginDate != today

	if firstLoginToday {
		p.TotalLogin++
		p.LastLoginDate = today
		debugLog(fmt.Sprintf("玩家 %s 今日首次登录，累计: %d", p.SteamID, p.TotalLogin))
	}

	// 计算当前 Tier
	var currentTier string
	highestDays := int64(-1)
	for tier := range levelCfg.Data {
		requiredDays := levelCfg.GetInt(tier, "Days")
		if int64(p.TotalLogin) >= requiredDays && requiredDays > highestDays {
			currentTier = tier
			highestDays = requiredDays
		}
	}

	// 更新数据库
	_, err := db.Exec(
		"UPDATE players_tier SET total_login_days=?, last_login=?, last_reward_tier=? WHERE steam_id=?",
		p.TotalLogin, today, currentTier, p.SteamID,
	)
	if err != nil {
		return err
	}

	p.LastTier = currentTier

	if firstLoginToday {
		debugLog(fmt.Sprintf("玩家 %s 当前 Tier 更新为 %s", p.SteamID, currentTier))
	}

	return nil
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

// 处理命令
func CommandHandler(cmdChan chan map[string]interface{}, levelCfg *execModules.Config, PMbucket *Public.Manager, db *sql.DB, chatChan chan string) {

	for Command := range cmdChan {
		steamID := Command["steamID"].(string)
		nickName := Public.LogWatcherInterface.Players[steamID].Name
		command := Command["command"].(string)

		debugLog(fmt.Sprintf("接收到玩家命令 [%s] from %s", command, steamID))

		// 获取或创建玩家信息
		player, err := getOrCreatePlayer(db, steamID)
		if err != nil {
			fmt.Println("[ERROR-CheckInReward]->DB Error:", err)
			continue
		}

		// 权限验证
		ok, msg := PMbucket.CanExecute(steamID, command)
		if !ok {
			debugLog("[CheckInReward]玩家 " + steamID + " 执行命令：" + command + " 失败")
			if command == "checkIn" {
				debugLog("[CheckInReward]玩家重复尝试签到: " + steamID)
			} else {
				chatChan <- fmt.Sprintf("玩家%s：%s", nickName, msg)
			}
			continue
		}

		switch command {

		case "checkIn":
			// 自动签到：更新每日登录 + 刷新已解锁 Tier
			if err := updateDailyLogin(db, player, levelCfg); err != nil {
				fmt.Println("[ERROR-CheckInReward]->DB UpdateDailyLogin Error:", err)
				continue
			}
			PMbucket.Consume(steamID, command)
			debugLog(fmt.Sprintf("玩家 %s 自动签到完成，累计 %d 天，当前解锁 Tier: %s", steamID, player.TotalLogin, player.LastTier))

		case "@签到":
			// 玩家主动查询签到信息
			tierDisc := ""
			if player.LastTier != "" {
				if v, ok := levelCfg.GetValue(player.LastTier, "Disc").(string); ok {
					tierDisc = v
				}
			}

			if tierDisc != "" {
				chatChan <- fmt.Sprintf("玩家: %s 已签到 %d 天\n%s", nickName, player.TotalLogin, tierDisc)
			} else {
				chatChan <- fmt.Sprintf("玩家: %s 已签到 %d 天", nickName, player.TotalLogin)
			}
			PMbucket.Consume(steamID, command)
			debugLog(fmt.Sprintf("玩家 %s 查询签到天数: %d, 描述: %s", steamID, player.TotalLogin, tierDisc))

		case "@老手礼包":
			// 玩家主动领取奖励
			targetTier := player.LastTier
			if targetTier == "" {
				chatChan <- fmt.Sprintf("玩家 %s 当前没有可领取的奖励", nickName)
				continue
			}

			cmdList, ok := levelCfg.GetValue(targetTier, "Command").([]string)
			if ok {
				for _, line := range cmdList {
					lines := Public.CommandSelecterInterface.Selecter(steamID, line)
					for _, l := range lines {
						chatChan <- l
						//fmt.Println("[CheckInRewardManager]:" + l)
					}
				}
			}

			PMbucket.Consume(steamID, command)
			debugLog(fmt.Sprintf("玩家 %s 使用老手礼包，执行 Tier %s 奖励命令", steamID, targetTier))

		default:
			// 其他未知命令
			debugLog(fmt.Sprintf("[ERROR-CheckInReward] 未知命令: %s", command))
		}
	}
}

// 模块启动
func CheckInRewardManager(regCommand *map[string][]string, CheckInRewardManagerChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {

	debugLog("初始化 CheckInRewardManager 模块")

	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data

	CommandRegister(cfg, regCommand)
	levelCfg := loadRewardLevels()

	db := createDB()
	//defer db.Close()

	go CommandHandler(CheckInRewardManagerChan, levelCfg, PmBucket, db, chatChan)

	debugLog("CheckInRewardManager 初始化完毕")

	close(initChan)
}
