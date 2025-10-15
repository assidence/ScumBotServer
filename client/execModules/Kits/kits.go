package Kits

import (
	"ScumBotServer/client/execModules"
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

func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string

	for section, _ := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["Kits"] = commandList
}

func CommandHandler(KitsChan chan map[string]interface{}, cfg *execModules.Config) {
	commandPrefix := "#"
	var commandLines []string
	for command := range KitsChan {
		//fmt.Println(command["command"].(string))
		//fmt.Println(cfg.Data)
		//fmt.Println(cfg.Data[command["command"].(string)]["Command"])
		commandLines = cfg.Data[command["command"].(string)]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			cfgChat := fmt.Sprintf(cfgCommand, command["steamID"].(string))
			fmt.Println("[Kits-Module]:" + cfgChat)
			execModules.SendChatMessage(commandPrefix + cfgChat)
		}
	}
}

func Kits(regCommand *map[string][]string, KitsChan chan map[string]interface{}, initChan chan struct{}) {
	cfg := iniLoader()
	CommandRegister(cfg, regCommand)
	go CommandHandler(KitsChan, cfg)
	close(initChan)
}
