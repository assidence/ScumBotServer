package Public

import (
	"ScumBotServer/client/execModules"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// TitleCommandType 指令类型
type TitleCommandType string

const (
	CommandGrant  TitleCommandType = "@给予称号" // 授予称号
	CommandRemove TitleCommandType = "@移除称号" // 移除称号
	CommandSet    TitleCommandType = "@设置称号" // 设置当前称号
	CommandUnSet  TitleCommandType = "@隐藏称号"
)

// TitleCommand 外部模块发送过来的指令
type TitleCommand struct {
	UserID  string
	Command TitleCommandType
	Title   string
	Done    chan struct{}
}

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
		CmdCh: make(chan TitleCommand, 64),
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
	if GlobalLogWatcher == nil {
		fmt.Println("[Prefix-Panic] LogWatcher is nil")
		return
	}
	defer m.wg.Done()
	for cmd := range m.CmdCh {
		switch cmd.Command {
		case CommandGrant:
			if err := m.PrefixGrantTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s授予称号失败: %v", GlobalLogWatcher.Players[cmd.UserID].Name, err)
				fmt.Println("[Error-Prefix] " + fmt.Sprintf("%s授予称号失败: %v", cmd.UserID, err))
			} else {
				chatChan <- fmt.Sprintf("%s获得称号 %s", GlobalLogWatcher.Players[cmd.UserID].Name, cmd.Title)
				fmt.Printf("[Prefix-Module]  玩家 %s 获得称号 %s\n", cmd.UserID, cmd.Title)
			}

		case CommandRemove:
			if err := m.PrefixRemoveTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s移除称号失败: %v", GlobalLogWatcher.Players[cmd.UserID].Name, err)
				fmt.Println("[Error-Prefix]" + fmt.Sprintf("[Error-Prefix] %s移除称号失败: %v", cmd.UserID, err))
			} else {
				chatChan <- fmt.Sprintf("玩家 %s 移除称号 %s", GlobalLogWatcher.Players[cmd.UserID].Name, cmd.Title)
				fmt.Printf("[Prefix-Module] 玩家 %s 移除称号 %s\n", cmd.UserID, cmd.Title)
			}

		case CommandSet:
			if err := m.PrefixSetActiveTitle(cmd.UserID, cmd.Title); err != nil {
				chatChan <- fmt.Sprintf("%s设置当前称号失败: %v", GlobalLogWatcher.Players[cmd.UserID].Name, err)
				fmt.Printf("[Error-Prefix] 玩家 %s设置当前称号失败: %v\n", cmd.UserID, err)
			} else {
				p := GlobalLogWatcher.Players[cmd.UserID]
				p.Prefix = cmd.Title
				GlobalLogWatcher.Players[cmd.UserID] = p
				line := fmt.Sprintf("#SetFakeName %s -★%s★-%s", cmd.UserID, cmd.Title, p.Name)
				chatChan <- line
				chatChan <- fmt.Sprintf("%s当前称号设为 %s 可使用@隐藏称号 来取消", p.Name, cmd.Title)
				fmt.Println("[Prefix-Module]:" + line)
			}
		case CommandUnSet:
			p := GlobalLogWatcher.Players[cmd.UserID]
			p.Prefix = ""
			GlobalLogWatcher.Players[cmd.UserID] = p
			line := fmt.Sprintf("#SetFakeName %s %s", cmd.UserID, p.Name)
			chatChan <- line
			chatChan <- fmt.Sprintf("%s当前称号已取消展示", p.Name)
			fmt.Println("[Prefix-Module]:" + line)
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
			ok, _ := GlobalTitleManager.PrefixHasTitle(var1, var2)
			if !ok {
				chatChan <- fmt.Sprintf("[Permission] 执行此命令需要称号【%s】", cfg.Data[commandString]["PrefixRequire"].(string))
				continue
			}
		}
		if len(commandArgs) == 1 {
			commandArgs = append(commandArgs, "")
		}
		if commandArgs[1] == "" {
			commandArgs[1] = command["steamID"].(string)
		}

		Done := make(chan struct{})
		GlobalTitleManager.CmdCh <- TitleCommand{UserID: commandArgs[1], Command: TitleCommandType(commandString), Title: commandArgs[0], Done: Done}
		<-Done
		commandLines = cfg.Data[commandString]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfglines := Selecter(command["steamID"].(string), cfgCommand)
			for _, lines := range cfglines {
				chatChan <- lines
				fmt.Println("[Prefix-Module]:" + lines)
			}
		}
	}
}

// Prefix 启动入口
func Prefix(regCommand *map[string][]string, PrefixChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {
	cfg := PrefixIniLoader()
	PrefixCommandRegister(cfg, regCommand)
	GlobalTitleManager, _ = PrefixNewTitleManager("./db/Prefix.db", chatChan)
	go PrefixCommandHandler(PrefixChan, cfg, chatChan)
	close(initChan)
}
