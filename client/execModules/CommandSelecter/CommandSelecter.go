package CommandSelecter

import (
	"ScumBotServer/client/execModules/Public"
	"fmt"
	"regexp"
)

func Selecter(steamID string, cfgCommand string) []string {
	if Public.GlobalLogWatcher == nil {
		fmt.Println("[CommandSelecter-Panic] LogWatcher is nil")
		return nil
	}
	commandPrefix := "#"
	re := regexp.MustCompile(`^\w+`)
	cmd := re.FindString(cfgCommand)
	var cfgChat []string
	switch cmd {
	case "DestroyDiDi":
		if Public.GlobalLogWatcher.Vehicles["BPC_Dirtbike"] == nil {
			cfgChat = append(cfgChat, fmt.Sprintf("找不到%s车辆类型的id列表", "BPC_Dirtbike"))

		}
		for _, vehicleID := range Public.GlobalLogWatcher.Vehicles["BPC_Dirtbike"] {
			cfgChat = append(cfgChat, fmt.Sprintf("%sDestroyVehicle %s", commandPrefix, vehicleID))
		}
	case "SpawnItem":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "ChangeCurrencyBalance":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "SpawnVehicle":
		PLocationX := Public.GlobalLogWatcher.Players[steamID].LocationX
		PLocationY := Public.GlobalLogWatcher.Players[steamID].LocationY
		PLocationZ := Public.GlobalLogWatcher.Players[steamID].LocationZ
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, PLocationX, PLocationY, PLocationZ))
	case "ListPlayers":
		cfgChat = append(cfgChat, commandPrefix+cfgCommand)
	case "SetFakeName":
		var nickName string
		if Public.GlobalLogWatcher.Players[steamID].Prefix != "" {
			nickName = fmt.Sprintf("-★%s★-%s", Public.GlobalLogWatcher.Players[steamID].Prefix, Public.GlobalLogWatcher.Players[steamID].Name)
		} else {
			fmt.Println("CommandSelecter:")
			//fmt.Println(lw.Players[steamID])
			nickName = Public.GlobalLogWatcher.Players[steamID].Name
		}
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID, nickName))
	case "Teleport":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "AchievementRewardItem":
		OnlinePlayers := Public.GlobalLogWatcher.GetPlayers()
		for steamid, _ := range OnlinePlayers {
			if steamid == "" {
				continue
			}
			playerTitle, _ := Public.TitleInterface.PrefixGetActiveTitle(steamid)
			if playerTitle == "" {
				continue
			}
			if RewardCommandLines, ok := Public.AchievementInterface.AchievementReward[playerTitle]; ok {
				for _, line := range RewardCommandLines {
					cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(line, steamid))
				}
			}
		}
	case "ShutdownServer":
		cfgChat = append(cfgChat, commandPrefix+"ShutdownServer pretty please")
	case "AddOnlineCurrency":
		// 匹配 AddOnlineCurrency 后面跟的数字
		re := regexp.MustCompile(`AddOnlineCurrency (\d+)`)
		// 替换为 ChangeCurrencyBalance Normal 数字 %s
		output := re.ReplaceAllString(cfgCommand, "ChangeCurrencyBalance Normal $1 %s")
		OnlinePlayers := Public.GlobalLogWatcher.GetPlayers()
		for steamid, _ := range OnlinePlayers {
			cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(output, steamid))
		}
	default:
		//fmt.Println("[ERROR-CommandSelecter]->Error:无法匹配命令 ", cmd)
		cfgChat = append(cfgChat, cfgCommand)
	}
	return cfgChat
}

func InitPublicSelecter(initChan chan struct{}) {
	Public.CommandSelecterInterface.Selecter = Selecter
	close(initChan)
}
