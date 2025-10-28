package Announcer

import (
	"ScumBotServer/client/execModules"
	"ScumBotServer/client/execModules/permissionBucket"
	"fmt"
)

func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/Announcer.ini")
	//fmt.Println(cfg)
	if err != nil {
		fmt.Println("[ERROR-Announcer]->Error:", err)
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
			fmt.Println("[ERROR-Announcer]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}

func createPermissionBucket() *permissionBucket.Manager {
	PmBucket, err := permissionBucket.NewManager("./db/Announcer.db")
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
	(*regCommand)["Announcer"] = commandList
}

func CommandHandler(AnnouncerChan chan map[string]interface{}, cfg *execModules.Config, PMbucket *permissionBucket.Manager, chatChan chan string) {
	var commandLines []string
	for command := range AnnouncerChan {
		//chatChan <- fmt.Sprintf("%s 礼包发放中 请耐心等待", command["nickName"].(string))
		//fmt.Println(command["command"].(string))
		//fmt.Println(cfg.Data)
		//fmt.Println(cfg.Data[command["command"].(string)]["Command"])
		ok, msg := PMbucket.CanExecute(command["steamID"].(string), command["command"].(string))
		//fmt.Println(command["steamID"].(string) + command["command"].(string))
		if !ok {
			fmt.Println("[ERROR-Announcer]->Error:", msg)
			continue
		}

		commandLines = cfg.Data[command["command"].(string)]["Command"].([]string)
		for _, cfgCommand := range commandLines {
			lines := fmt.Sprintf(cfgCommand, command["commandArgs"].(string))
			chatChan <- lines
			fmt.Println("[Announcer-Module]:" + lines)
		}
		//chatChan <- fmt.Sprintf("%s 礼包发放完成", command["nickName"].(string))
	}
	defer PMbucket.Close()
}

//var lw = PublicInterface.LogWatcher

func Announcer(regCommand *map[string][]string, AnnouncerChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {
	cfg := iniLoader()
	PmBucket := createPermissionBucket()
	PmBucket.CommandConfigChan <- cfg.Data
	CommandRegister(cfg, regCommand)
	go CommandHandler(AnnouncerChan, cfg, PmBucket, chatChan)
	close(initChan)
	//select {}
}
