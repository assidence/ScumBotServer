package scheduleTasks

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/CommandSelecter"
	"ScumBotServer/client/execModules/LogWacher"
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
	PmBucket, err := permissionBucket.NewManager("./db/didiCar.db")
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
		chatChan <- fmt.Sprintf("%s 任务执行中 请耐心等待", command["nickName"].(string))
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
	}
	defer PMbucket.Close()
}

// DailyTask 每天在指定小时和分钟执行 task，直到 quit 通道关闭
func DailyTask(schedule string, com string, ScheduleTasksChan chan map[string]interface{}, quit <-chan struct{}) {
	// Schedule 格式 HH:MM
	scheduleStr := schedule
	parts := strings.Split(scheduleStr, ":")
	hour, _ := strconv.Atoi(parts[0])
	mins, _ := strconv.Atoi(parts[1])
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), hour, mins, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}

			select {
			case <-time.After(time.Until(next)):
				fmt.Printf("[ScheduleTasks-Module] %s:定期任务已执行\n", com)
				TaskFunction(com, ScheduleTasksChan)
			case <-quit:
				fmt.Printf("[ScheduleTasks-Module] %s:定期任务已停止\n", com)
				return
			}
		}
	}()
}

// IntervalTask 每隔指定的时间间隔执行 task，直到 quit 通道关闭
func BuildInPlayerMonitor(interval time.Duration, chatChan chan string) {
	com := "#ListPlayers"
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				//fmt.Printf("[ScheduleTasks-Module] %s:定期任务已执行\n", com)
				chatChan <- com
			}
		}
	}()
}

func TaskFunction(com string, ScheduleTasksChan chan map[string]interface{}) {
	command := make(map[string]interface{})
	command["steamID"] = "000000"
	command["nickName"] = "System"
	command["command"] = com
	ScheduleTasksChan <- command
}

func ScheduleTasksTickerStartup(ScheduleTasksChan chan map[string]interface{}, cfg *execModules.Config) {
	for command, commandCFG := range cfg.Data {
		schedIface, ok := commandCFG["Schedule"]
		if !ok || schedIface == nil {
			fmt.Printf("[ScheduleTasks-Module] 跳过任务 %s，没有 Schedule 配置\n", command)
			continue
		}

		// 强制转换为字符串
		schedStr := fmt.Sprint(schedIface)
		if strings.TrimSpace(schedStr) == "" {
			fmt.Printf("[ScheduleTasks-Module] 跳过任务 %s，Schedule 配置为空\n", command)
			continue
		}

		quit := make(chan struct{})
		DailyTask(schedStr, command, ScheduleTasksChan, quit)
		fmt.Printf("[ScheduleTasks-Module]: 已加载定期任务：%s (%s)\n", command, schedStr)
	}
}

func ScheduleTasks(regCommand *map[string][]string, ScheduleTasksChan chan map[string]interface{}, chatChan chan string, lw *LogWacher.LogWatcher, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	CommandRegister(cfg, regCommand)
	go CommandHandler(ScheduleTasksChan, cfg, PmBucket, chatChan, lw)
	ScheduleTasksTickerStartup(ScheduleTasksChan, cfg)
	BuildInPlayerMonitor(10*time.Second, chatChan)
	close(initChan)
	//select {}
}
