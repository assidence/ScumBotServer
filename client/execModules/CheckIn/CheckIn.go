package CheckIn

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/Prefix"
	"ScumBotServer/client/execModules/permissionBucket"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// iniLoader: 读取 ./ini/CheckIn.ini（与其他模块风格一致）
// 将每个 section 的 Command 文件读取为 []string 放回 cfg.Data[section]["Command"]
func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/checkin.ini")
	if err != nil {
		fmt.Println("[ERROR-CheckIn]->Error:", err)
		return &execModules.Config{}
	}

	for section, secMap := range cfg.Data {
		if section == "DEFAULT" {
			continue
		}
		// 如果有 Command 字段，读取命令文件（保持与其他模块一致的行为）
		if cmdFileIface, ok := secMap["Command"]; ok && cmdFileIface != nil {
			cmdFile := fmt.Sprint(cmdFileIface)
			lines, err := execModules.CommandFileReadLines(cmdFile)
			if err != nil {
				fmt.Printf("[ERROR-CheckIn]->Read Command file %s failed: %v\n", cmdFile, err)
				// 保留原值
				continue
			}
			cfg.Data[section]["Command"] = lines
		}
	}

	return cfg
}

// createPermissionBucket: 保持与其它模块相同的权限桶创建风格
func createPermissionBucket() *permissionBucket.Manager {
	pm, err := permissionBucket.NewManager("./db/CheckInPerm.db")
	if err != nil {
		panic(err)
	}
	//defer pm.Close()
	return pm
}

// CommandRegister: 注册模块提供的命令到 regCommand 映射
// 与其它模块保持一致：将所有 ini section 名称作为命令条目
func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string
	for section, _ := range cfg.Data {
		// 跳过 DEFAULT
		if section == "DEFAULT" {
			continue
		}
		commandList = append(commandList, section)
	}
	(*regCommand)["CheckIn"] = commandList
}

// CommandHandler: 与项目其它模块风格保持一致
// 从 CheckInChan 接收 map[string]interface{}，字段预期： steamID, nickName, command, action
func CommandHandler(CheckInChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string, lw *LogWacher.LogWatcher) {
	for command := range CheckInChan {
		nick := fmt.Sprint(command["nickName"])
		chatChan <- fmt.Sprintf("%s 签到处理中 请耐心等待", nick)

		steamID := fmt.Sprint(command["steamID"])
		cmdName := fmt.Sprint(command["command"])
		action := ""
		if a, ok := command["action"]; ok && a != nil {
			action = strings.ToLower(fmt.Sprint(a))
		}

		// 权限检查（使用 ini 中的 Command 名称作为权限 key）
		ok, msg := PMbucket.CanExecute(steamID, cmdName)
		if !ok {
			fmt.Println("[ERROR-CheckIn]->Error:", msg)
			chatChan <- fmt.Sprintf("%s 权限不足：%s", nick, msg)
			continue
		}

		// 默认兼容：如果未指定 action，但 command 就是签到的 section 名，认为是 sign
		if action == "" {
			action = "sign"
		}

		switch action {
		case "sign":
			resp, err := handleSign(steamID, nick, cfg, chatChan, lw)
			if err != nil {
				fmt.Println("[CheckIn-Module] sign error:", err)
				chatChan <- "签到失败，内部错误。"
				continue
			}
			chatChan <- resp
		case "query":
			resp, err := handleQuery(steamID, nick)
			if err != nil {
				fmt.Println("[CheckIn-Module] query error:", err)
				chatChan <- "查询签到失败，内部错误。"
				continue
			}
			chatChan <- resp
		default:
			chatChan <- fmt.Sprintf("未知操作: %s", action)
		}
	}
	defer PMbucket.Close()
}

// handleSign: 主业务入口 - 签到并（可选）发放奖励
// 按项目风格：奖励用 cfg.Data[section][\"Command\"] 中的命令行模板（如果存在）通过 CommandSelecter 发放
func handleSign(steamID, nick string, cfg *execModules.Config, chatChan chan string, lw *LogWacher.LogWatcher) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	today := todayDate()
	lastDate, streak, total, err := getRecord(steamID)
	if err == sql.ErrNoRows {
		// 首次签到
		if err := insertRecord(steamID, today); err != nil {
			return "", err
		}
		// 发放奖励：尝试从 cfg 中读取默认 section "@签到" 或 "[@签到]" 对应的 Command 列表
		//sendRewardsFromCfg("签到", steamID, chatChan, lw, cfg)
		return fmt.Sprintf("%s 签到成功！连续签到：1 天，总签到：1 次。奖励已发放（如有配置）。", nick), nil
	} else if err != nil {
		return "", err
	}

	if lastDate == today {
		return fmt.Sprintf("%s 今天已签到过。", nick), nil
	}

	last, perr := time.Parse("2006-01-02", lastDate)
	if perr != nil {
		// 若解析失败则重置连续
		streak = 1
	} else {
		if last.Add(24*time.Hour).Format("2006-01-02") == today {
			streak = streak + 1
		} else {
			streak = 1
		}
	}
	total = total + 1

	if err := updateRecord(steamID, today, streak, total); err != nil {
		return "", err
	}

	// 发放奖励（如果配置了）
	//sendRewardsFromCfg("签到", steamID, chatChan, lw, cfg)
	return fmt.Sprintf("%s 签到成功！连续签到：%d 天，总签到：%d 次。奖励已发放（如有配置）。", nick, streak, total), nil
}

// handleQuery: 返回查询文本
func handleQuery(steamID, nick string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	lastDate, streak, total, err := getRecord(steamID)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("%s 暂无签到记录。", nick), nil
	} else if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s 上次签到：%s，连续签到：%d 天，总签到：%d 次。", nick, lastDate, streak, total), nil
}

/*
// sendRewardsFromCfg: 从 cfg（ini）中寻找对应 section 的 Command 字段并执行
// 这里的 sectionName 为 ini 中的节名（例如: "@签到" 或 "签到"）
// 为了兼容现有模块，我们尝试多种 key：1) 完整 section 名 2) 去掉 @ 前缀后的名字
func sendRewardsFromCfg(sectionName, steamID string, chatChan chan string, lw *LogWacher.LogWatcher, cfg *execModules.Config) {
	if cfg == nil {
		return
	}

	// 直接尝试以 sectionName 查找
	tryKeys := []string{sectionName}
	// 如果 sectionName 没有 @ 前缀，也加上 @ 形式尝试
	if !strings.HasPrefix(sectionName, "@") {
		tryKeys = append(tryKeys, "@"+sectionName)
	}
	// 另外也遍历所有配置项，找一个包含“签到”或 sectionName 的条目（兼容性）
tryLoop:
	for _, key := range tryKeys {
		if sec, ok := cfg.Data[key]; ok {
			if cmdListIface, ok2 := sec["Command"]; ok2 && cmdListIface != nil {
				// cfg.Data[section]["Command"] 在 iniLoader 已经被读成 []string（如果配置了文件）
				switch v := cmdListIface.(type) {
				case []string:
					for _, tpl := range v {
						// 可能的模板替换：{steamID}
						line := strings.ReplaceAll(tpl, "{steamID}", steamID)
						// 使用 CommandSelecter 解析成最终需要发送的行
						lines := CommandSelecter.Selecter(steamID, line, lw)
						for _, l := range lines {
							chatChan <- l
						}
					}
				case string:
					line := strings.ReplaceAll(v, "{steamID}", steamID)
					lines := CommandSelecter.Selecter(steamID, line, lw)
					for _, l := range lines {
						chatChan <- l
					}
				}
			}
			// 找到就返回
			return
		}
	}

	// 作为兜底：查找任意含有关键词的 section
	for key, sec := range cfg.Data {
		if strings.Contains(strings.ToLower(key), strings.ToLower(sectionName)) || strings.Contains(strings.ToLower(key), "签到") {
			if cmdListIface, ok2 := sec["Command"]; ok2 && cmdListIface != nil {
				switch v := cmdListIface.(type) {
				case []string:
					for _, tpl := range v {
						line := strings.ReplaceAll(tpl, "{steamID}", steamID)
						lines := CommandSelecter.Selecter(steamID, line, lw)
						for _, l := range lines {
							chatChan <- l
						}
					}
				case string:
					line := strings.ReplaceAll(v, "{steamID}", steamID)
					lines := CommandSelecter.Selecter(steamID, line, lw)
					for _, l := range lines {
						chatChan <- l
					}
				}
			}
			return
		}
	}
}
*/
// Module 启动入口（与其他模块保持签名一致）
// regCommand: 全局命令注册表（模块应把其 ini 中的每个 section 名注册进去）
// CheckInChan: 模块接收命令通道
func CheckInModule(regCommand *map[string][]string, CheckInChan chan map[string]interface{}, chatChan chan string, lw *LogWacher.LogWatcher, TitleManager *Prefix.TitleManager, initChan chan struct{}) {
	// 1. 读取 ini
	cfg := iniLoader()

	// 2. 打开/初始化 DB
	if err := InitDB(); err != nil {
		panic(err)
	}

	// 3. 权限管理对象（独立文件），并把 ini 配置传给它（与其它模块一致）
	pm := createPermissionBucket()
	pm.CommandConfigChan <- cfg.Data
	pm.TitleManager = TitleManager

	// 4. 注册命令到全局命令表
	CommandRegister(cfg, regCommand)

	// 5. 启动命令处理 goroutine（与项目其它模块同风格）
	go CommandHandler(CheckInChan, cfg, pm, chatChan, lw)

	// 6. 标记初始化完成（caller 一般会 <-initChan）
	close(initChan)
}
