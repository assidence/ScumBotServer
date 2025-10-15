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
	/*
		for section, secMap := range cfg.Data {
			fmt.Println("section:", section)
			for key, value := range secMap {
				fmt.Println("key:", key, "value:", value)
			}
		}

	*/

	return cfg
}

func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string

	for section, _ := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["Kits"] = commandList
}

func CommandHandler(KitsChan chan map[string]interface{}) {
	for command := range KitsChan {
		fmt.Println("[INFO-Kits]->", command)
	}
}

func Kits(regCommand *map[string][]string, KitsChan chan map[string]interface{}, initChan chan struct{}) {
	cfg := iniLoader()
	CommandRegister(cfg, regCommand)
	go CommandHandler(KitsChan)
	close(initChan)
}
