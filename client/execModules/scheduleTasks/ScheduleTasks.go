package scheduleTasks

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/CommandSelecter"
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/Prefix"
	"ScumBotServer/client/execModules/permissionBucket"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/ScheduleTasks.ini")
	//fmt.Println(cfg)
	if err != nil {
		fmt.Println("[ERROR-ScheduleTask]->Error:", err)
		return &execModules.Config{}
	}
	var commandList []string
	//fmt.Println(cfg.Data)
	for section, secMap := range cfg.Data {
		//fmt.Println(secMap)
		if section == "DEFAULT" {
			continue
		}
		commandFilePart := secMap["Command"].(string)
		commandList, err = execModules.CommandFileReadLines(commandFilePart)
		if err != nil {
			fmt.Println("[ERROR-ScheduleTask]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}

func createPermissionBucket() *permissionBucket.Manager {
	PmBucket, err := permissionBucket.NewManager("./db/ScheduleTasks.db")
	if err != nil {
		panic(err)
	}
	//defer PmBucket.Close()

	return PmBucket
}

func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string

	for section, _ := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["ScheduleTasks"] = commandList
}

func CommandHandler(ScheduleTasksChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string, lw *LogWacher.LogWatcher) {
	var commandLines []string
	for command := range ScheduleTasksChan {
		//chatChan <- fmt.Sprintf("%s 任务执行中 请耐心等待", command["nickName"].(string))
		//fmt.Println(command["command"].(string))
		//fmt.Println(cfg.Data)
		//fmt.Println(cfg.Data[command["command"].(string)]["Command"])
		ok, msg := PMbucket.CanExecute(command["steamID"].(string), command["command"].(string))
		//fmt.Println(command["steamID"].(string) + command["command"].(string))
		if !ok {
			fmt.Println("[ERROR-ScheduleTasks]->Error:", msg)
			continue
		}

		commandLines = cfg.Data[command["command"].(string)]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfglines := CommandSelecter.Selecter(command["steamID"].(string), cfgCommand, lw)
			for _, lines := range cfglines {
				chatChan <- lines
				fmt.Println("[ScheduleTasks-Module]:" + lines)
			}
		}
		//chatChan <- fmt.Sprintf("%s 任务执行完成", command["nickName"].(string))
	}
	defer PMbucket.Close()
}

// IntervalTask 每隔指定的时间间隔执行 task，直到 quit 通道关闭
func IntervalTask(intervalStr string, com string, ScheduleTasksChan chan map[string]interface{}, quit <-chan struct{}) {
	duration, err := time.ParseDuration(strings.TrimSpace(intervalStr))
	if err != nil {
		fmt.Printf("[ScheduleTasks-Module] 无效的间隔格式 (%s): %v\n", intervalStr, err)
		return
	}

	go func() {
		ticker := time.NewTicker(duration)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[ScheduleTasks-Module] %s:间隔任务已执行 (every %s)\n", com, duration)
				TaskFunction(com, ScheduleTasksChan)
			case <-quit:
				fmt.Printf("[ScheduleTasks-Module] %s:间隔任务已停止\n", com)
				return
			}
		}
	}()
}

// DailyTask 每天在指定小时和分钟执行 task，直到 quit 通道关闭
func DailyTask(schedule string, com string, ScheduleTasksChan chan map[string]interface{}, quit <-chan struct{}) {
	parts := strings.Split(strings.TrimSpace(schedule), ":")
	if len(parts) != 2 {
		fmt.Printf("[ScheduleTasks-Module] 无效的时间格式 (%s)，应为 HH:MM，例如 03:30 或 15:05\n", schedule)
		return
	}
	hour, err1 := strconv.Atoi(parts[0])
	mins, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || mins < 0 || mins > 59 {
		fmt.Printf("[ScheduleTasks-Module] 时间格式错误 (%s)，小时应在 0-23，分钟应在 0-59\n", schedule)
		return
	}

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), hour, mins, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}

			select {
			case <-time.After(time.Until(next)):
				fmt.Printf("[ScheduleTasks-Module] %s:定时任务已执行 (at %02d:%02d)\n", com, hour, mins)
				TaskFunction(com, ScheduleTasksChan)
			case <-quit:
				fmt.Printf("[ScheduleTasks-Module] %s:定时任务已停止\n", com)
				return
			}
		}
	}()
}

func ScheduleTasksTickerStartup(ScheduleTasksChan chan map[string]interface{}, cfg *execModules.Config) {
	for command, commandCFG := range cfg.Data {
		schedIface, ok := commandCFG["Schedule"]
		if !ok || schedIface == nil {
			fmt.Printf("[ScheduleTasks-Module] 跳过任务 %s，没有 Schedule 配置\n", command)
			continue
		}

		// 优先读取类型（可选）
		schedType := ""
		if t, ok := commandCFG["ScheduleType"]; ok && t != nil {
			schedType = strings.ToLower(strings.TrimSpace(fmt.Sprint(t)))
		}

		schedStr := strings.TrimSpace(fmt.Sprint(schedIface))
		if schedStr == "" {
			fmt.Printf("[ScheduleTasks-Module] 跳过任务 %s，Schedule 配置为空\n", command)
			continue
		}

		quit := make(chan struct{})

		// 若用户未指定类型，则尝试自动识别：先尝试 ParseDuration -> interval；否则尝试 HH:MM -> daily
		if schedType == "" {
			if _, err := time.ParseDuration(schedStr); err == nil {
				schedType = "interval"
			} else {
				schedType = "daily"
			}
		}

		switch schedType {
		case "daily":
			DailyTask(schedStr, command, ScheduleTasksChan, quit)
			fmt.Printf("[ScheduleTasks-Module]: 已加载每日定时任务：%s (%s)\n", command, schedStr)
		case "interval":
			IntervalTask(schedStr, command, ScheduleTasksChan, quit)
			fmt.Printf("[ScheduleTasks-Module]: 已加载间隔任务：%s (%s)\n", command, schedStr)
		default:
			fmt.Printf("[ScheduleTasks-Module]: 跳过任务 %s，未知的 ScheduleType: %s\n", command, schedType)
		}
	}
}

func TaskFunction(com string, ScheduleTasksChan chan map[string]interface{}) {
	command := make(map[string]interface{})
	command["steamID"] = "000000"
	command["nickName"] = "System"
	command["command"] = com
	ScheduleTasksChan <- command
}

func ScheduleTasks(regCommand *map[string][]string, ScheduleTasksChan chan map[string]interface{}, chatChan chan string, lw *LogWacher.LogWatcher, TitleManager *Prefix.TitleManager, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	PmBucket.TitleManager = TitleManager
	CommandRegister(cfg, regCommand)
	go CommandHandler(ScheduleTasksChan, cfg, PmBucket, chatChan, lw)
	ScheduleTasksTickerStartup(ScheduleTasksChan, cfg)
	//BuildInPlayerMonitor(10*time.Second, chatChan)
	close(initChan)
	//select {}
}
