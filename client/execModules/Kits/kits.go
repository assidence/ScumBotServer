package Kits

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/permissionBucket"
	"fmt"
	"regexp"
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
		var cfgChat string
		for _, cfgCommand := range commandLines {
			re := regexp.MustCompile(`^\w+`)
			cmd := re.FindString(cfgCommand)
			switch cmd {
			case "DestroyDiDi":
				if lw.Vehicles["BPC_Dirtbike"] == nil {
					chatChan <- fmt.Sprintf("找不到%s车辆类型的id列表", "BPC_Dirtbike")
					continue
				}
				for _, vehicleID := range lw.Vehicles["BPC_Dirtbike"] {
					cfgChat = fmt.Sprintf("DestroyVehicle %s", vehicleID)
					chatChan <- commandPrefix + cfgChat
					fmt.Println("[Kits-Module]:" + cfgChat)
					PMbucket.Consume(command["steamID"].(string), command["command"].(string))
				}
			case "SpawnItem":
				cfgChat = fmt.Sprintf(cfgCommand, command["steamID"].(string))
				chatChan <- commandPrefix + cfgChat
				fmt.Println("[Kits-Module]:" + cfgChat)
				PMbucket.Consume(command["steamID"].(string), command["command"].(string))
			case "ChangeCurrencyBalance":
				cfgChat = fmt.Sprintf(cfgCommand, command["steamID"].(string))
				chatChan <- commandPrefix + cfgChat
				fmt.Println("[Kits-Module]:" + cfgChat)
				PMbucket.Consume(command["steamID"].(string), command["command"].(string))
			case "SpawnVehicle":
				PLocationX := lw.Players[command["steamID"].(string)].LocationX
				PLocationY := lw.Players[command["steamID"].(string)].LocationY
				PLocationZ := lw.Players[command["steamID"].(string)].LocationZ
				cfgChat = fmt.Sprintf(cfgCommand, PLocationX, PLocationY, PLocationZ)
				chatChan <- commandPrefix + cfgChat
				fmt.Println("[Kits-Module]:" + cfgChat)
				PMbucket.Consume(command["steamID"].(string), command["command"].(string))
			default:
				fmt.Println("[ERROR-Kits]->Error:无法匹配命令 ", cmd)
			}
		}
		chatChan <- fmt.Sprintf("%s 礼包发放中 请耐心等待", command["nickName"].(string))
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
