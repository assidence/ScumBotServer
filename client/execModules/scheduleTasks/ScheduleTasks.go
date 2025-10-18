package scheduleTasks

import (
	"ScumBotServer/client/execModules"
	"fmt"
)

func iniLoader() *execModules.Config {
	cfg, err := execModules.NewConfig("./ini/ScheduleTasks.ini")
	//fmt.Println(cfg)
	if err != nil {
		fmt.Println("[ERROR-ScheduleTask]->Error:", err)
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
			fmt.Println("[ERROR-ScheduleTask]->Error:", err)
		}
		cfg.Data[section]["Command"] = commandList
	}
	return cfg
}
