package StatusMonitor

import (
	"ScumBotServer/client/execModules/Public"
	"encoding/json"
	"strings"
	"time"
)

// sequenceJson sequence dict to Json
func sequenceJson(execData *map[string]interface{}) []byte {
	jsonByte, _ := json.Marshal(execData)
	return jsonByte
}

func SendOnline(sendChannel chan []byte) {
	var execData = map[string]interface{}{
		"type":        "onlinePlayers",
		"SteamIdList": "",
	}
	onlinePlayer := Public.LogWatcherInterface.GetPlayers()
	//fmt.Println("GetPlayers:", onlinePlayer)
	PlayerList := []string{}
	for steamId, _ := range onlinePlayer {
		PlayerList = append(PlayerList, steamId)
	}
	if len(PlayerList) == 0 {
		return
	}
	sendList := strings.Join(PlayerList, "-")
	//fmt.Println("sendlist:", sendList)
	execData["SteamIdList"] = sendList
	//fmt.Println("execData:", execData)
	jsonByte := sequenceJson(&execData)
	//fmt.Println("[StatusMonitor] Sending onlinePlayers")
	sendChannel <- jsonByte
	//fmt.Println("[StatusMonitor] Echo data success!")

	//让服务器停止无限的寻找
	//execData["SteamIdList"] = ""
	//jsonByte = sequenceJson(&execData)
	//sendChannel <- jsonByte
	//fmt.Println("[StatusMonitor-Module] 已广播在线玩家列表:", string(jsonByte))
}

//var lw = Public.LogWatcher

func StatusMonitor(sendChannel chan []byte, initChan chan struct{}) {
	close(initChan)
	for {
		SendOnline(sendChannel)
		time.Sleep(10 * time.Second)
	}
}
