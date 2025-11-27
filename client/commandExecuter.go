package main

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/Achievement"
	"ScumBotServer/client/execModules/Announcer"
	"ScumBotServer/client/execModules/CheckInRewardManager"
	"ScumBotServer/client/execModules/CommandSelecter"
	"ScumBotServer/client/execModules/DidiCar"
	"ScumBotServer/client/execModules/Kits"
	"ScumBotServer/client/execModules/LogWatcher"
	"ScumBotServer/client/execModules/PlayersInfo"
	"ScumBotServer/client/execModules/Prefix"
	"ScumBotServer/client/execModules/StatusMonitor"
	"ScumBotServer/client/execModules/scheduleTasks"
	"fmt"
)

var KitsChan = make(chan map[string]interface{}, 100)

var didiCarChan = make(chan map[string]interface{}, 100)

var ScheduleTaskChan = make(chan map[string]interface{}, 100)

var PrefixChan = make(chan map[string]interface{}, 100)

var AnnouncerChan = make(chan map[string]interface{}, 100)

var AchievementChan = make(chan map[string]interface{}, 100)

var PlayersInfoChan = make(chan map[string]interface{}, 100)

var CheckInRewardManagerChan = make(chan map[string]interface{}, 100)

var chatChan = make(chan string, 1024)

// moduleInit initiation the command function module
func moduleInit(regCommand *map[string][]string, sendChannel chan []byte) {

	var initChan = make(chan struct{})
	go commandSendToChat(initChan)
	<-initChan
	fmt.Println("[Module] 命令执行器已加载")

	initChan = make(chan struct{})
	go CommandSelecter.InitPublicSelecter(initChan)
	<-initChan
	fmt.Println("[Module] 游戏命令翻译器已加载")

	initChan = make(chan struct{})
	go LogWatcher.RunLogWatcher(initChan)
	<-initChan
	fmt.Println("[Module] 客户端日志监控模组已加载")

	initChan = make(chan struct{})
	go Prefix.Prefix(regCommand, PrefixChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 称号模组已加载")

	initChan = make(chan struct{})
	go Achievement.AchievementModule(regCommand, AchievementChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 成就模组已加载")

	initChan = make(chan struct{})
	go Kits.Kits(regCommand, KitsChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 新手礼包模组已加载")

	initChan = make(chan struct{})
	go DidiCar.DidiCar(regCommand, didiCarChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 滴滴车模组已加载")

	initChan = make(chan struct{})
	go scheduleTasks.ScheduleTasks(regCommand, ScheduleTaskChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 定时任务模组已加载")

	initChan = make(chan struct{})
	go Announcer.Announcer(regCommand, AnnouncerChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 广播模组已加载")

	initChan = make(chan struct{})
	go CheckInRewardManager.CheckInRewardManager(regCommand, CheckInRewardManagerChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 签到管理器已加载")

	initChan = make(chan struct{})
	go PlayersInfo.PlayersInfo(regCommand, PlayersInfoChan, AchievementChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 玩家状态解释及奖励器已加载")

	//======================================================================================
	initChan = make(chan struct{})
	go StatusMonitor.StatusMonitor(sendChannel, initChan)
	<-initChan
	fmt.Println("[Module] 状态网络广播模组已加载")

}

func listToMap(list []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, v := range list {
		m[v] = struct{}{}
	}
	return m
}

// commandSelecter detect which module match the command
func commandSelecter(command map[string]interface{}, regCommand *map[string][]string) {
	for moduleName, moduleCommands := range *regCommand {
		//moduleName = moduleName
		//fmt.Printf("%s::%s\n", moduleName, moduleCommands)
		commandMap := listToMap(moduleCommands)
		//fmt.Println(moduleName, moduleCommands)
		if _, ok := commandMap[command["command"].(string)]; ok {
			switch moduleName {
			case "Kits":
				KitsChan <- command
				return
			case "DidiCar":
				didiCarChan <- command
				return
			case "ScheduleTasks":
				ScheduleTaskChan <- command
				return
			case "Prefix":
				PrefixChan <- command
				return
			case "Announcer":
				AnnouncerChan <- command
				return
			case "Achievement":
				AchievementChan <- command
				return
			case "PlayersInfo":
				PlayersInfoChan <- command
				return
			case "CheckInRewardManager":
				CheckInRewardManagerChan <- command
				return
			}
		}
	}
	fmt.Printf("[Module] Command not Found!:%s\n", command["command"].(string))
}

// commandSendToChat send and execute command to chat
func commandSendToChat(iniChan chan struct{}) {
	close(iniChan)
	for commandString := range chatChan {
		//fmt.Println("[DEBUG]commandString:", commandString)
		err := execModules.SendChatMessage(commandString)
		if err != nil {
			fmt.Println("[CommandExecuter]->Error:", err)
		}
	}
}

// commandExecuter aka commandExecuter
func commandExecuter(execCommand chan map[string]interface{}, sendChannel chan []byte) {
	//var exec = ""
	var regCommand = make(map[string][]string)

	moduleInit(&regCommand, sendChannel)
	for command := range execCommand {
		//fmt.Println("[CommandExecuter]->Command:", command)
		commandSelecter(command, &regCommand)
	}
}
