package Prefix

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/Public"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"
	"sync"
)

// PlayerTitle 玩家称号记录
type PlayerTitle struct {
	UserID string
	Title  string
	Active bool
}

// TitleManager 核心管理器
type TitleManager struct {
	db     *sql.DB
	CmdCh  chan TitleCommand
	mu     sync.Mutex
	wg     sync.WaitGroup
	closed bool
}

// PrefixNewTitleManager 创建并初始化模块
func PrefixNewTitleManager(dbPath string, chatChan chan string) (*TitleManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	manager := &TitleManager{
		db:    db,
		CmdCh: Public.TitleInterface.CmdCh,
	}

	if err := manager.PrefixInitDB(); err != nil {
		return nil, err
	}

	manager.wg.Add(1)
	go manager.PrefixListenCommands(chatChan)

	return manager, nil
}

// PrefixInitDB 初始化数据库
func (m *TitleManager) PrefixInitDB() error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS player_titles (
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		active INTEGER DEFAULT 0,
		PRIMARY KEY (user_id, title)
	);
	`
	_, err := m.db.Exec(sqlStmt)
	return err
}

// PrefixListenCommands 监听来自其他模块的指令
func (m *TitleManager) PrefixListenCommands(chatChan chan string) {
	defer m.wg.Done()
	for cmd := range m.CmdCh {
		switch cmd.Command {
		case CommandGrant:
			if err := m.PrefixGrantTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s授予称号失败: %v", Public.LogWatcherInterface.Players[cmd.UserID].Name, err)
				fmt.Println("[Error-Prefix] " + fmt.Sprintf("%s授予称号失败: %v", cmd.UserID, err))
			} else {
				chatChan <- fmt.Sprintf("%s获得称号 %s", Public.LogWatcherInterface.Players[cmd.UserID].Name, cmd.Title)
				fmt.Printf("[Prefix-Module]  玩家 %s 获得称号 %s\n", cmd.UserID, cmd.Title)
			}

		case CommandRemove:
			if err := m.PrefixRemoveTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s移除称号失败: %v", Public.LogWatcherInterface.Players[cmd.UserID].Name, err)
				fmt.Println("[Error-Prefix]" + fmt.Sprintf("[Error-Prefix] %s移除称号失败: %v", cmd.UserID, err))
			} else {
				chatChan <- fmt.Sprintf("玩家 %s 移除称号 %s", Public.LogWatcherInterface.Players[cmd.UserID].Name, cmd.Title)
				fmt.Printf("[Prefix-Module] 玩家 %s 移除称号 %s\n", cmd.UserID, cmd.Title)
			}

		case CommandSet:
			if err := m.PrefixSetActiveTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s设置当前称号失败: %v", Public.LogWatcherInterface.Players[cmd.UserID].Name, err)
				fmt.Printf("[Error-Prefix] 玩家 %s设置当前称号失败: %v\n", cmd.UserID, err)
			} else {
				p := Public.LogWatcherInterface.Players[cmd.UserID]
				p.Prefix = cmd.Title
				Public.LogWatcherInterface.Players[cmd.UserID] = p
				line := fmt.Sprintf("#SetFakeName %s -★%s★-%s", cmd.UserID, cmd.Title, p.Name)
				chatChan <- line
				chatChan <- fmt.Sprintf("%s当前称号设为 %s 可使用@隐藏称号 来取消", p.Name, cmd.Title)
				fmt.Println("[Prefix-Module]:" + line)
			}
		case CommandUnSet:
			p := Public.LogWatcherInterface.Players[cmd.UserID]
			p.Prefix = ""
			Public.LogWatcherInterface.Players[cmd.UserID] = p
			line := fmt.Sprintf("#SetFakeName %s %s", cmd.UserID, p.Name)
			chatChan <- line
			chatChan <- fmt.Sprintf("%s当前称号已取消展示", p.Name)
			fmt.Println("[Prefix-Module]:" + line)
		case CommandQuery:
			if Public.TitleInterface.OnlinePlayerPrefixList == nil {
				fmt.Println("[Error-Prefix] 玩家称号列表未初始化")
				break
			}
			titleList, err := m.PrefixGetTitlesByUserID(cmd.UserID)
			var line string
			if err != nil || len(titleList) == 0 {
				fmt.Printf("[Error-Prefix] 玩家 %s查询拥有的称号失败: %v\n", cmd.UserID, err)
				line = fmt.Sprintf("玩家: %s 还未拥有任何称号 称号会随着游戏行为解锁", Public.LogWatcherInterface.Players[cmd.UserID].Name)
			} else {
				Public.TitleInterface.OnlinePlayerPrefixList[cmd.UserID] = titleList
				line = fmt.Sprintf("玩家: %s 拥有的称号是: ", Public.LogWatcherInterface.Players[cmd.UserID].Name)
				for i, pid := range titleList {
					line += fmt.Sprintf(" [%d]-★%s★-  ", i+1, pid)
				}
				line += fmt.Sprintf("使用示例 @使用称号 1 可以设置-★%s★-作为称号", titleList[0])
			}
			chatChan <- line
		case CommandUse:
			p := Public.LogWatcherInterface.Players[cmd.UserID]
			if Public.TitleInterface.OnlinePlayerPrefixList == nil {
				fmt.Println("[Error-Prefix] 玩家称号列表未初始化")
				break
			}
			i, err := strconv.Atoi(cmd.Title)
			if err != nil {
				fmt.Printf("[Error-Prefix] 玩家 %s 设置称号命令执行失败: %v\n", cmd.UserID, err)
				break
			}

			var line string
			titles, ok := Public.TitleInterface.OnlinePlayerPrefixList[cmd.UserID]
			if !ok || len(titles) == 0 || i < 0 || i-1 >= len(titles) {
				line = fmt.Sprintf("玩家: %s 找不到对应序号的称号\n", p.Name)
				fmt.Printf("[Error-Prefix] 找不到对应玩家%s的称号记录：记录长度%d\n", cmd.UserID, len(titles))
				chatChan <- line
				break
			}

			cmd.Title = titles[i-1]
			if cmd.Title == "" {
				line = fmt.Sprintf("玩家: %s 找不到对应序号的称号\n", p.Name)
				chatChan <- line
				break
			}
			if err = m.PrefixSetActiveTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s设置当前称号失败: %v", Public.LogWatcherInterface.Players[cmd.UserID].Name, err)
				fmt.Printf("[Error-Prefix] 玩家 %s设置当前称号失败: %v\n", cmd.UserID, err)
				break
			} else {
				// 这里可以放你成功设置称号的逻辑
				p.Prefix = cmd.Title
				Public.LogWatcherInterface.Players[cmd.UserID] = p
				line = fmt.Sprintf("#SetFakeName %s -★%s★-%s", cmd.UserID, cmd.Title, p.Name)
				chatChan <- line
				chatChan <- fmt.Sprintf("%s当前称号设为 %s 可使用@隐藏称号 来取消", p.Name, cmd.Title)
				fmt.Println("[Prefix-Module]:" + line)
			}
		}
		close(cmd.Done)
	}
}

// PrefixGrantTitle 授予称号
func (m *TitleManager) PrefixGrantTitle(userID, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec(`
	INSERT OR IGNORE INTO player_titles (user_id, title, active)
	VALUES (?, ?, 0)
	`, userID, title)
	return err
}

// PrefixRemoveTitle 移除称号
func (m *TitleManager) PrefixRemoveTitle(userID, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec(`
	DELETE FROM player_titles WHERE user_id = ? AND title = ?
	`, userID, title)
	return err
}

// PrefixSetActiveTitle 设置当前使用称号
func (m *TitleManager) PrefixSetActiveTitle(userID, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE player_titles SET active = 0 WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	res, err := tx.Exec(`
	UPDATE player_titles SET active = 1 WHERE user_id = ? AND title = ?
	`, userID, title)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("玩家 %s 没有称号 %s", userID, title)
	}
	return tx.Commit()
}

// PrefixHasTitle 查询玩家是否拥有称号
func (m *TitleManager) PrefixHasTitle(userID, title string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int
	err := m.db.QueryRow(`
	SELECT COUNT(*) FROM player_titles WHERE user_id = ? AND title = ?
	`, userID, title).Scan(&count)
	return count > 0, err
}

// PrefixGetTitlesByUserID 查询指定用户的所有称号列表（含激活状态）
func (m *TitleManager) PrefixGetTitlesByUserID(userID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows, err := m.db.Query(`
		SELECT title, active FROM player_titles WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		var active int
		if err := rows.Scan(&title, &active); err != nil {
			return nil, err
		}
		titles = append(titles, title)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return titles, nil
}

// PrefixGetActiveTitle 查询当前使用称号
func (m *TitleManager) PrefixGetActiveTitle(userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var title string
	err := m.db.QueryRow(`
	SELECT title FROM player_titles WHERE user_id = ? AND active = 1
	`, userID).Scan(&title)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return title, err
}

// PrefixCommandChan 返回供外部发送指令的通道
func (m *TitleManager) PrefixCommandChan() chan<- TitleCommand {
	return m.CmdCh
}

// PrefixClose 关闭模块
func (m *TitleManager) PrefixClose() {
	if m.closed {
		return
	}
	close(m.CmdCh)
	m.wg.Wait()
	m.db.Close()
	m.closed = true
}

//==============================================================

// PrefixIniLoader 读取 ini 配置
func PrefixIniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/Prefix.ini")
	if err != nil {
		fmt.Println("[ERROR-Prefix]->Error:", err)
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
			fmt.Println("[ERROR-Prefix]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}

// PrefixCommandRegister 注册命令
func PrefixCommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string

	for section := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["Prefix"] = commandList
}

// PrefixCommandHandler 命令处理器
func PrefixCommandHandler(PrefixChan chan map[string]interface{}, cfg *execModules.Config, chatChan chan string) {
	var commandLines []string
	for command := range PrefixChan {
		commandString := command["command"].(string)
		commandArgs := strings.Split(command["commandArgs"].(string), " ")

		if cfg.Data[commandString]["PrefixRequire"].(string) != "default" {
			var1 := command["steamID"].(string)
			var2 := cfg.Data[commandString]["PrefixRequire"].(string)
			ok, _ := TitleMgr.PrefixHasTitle(var1, var2)
			if !ok {
				chatChan <- fmt.Sprintf("[Permission] 执行此命令需要称号【%s】", cfg.Data[commandString]["PrefixRequire"].(string))
				continue
			}
		}
		if len(commandArgs) == 0 {
			commandArgs = append(commandArgs, "")
		}
		if len(commandArgs) == 1 {
			commandArgs = append(commandArgs, "")
		}
		if commandArgs[1] == "" {
			commandArgs[1] = command["steamID"].(string)
		}

		Done := make(chan struct{})
		TitleMgr.CmdCh <- TitleCommand{UserID: commandArgs[1], Command: TitleCommandType(commandString), Title: commandArgs[0], Done: Done}
		<-Done
		commandLines = cfg.Data[commandString]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfglines := Public.CommandSelecterInterface.Selecter(command["steamID"].(string), cfgCommand)
			for _, lines := range cfglines {
				chatChan <- lines
				fmt.Println("[Prefix-Module]:" + lines)
			}
		}
	}
}

var TitleMgr *TitleManager

// Prefix 启动入口
func Prefix(regCommand *map[string][]string, PrefixChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {
	cfg := PrefixIniLoader()
	PrefixCommandRegister(cfg, regCommand)
	TitleMgr, _ = PrefixNewTitleManager("./db/Prefix.db", chatChan)
	Public.TitleInterface.PrefixHasTitle = TitleMgr.PrefixHasTitle
	Public.TitleInterface.PrefixGetActiveTitle = TitleMgr.PrefixGetActiveTitle
	go PrefixCommandHandler(PrefixChan, cfg, chatChan)
	close(initChan)
}

// 在 Prefix 包中直接定义别名
type TitleCommand = Public.TitleCommand
type TitleCommandType = Public.TitleCommandType

const (
	CommandGrant  = Public.CommandGrant
	CommandRemove = Public.CommandRemove
	CommandSet    = Public.CommandSet
	CommandUnSet  = Public.CommandUnSet
	CommandQuery  = Public.CommandQuery
	CommandUse    = Public.CommandUse
)
