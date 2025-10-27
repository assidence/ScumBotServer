package StatusMonitor

import (
	"ScumBotServer/client/execModules/LogWacher"
	"ScumBotServer/client/execModules/Prefix"
	"encoding/json"
	"strings"
	"time"
)

// sequenceJson sequence dict to Json
func sequenceJson(execData *map[string]interface{}) []byte {
	jsonByte, _ := json.Marshal(execData)
	return jsonByte
}

func SendOnline(lw *LogWacher.LogWatcher, sendChannel chan []byte) {
	var execData = map[string]interface{}{
		"type":        "onlinePlayers",
		"SteamIdList": "",
	}
	onlinePlayer := lw.GetPlayers()
	//fmt.Println("GetPlayers:", onlinePlayer)
	PlayerList := []string{}
	for steamId, _ := range onlinePlayer {
		PlayerList = append(PlayerList, steamId)
	}
	sendList := strings.Join(PlayerList, "-")
	//fmt.Println("sendlist:", sendList)
	execData["SteamIdList"] = sendList
	//fmt.Println("execData:", execData)
	jsonByte := sequenceJson(&execData)
	sendChannel <- jsonByte
	//fmt.Println("[StatusMonitor-Module] 已广播在线玩家列表:", string(jsonByte))
}

func StatusMonitor(sendChannel chan []byte, lw *LogWacher.LogWatcher, tm *Prefix.TitleManager, initChan chan struct{}) {
	close(initChan)
	for {
		SendOnline(lw, sendChannel)
		time.Sleep(10 * time.Second)
	}
}
