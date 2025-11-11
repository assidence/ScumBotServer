package PlayersInfo

import (
	"fmt"
	"log"

	"ScumBotServer/client/execModules"
)

// Debug 输出管理
var DebugEnabled = true // 可以在模块初始化或运行时控制开关

func debug(format string, a ...interface{}) {
	if DebugEnabled {
		log.Printf("[PlayersInfo-DEBUG] "+format, a...)
	}
}

// 注册命令
func CommandRegister(cfg *execModules.Config, regCommand *map[string][]string) {
	var commandList []string
	for section := range cfg.Data {
		commandList = append(commandList, section)
	}
	(*regCommand)["PlayersInfo"] = commandList
}

// 命令处理
func CommandHandler(PlayersInfoChan chan map[string]interface{}, AchievementChan chan map[string]interface{}, chatChan chan string) {
	for command := range PlayersInfoChan {
		cmdType, ok := command["command"].(string)
		if !ok {
			fmt.Println("[PlayersInfo-Error] command 类型断言失败")
			continue
		}

		if cmdType == "PlayersAttributeInfo" {
			players := command["commandArgs"].(map[string]interface{})
			result := PlayerAchievementTrick(players)
			//fmt.Println("finish")
			if len(result) == 0 {
				continue
			}
			PlayersAttributeInfoexecData := map[string]interface{}{
				"steamID":  "000000",
				"nickName": "System",
				"command":  "",
				//"commandArgs": steamId + "-" + "naked" + "-" + "1",
				"commandArgs": "",
			}
			for group, ids := range result {
				if len(ids) == 0 {
					continue
				}
				PlayersAttributeInfoexecData["command"] = "skills"
				for _, id := range ids {
					PlayersAttributeInfoexecData["commandArgs"] = fmt.Sprintf("%s-%s-1", id, group)
					AchievementChan <- PlayersAttributeInfoexecData
				}
				fmt.Printf("[PlayersInfo] 技能条件组 %s 符合玩家: %v\n", group, ids)
			}
		}
		if cmdType == "PlayerEquipmentInfo" {
			players := command["commandArgs"].(map[string]interface{})
			result := EvaluatePlayerEquipment(players)
			if len(result) == 0 {
				continue
			}
			PlayerEquipmentInfoexecData := map[string]interface{}{
				"steamID":  "000000",
				"nickName": "System",
				"command":  "",
				//"commandArgs": steamId + "-" + "naked" + "-" + "1",
				"commandArgs": "",
			}
			for group, ids := range result {
				if len(ids) == 0 {
					continue
				}
				PlayerEquipmentInfoexecData["command"] = "equip"
				for _, id := range ids {
					PlayerEquipmentInfoexecData["commandArgs"] = fmt.Sprintf("%s-%s-1", id, group)
					AchievementChan <- PlayerEquipmentInfoexecData
				}
				fmt.Printf("[PlayersInfo] 装备条件组 %s 符合玩家: %v\n", group, ids)
			}
		}
	}
}

var pcGroups map[string]*PlayerConditionGroup
var itemsDB map[string][]string
var eqiupCfg *EquipmentConfig

// 主入口
func PlayersInfo(regCommand *map[string][]string, PlayersInfoChan chan map[string]interface{}, AchievementChan chan map[string]interface{}, chatChan chan string, initChan chan struct{}) {
	cfg, err := execModules.NewConfig("./ini/PlayersInfo/PlayersInfo.ini")
	if err != nil {
		log.Println("[PlayersInfo-Error] 加载 PlayersInfo.ini 失败:", err)
		cfg = &execModules.Config{}
	}

	CommandRegister(cfg, regCommand)

	pcGroups = LoadPlayerCondition("./ini/PlayersInfo/PlayersCondition.ini")
	itemsDB, _ = LoadClothesItems("./db/itemsDB.db")

	eqiupCfg, err = LoadEquipmentConfig("./ini/PlayersInfo/PlayersEquiped.ini")
	if err != nil {
		log.Println("[PlayersInfo-Error] 加载 PlayersInfoEquipment.ini 失败:", err)
		eqiupCfg = &EquipmentConfig{} // 避免 nil
	}

	fmt.Println("[PlayersInfo] 读取到的物品id:")
	for c, _ := range itemsDB {
		fmt.Print(c, ",")
		//fmt.Printf("=====%s=====\n", c)
		//fmt.Println(ilist)
	}
	fmt.Println()

	go CommandHandler(PlayersInfoChan, AchievementChan, chatChan)
	close(initChan)
}
