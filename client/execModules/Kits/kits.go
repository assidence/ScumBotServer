package Kits

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/permissionBucket"
	"fmt"
)

func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/Kits.ini")
	if err != nil {
		fmt.Println("[ERROR-KIT]->Error:", err)
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
			fmt.Println("[ERROR-KIT]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}

func createPermissionBucket() *permissionBucket.Manager {
	PmBucket, err := permissionBucket.NewManager("./db/Kits.db")
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
	(*regCommand)["Kits"] = commandList
}

func CommandHandler(KitsChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string) {
	commandPrefix := "#"
	var commandLines []string
	for command := range KitsChan {
		//fmt.Println(command["command"].(string))
		//fmt.Println(cfg.Data)
		//fmt.Println(cfg.Data[command["command"].(string)]["Command"])
		ok, msg := PMbucket.CanExecute(command["steamID"].(string), command["command"].(string))
		//fmt.Println(command["steamID"].(string) + command["command"].(string))
		if !ok {
			fmt.Println("[ERROR-KIT]->Error:", msg)
			continue
		}
		commandLines = cfg.Data[command["command"].(string)]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfgChat := fmt.Sprintf(cfgCommand, command["steamID"].(string))
			fmt.Println("[Kits-Module]:" + cfgChat)
			/*
				err := execModules.SendChatMessage(commandPrefix + cfgChat)
				if err != nil {
					fmt.Println("[ERROR-Kit]->Error:", err)
				}

			*/
			chatChan <- commandPrefix + cfgChat
		}
		chatChan <- fmt.Sprintf("%s 物品发放中 请耐心等待", command["nickName"].(string))
		PMbucket.Consume(command["steamID"].(string), command["command"].(string))
	}
	defer PMbucket.Close()
}

func Kits(regCommand *map[string][]string, KitsChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	permissionBucket.CommandConfigChan <- cfg.Data
	CommandRegister(cfg, regCommand)
	go CommandHandler(KitsChan, cfg, PmBucket, chatChan)
	close(initChan)
}
