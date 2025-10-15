package main

import (
	"ScumBotServer/client/execModules/Kits"
	"fmt"
)

var KitsChan = make(chan map[string]interface{})

// moduleInit initiation the command function module
func moduleInit(regCommand *map[string][]string) {
	var initChan = make(chan struct{})
	go Kits.Kits(regCommand, KitsChan, initChan)
	<-initChan
	fmt.Println("[Module] 新手礼包模组已加载")
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
		moduleName = moduleName
		commandMap := listToMap(moduleCommands)
		if _, ok := commandMap[command["command"].(string)]; ok {
			switch moduleName {
			case "Kits":
				KitsChan <- command
			default:
				continue
			}
		} else {
			fmt.Printf("[Module] Command not Found!:%s\n", command["command"].(string))
		}
	}
}

// commandExecuter aka commandExecuter
func commandExecuter(execCommand chan map[string]interface{}) {
	//var exec = ""
	var regCommand = make(map[string][]string)

	moduleInit(&regCommand)
	for command := range execCommand {
		commandSelecter(command, &regCommand)
	}
}
