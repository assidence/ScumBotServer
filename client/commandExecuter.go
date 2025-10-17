package main

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/DidiCar"
	"ScumBotServer/client/execModules/Kits"
	"ScumBotServer/client/execModules/LogWacher"
	"fmt"
)

var KitsChan = make(chan map[string]interface{}, 100)

var didiCarChan = make(chan map[string]interface{}, 100)

var chatChan = make(chan string, 100)

var lw = &LogWacher.LogWatcher{
	FilePath: "",
	Interval: 0,
	Players:  nil,
}

// moduleInit initiation the command function module
func moduleInit(regCommand *map[string][]string) {
	var initChan = make(chan struct{})
	go commandSendToChat(initChan)
	<-initChan
	fmt.Println("[Module] 命令执行器已加载")

	initChan = make(chan struct{})
	var LWChan = make(chan *LogWacher.LogWatcher)
	go LogWacher.RunLogWatcher(lw, LWChan, initChan)
	lw = <-LWChan
	<-initChan
	fmt.Println("[Module] 客户端日志监控模组已加载")

	initChan = make(chan struct{})
	go Kits.Kits(regCommand, KitsChan, chatChan, initChan)
	<-initChan
	fmt.Println("[Module] 新手礼包模组已加载")

	initChan = make(chan struct{})
	go DidiCar.DidiCar(regCommand, didiCarChan, chatChan, lw, initChan)
	<-initChan
	fmt.Println("[Module] 滴滴车模组已加载")

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
		if _, ok := commandMap[command["command"].(string)]; ok {
			switch moduleName {
			case "Kits":
				KitsChan <- command
				return
			case "DidiCar":
				didiCarChan <- command
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
		err := execModules.SendChatMessage(commandString)
		if err != nil {
			fmt.Println("[CommandExecuter]->Error:", err)
		}
	}
}

// commandExecuter aka commandExecuter
func commandExecuter(execCommand chan map[string]interface{}) {
	//var exec = ""
	var regCommand = make(map[string][]string)

	moduleInit(&regCommand)
	for command := range execCommand {
		//fmt.Println("[CommandExecuter]->Command:", command)
		commandSelecter(command, &regCommand)
	}
}
