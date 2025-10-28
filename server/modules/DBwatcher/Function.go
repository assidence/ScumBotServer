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
	dbPath := strings.Replace(os.Args[2], "Logs", "SCUM.db", 1)
	fmt.Println("[DBWatcher] Database path:", dbPath)

	db, err := OpenDBRO(dbPath)
	if err != nil {
		log.Fatal("无法打开数据库:", err)
	}
	defer db.Close()

	for msg := range DBWincomech {
		fmt.Printf("[DBWatcher] 收到数据库查询任务: %s\n", msg["type"])

		if strings.Contains(msg["type"].(string), "onlinePlayers") {
			PlayerList := strings.Split(msg["SteamIdList"].(string), "-")
			result := GetNakedPlayers(db, PlayerList)
			if result == nil || len(result) == 0 {
				continue
			}
			for _, steamId := range result {
				//fmt.Println("识别到裸体玩家:", steamId)
				execData := map[string]string{
					"steamID":     "000000",
					"nickName":    "System",
					"command":     "equip",
					"commandArgs": steamId + "-" + "naked" + "-" + "1",
				}
				jsonBytes, _ := json.Marshal(execData)
				execch <- string(jsonBytes)
			}
		}

	}
}
