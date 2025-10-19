package Kits

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/CommandSelecter"
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/permissionBucket"
	"fmt"
)

func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/Kits.ini")
	//fmt.Println(cfg)
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

func CommandHandler(KitsChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string, lw *LogWacher.LogWatcher) {
	var commandLines []string
	for command := range KitsChan {
		chatChan <- fmt.Sprintf("%s 礼包发放中 请耐心等待", command["nickName"].(string))
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
			cfglines := CommandSelecter.Selecter(command["steamID"].(string), cfgCommand, lw)
			for _, lines := range cfglines {
				chatChan <- lines
				fmt.Println("[Kits-Module]:" + lines)
			}
		}
		chatChan <- fmt.Sprintf("%s 礼包发放完成", command["nickName"].(string))
	}
	defer PMbucket.Close()
}

func Kits(regCommand *map[string][]string, KitsChan chan map[string]interface{}, chatChan chan string, lw *LogWacher.LogWatcher, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	CommandRegister(cfg, regCommand)
	go CommandHandler(KitsChan, cfg, PmBucket, chatChan, lw)
	close(initChan)
	//select {}
}
