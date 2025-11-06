package Achievement

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/Public"
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
	"strings"
	"sync"
)

type BehaviorRecorder struct {
	db *sql.DB
}

type Achievement struct {
	Name               string   // 成就名称，对应 ini 节名
	ActionType         string   // Kill / purchased / sold / Death / ...
	Target             string   // 行为目标，例如 Zombie、Gold Bar
	Require            int      // 达成数量，例如 100
	RewardCommand      string   // 原来的单行命令或文件路径
	RewardCommandLines []string // 解析后的命令列表（每行一条）
	RewardTitle        string   // 奖励称号，例如 Zombie Hunter
}

// 全局 TitleManager
//var GlobalTitleManager *TitleManager

// 互斥锁
var mu sync.RWMutex

// ------------------------ 基础工具函数 ------------------------

// 关闭数据库
func (r *BehaviorRecorder) AchievementClose() {
	if r.db != nil {
		r.db.Close()
	}
}

// 安全类型断言
func AchievementInterfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// 读取 ini 配置
func AchievementIniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/Achievement.ini")
	if err != nil {
		fmt.Println("[ERROR-Achievement]->Error:", err)
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
			fmt.Println("[ERROR-Achievement]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}

// 读取成就配置（RewardCommand 必须是文件路径）
func AchievementLoadAchievements(path string) ([]Achievement, error) {
	cfg, err := execModules.NewConfig(path)
	if err != nil {
		return nil, err
	}

	var achievements []Achievement
	for section, secMap := range cfg.Data {
		if section == "DEFAULT" {
			continue
		}

		// 安全解析 Require
		require := 0
		if secMap["Require"] != nil {
			require, _ = strconv.Atoi(AchievementInterfaceToString(secMap["Require"]))
		}

		// 安全解析 ActionType、Target、RewardTitle
		actionType := AchievementInterfaceToString(secMap["ActionType"])
		target := AchievementInterfaceToString(secMap["Target"])
		rewardTitle := AchievementInterfaceToString(secMap["RewardTitle"])

		// RewardCommand 必须是文件路径
		rewardFile := AchievementInterfaceToString(secMap["RewardCommand"])
		rewardLines := []string{}

		file, err := os.Open(rewardFile)
		if err != nil {
			fmt.Printf("[Achievement]->Error opening reward file '%s': %v\n", rewardFile, err)
		} else {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					rewardLines = append(rewardLines, line)
				}
			}
			file.Close()
			if err := scanner.Err(); err != nil {
				fmt.Println("[Achievement]->Error reading reward file:", err)
			}
		}

		achv := Achievement{
			Name:               section,
			ActionType:         actionType,
			Target:             target,
			Require:            require,
			RewardCommand:      rewardFile,
			RewardCommandLines: rewardLines,
			RewardTitle:        rewardTitle,
		}

		achievements = append(achievements, achv)
	}

	return achievements, nil
}

// ------------------------ 行为记录 ------------------------

// 初始化行为记录器
func AchievementNewBehaviorRecorder() *BehaviorRecorder {
	db, err := sql.Open("sqlite3", "./db/Achievement.db")
	if err != nil {
		panic(err)
	}
	createTable := `
	CREATE TABLE IF NOT EXISTS player_behaviors (
		steam_id TEXT NOT NULL,
		action_type TEXT NOT NULL,
		target TEXT,
		quantity INTEGER DEFAULT 0,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (steam_id, action_type, target)
	);
	CREATE TABLE IF NOT EXISTS player_achievements (
		steam_id TEXT NOT NULL,
		achievement_name TEXT NOT NULL,
		reached_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (steam_id, achievement_name)
	);
	`
	if _, err := db.Exec(createTable); err != nil {
		panic(err)
	}
	return &BehaviorRecorder{db: db}
}

// 记录行为
func (r *BehaviorRecorder) AchievementRecordBehaviorDetail(steamID, actionType, target string, quantity int) {
	if quantity <= 0 {
		quantity = 1
	}

	var exists int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM player_behaviors
		WHERE steam_id = ? AND action_type = ? AND target = ?`,
		steamID, actionType, target).Scan(&exists)
	if err != nil {
		fmt.Println("[ERROR-Achievement]->Check:", err)
		return
	}

	if exists > 0 {
		_, err = r.db.Exec(`
			UPDATE player_behaviors
			SET quantity = quantity + ?, updated_at = CURRENT_TIMESTAMP
			WHERE steam_id = ? AND action_type = ? AND target = ?`,
			quantity, steamID, actionType, target)
	} else {
		_, err = r.db.Exec(`
			INSERT INTO player_behaviors (steam_id, action_type, target, quantity)
			VALUES (?, ?, ?, ?)`,
			steamID, actionType, target, quantity)
	}
	if err != nil {
		fmt.Println("[ERROR-Achievement]->UPDATE/INSERT:", err)
	}
}

// 获取累计行为
func (r *BehaviorRecorder) AchievementGetBehaviorSum(steamID, actionType string) int {
	var total int
	err := r.db.QueryRow(`SELECT COALESCE(SUM(quantity),0) FROM player_behaviors WHERE steam_id = ? AND action_type = ?`, steamID, actionType).Scan(&total)
	if err != nil {
		return 0
	}
	return total
}

// ------------------------ 成就检查与触发 ------------------------

func (r *BehaviorRecorder) AchievementCheckAchievements(steamID string, achievements []Achievement, chatChan chan string) {
	for _, achv := range achievements {
		var qty int
		err := r.db.QueryRow(`SELECT quantity FROM player_behaviors WHERE steam_id = ? AND action_type = ? AND target = ?`,
			steamID, achv.ActionType, achv.Target).Scan(&qty)
		if err != nil {
			continue
		}

		if qty >= achv.Require {
			var exists int
			err = r.db.QueryRow(`SELECT COUNT(*) FROM player_achievements WHERE steam_id = ? AND achievement_name = ?`,
				steamID, achv.Name).Scan(&exists)
			if err != nil || exists > 0 {
				continue
			}

			r.AchievementUnlockAchievement(steamID, achv, chatChan)
		}
	}
}

// 执行奖励
func (r *BehaviorRecorder) AchievementUnlockAchievement(steamID string, achv Achievement, chatChan chan string) {
	_, err := r.db.Exec(`INSERT INTO player_achievements (steam_id, achievement_name) VALUES (?, ?)`, steamID, achv.Name)
	if err != nil {
		return
	}

	if achv.RewardTitle != "" {
		Done := make(chan struct{})
		Public.TitleInterface.CmdCh <- Public.TitleCommand{UserID: steamID, Command: "@给予称号", Title: achv.RewardTitle, Done: Done}
		<-Done
		Done = make(chan struct{})
		Public.TitleInterface.CmdCh <- Public.TitleCommand{UserID: steamID, Command: "@设置称号", Title: achv.RewardTitle, Done: Done}
		<-Done
	}

	for _, cmd := range achv.RewardCommandLines {
		cfglines := Public.CommandSelecterInterface.Selecter(steamID, cmd)
		for _, lines := range cfglines {
			chatChan <- lines
			fmt.Println("[Achievement-Module]:" + lines)
		}
	}

	fmt.Printf("[Achievement]->玩家 %s 解锁成就: %s\n", steamID, achv.Name)
}

// 分流行为记录
func AchievementRecordSelecter(steamID string, action string, target string, targetQuantity int, recorder *BehaviorRecorder, achv []Achievement, chatChan chan string) {
	switch action {
	case "Killer", "Died":
		recorder.AchievementRecordBehaviorDetail(steamID, action, target, targetQuantity)
	case "purchased":
		recorder.AchievementRecordBehaviorDetail(steamID, action, target, targetQuantity)
	case "sold":
		recorder.AchievementRecordBehaviorDetail(steamID, action, target, targetQuantity)
	case "equip":
		recorder.AchievementRecordBehaviorDetail(steamID, action, target, targetQuantity)
	}

	recorder.AchievementCheckAchievements(steamID, achv, chatChan)
}

// ------------------------ 命令处理 ------------------------

func AchievementCommandHandler(AchievementChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *Public.Manager, chatChan chan string, recorder *BehaviorRecorder, achv []Achievement) {
	for command := range AchievementChan {
		steamID := command["steamID"].(string)
		cmdName := command["command"].(string)
		commandArgs := strings.Split(command["commandArgs"].(string), "-")

		ok, msg := PMbucket.CanExecute(steamID, cmdName)
		if !ok {
			fmt.Println("[ERROR-Achievement]->Error:", msg)
			continue
		}

		amount, _ := strconv.Atoi(commandArgs[2])
		AchievementRecordSelecter(commandArgs[0], cmdName, commandArgs[1], amount, recorder, achv, chatChan)

		commandLines := cfg.Data[cmdName]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfglines := Public.CommandSelecterInterface.Selecter(commandArgs[0], cfgCommand)
			for _, lines := range cfglines {
				chatChan <- lines
				fmt.Println("[Achievement-Module]:" + lines)
			}
		}
	}
	defer PMbucket.Close()
}

// -------------------传输成就信息到公共接口--------------------
func PhaseAchievementDetailsToGlobalInterface(ArchList []Achievement) {
	for _, A := range ArchList {
		Public.AchievementInterface.AchievementReward[A.Name] = A.RewardCommandLines
	}
}

// ------------------------ 模块入口 ------------------------

func AchievementModule(regCommand *map[string][]string, AchievementChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {
	cfg := AchievementIniLoader()
	PmBucket := AchievementcreatePermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	AchievementCommandRegister(cfg, regCommand)

	recorder := AchievementNewBehaviorRecorder()
	achievements, _ := AchievementLoadAchievements("./ini/Achievement-gold.ini")
	PhaseAchievementDetailsToGlobalInterface(achievements)
	//Public.GlobalAchievements = &achievements

	go func() {
		AchievementCommandHandler(AchievementChan, cfg, PmBucket, chatChan, recorder, achievements)
		recorder.AchievementClose()
	}()

	close(initChan)
}

// ------------------------ 注册命令 ------------------------

func AchievementCommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string
	for section := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["Achievement"] = commandList
}

// ------------------------ 权限管理器 ------------------------

func AchievementcreatePermissionBucket() *Public.Manager {
	PmBucket, err := Public.NewManager("./db/Achievement-Perm.db")
	if err != nil {
		panic(err)
	}
	return PmBucket
}
