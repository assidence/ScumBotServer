package DBwatcher

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Start 启动 DBwatcher 模块（被动模式）
func Start(execch chan string, DBWincomech chan map[string]interface{}) {
	for msg := range DBWincomech {
		//fmt.Printf("[DBWatcher] 收到数据库查询任务: %s\n", msg["type"])
		dbPath := strings.Replace(os.Args[2], "Logs", "SCUM.db", 1)
		//fmt.Println("[DBWatcher] Database path:", dbPath)

		db, err := OpenDBRO(dbPath)
		if err != nil {
			log.Fatal("无法打开数据库:", err)
		}
		//defer db.Close()

		if strings.Contains(msg["type"].(string), "onlinePlayers") {
			PlayerList := strings.Split(msg["SteamIdList"].(string), "-")
			PlayersAttributeInfo := GetPlayerFullInfoByList(db, PlayerList)
			if len(PlayersAttributeInfo) > 0 {
				// 封装要发送的数据
				execData := map[string]interface{}{
					"steamID":     "000000",
					"nickName":    "System",
					"command":     "PlayersAttributeInfo",
					"commandArgs": PlayersAttributeInfo,
				}

				jsonBytes, err := json.Marshal(execData)
				if err != nil {
					log.Println("序列化玩家信息失败:", err)
				} else {
					// 发送到客户端
					execch <- string(jsonBytes)
					fmt.Printf("[DBWatcher] 已发送玩家数值数据，发送数据包大小: %d bytes\n", len(jsonBytes))
				}
			} else {
				fmt.Println("[DBWatcher-Error] PlayersAttributeInfo 结果为空")
			}
			PlayerEquipmentInfo, _ := QueryEquippedItemsBySteamIDs(db, PlayerList)
			if len(PlayerEquipmentInfo) > 0 {
				execData := map[string]interface{}{
					"steamID":     "000000",
					"nickName":    "System",
					"command":     "PlayerEquipmentInfo",
					"commandArgs": PlayerEquipmentInfo,
				}
				jsonBytes, err := json.Marshal(execData)
				if err != nil {
					log.Println("序列化玩家信息失败:", err)
				} else {
					// 发送到客户端
					execch <- string(jsonBytes)
					fmt.Printf("[DBWatcher] 已发送玩家装备数据，发送数据包大小: %d bytes\n", len(jsonBytes))
				}
			} else {
				fmt.Println("[DBWatcher-Error] PlayerEquipmentInfo 结果为空", PlayerEquipmentInfo)
			}

		}
		db.Close()
		//fmt.Println("[DBWatcher] 数据库查询任务已完成")
	}
}
