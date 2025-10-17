package DidiCar

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/permissionBucket"
	"fmt"
)

func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/DidiCar.ini")
	if err != nil {
		fmt.Println("[ERROR-DiDiCar]->Error:", err)
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
			fmt.Println("[ERROR-DiDiCar]->Error:", err)
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
	(*regCommand)["DidiCar"] = commandList
}

func CommandHandler(didiCarChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string, lw *LogWacher.LogWatcher) {
	//fmt.Println("im here")
	commandPrefix := "#"
	var commandLines []string
	for command := range didiCarChan {
		//fmt.Println("DidiCar Handler is Up and Running")
		//fmt.Println(command["command"].(string))
		//fmt.Println(cfg.Data)
		//fmt.Println(cfg.Data[command["command"].(string)]["Command"])
		ok, msg := PMbucket.CanExecute(command["steamID"].(string), command["command"].(string))
		//fmt.Println(command["steamID"].(string) + command["command"].(string))
		if !ok {
			fmt.Println("[ERROR-DidiCar]->Error:", msg)
			continue
		}

		commandLines = cfg.Data[command["command"].(string)]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			/*
				err := execModules.SendChatMessage(commandPrefix + cfgChat)
				if err != nil {
					fmt.Println("[ERROR-Kit]->Error:", err)
				}

			*/
			PLocationX := lw.Players[command["steamID"].(string)].LocationX
			PLocationY := lw.Players[command["steamID"].(string)].LocationY
			PLocationZ := lw.Players[command["steamID"].(string)].LocationZ
			cfgChat := fmt.Sprintf(cfgCommand, PLocationX, PLocationY, PLocationZ)
			fmt.Println("[DidiCar-Module]:" + cfgChat)
			chatChan <- commandPrefix + cfgChat
		}
		chatChan <- fmt.Sprintf("%s 滴滴车呼叫中 请耐心等待", command["nickName"].(string))
		PMbucket.Consume(command["steamID"].(string), command["command"].(string))
	}
	defer PMbucket.Close()
}

func DidiCar(regCommand *map[string][]string, didiCarChan chan map[string]interface{}, chatChan chan string, lw *LogWacher.LogWatcher, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	CommandRegister(cfg, regCommand)
	go CommandHandler(didiCarChan, cfg, PmBucket, chatChan, lw)
	close(initChan)
	//select {}
}
