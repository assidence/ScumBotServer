package DBwatcher

import (
	"encoding/json"
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
			//fmt.Println("[DBWatcher] 玩家列表:", PlayerList)
			result := GetNakedPlayers(db, PlayerList)
			if result != nil && len(result) > 0 {
				for _, steamId := range result {
					//fmt.Println("识别到裸体玩家:", steamId)
					execData := map[string]string{
						"steamID":     "000000",
						"nickName":    "System",
						"command":     "equip",
						"commandArgs": steamId + "-" + "naked" + "-" + "1",
					}
					jsonBytes, _ := json.Marshal(execData)
					//fmt.Println("[DBWatcher] 准备发送裸体玩家查询结果")
					execch <- string(jsonBytes)
					//fmt.Println("[DBWatcher] 裸体玩家查询结果发送完成")
				}
			}
			result = GetStrongPlayers(db, PlayerList)
			if result != nil && len(result) > 0 {
				for _, steamId := range result {
					//fmt.Println("识别到头尖尖的玩家:", steamId)
					execData := map[string]string{
						"steamID":     "000000",
						"nickName":    "System",
						"command":     "skills",
						"commandArgs": steamId + "-" + "fit" + "-" + "1",
					}
					jsonBytes, _ := json.Marshal(execData)
					//fmt.Println("[DBWatcher] 准备发送头尖尖玩家查询结果")
					execch <- string(jsonBytes)
					//fmt.Println("[DBWatcher] 头尖尖玩家查询结果发送完成")
				}
			}
		}
		db.Close()
		//fmt.Println("[DBWatcher] 数据库查询任务已完成")
	}
}
