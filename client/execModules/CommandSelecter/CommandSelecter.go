package CommandSelecter

import (
	"ScumBotServer/client/execModules/LogWacher"
	"fmt"
	"regexp"
)

func Selecter(steamID string, cfgCommand string, lw *LogWacher.LogWatcher) []string {
	commandPrefix := "#"
	re := regexp.MustCompile(`^\w+`)
	cmd := re.FindString(cfgCommand)
	var cfgChat []string
	switch cmd {
	case "DestroyDiDi":
		if lw.Vehicles["BPC_Dirtbike"] == nil {
			cfgChat = append(cfgChat, fmt.Sprintf("找不到%s车辆类型的id列表", "BPC_Dirtbike"))

		}
		for _, vehicleID := range lw.Vehicles["BPC_Dirtbike"] {
			cfgChat = append(cfgChat, fmt.Sprintf("%sDestroyVehicle %s", commandPrefix, vehicleID))
		}
	case "SpawnItem":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "ChangeCurrencyBalance":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "SpawnVehicle":
		PLocationX := lw.Players[steamID].LocationX
		PLocationY := lw.Players[steamID].LocationY
		PLocationZ := lw.Players[steamID].LocationZ
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, PLocationX, PLocationY, PLocationZ))
	case "ListPlayers":
		cfgChat = append(cfgChat, commandPrefix+cfgCommand)
	case "SetFakeName":
		var nickName string
		if lw.Players[steamID].Prefix != "" {
			nickName = fmt.Sprintf("-★%s★-%s", lw.Players[steamID].Prefix, lw.Players[steamID].Name)
		} else {
			fmt.Println("CommandSelecter:")
			//fmt.Println(lw.Players[steamID])
			nickName = lw.Players[steamID].Name
		}
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID, nickName))
	default:
		fmt.Println("[ERROR-CommandSelecter]->Error:无法匹配命令 ", cmd)
	}
	return cfgChat
}
