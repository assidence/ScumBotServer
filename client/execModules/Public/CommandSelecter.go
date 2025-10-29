package Public

import (
	"fmt"
	"regexp"
)

func Selecter(steamID string, cfgCommand string) []string {
	if GlobalLogWatcher == nil {
		fmt.Println("[CommandSelecter-Panic] LogWatcher is nil")
		return nil
	}
	if GlobalTitleManager == nil {
		fmt.Println("[CommandSelecter-Panic] TitleManager is null")
		return nil
	}
	commandPrefix := "#"
	re := regexp.MustCompile(`^\w+`)
	cmd := re.FindString(cfgCommand)
	var cfgChat []string
	switch cmd {
	case "DestroyDiDi":
		if GlobalLogWatcher.Vehicles["BPC_Dirtbike"] == nil {
			cfgChat = append(cfgChat, fmt.Sprintf("找不到%s车辆类型的id列表", "BPC_Dirtbike"))

		}
		for _, vehicleID := range GlobalLogWatcher.Vehicles["BPC_Dirtbike"] {
			cfgChat = append(cfgChat, fmt.Sprintf("%sDestroyVehicle %s", commandPrefix, vehicleID))
		}
	case "SpawnItem":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "ChangeCurrencyBalance":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "SpawnVehicle":
		PLocationX := GlobalLogWatcher.Players[steamID].LocationX
		PLocationY := GlobalLogWatcher.Players[steamID].LocationY
		PLocationZ := GlobalLogWatcher.Players[steamID].LocationZ
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, PLocationX, PLocationY, PLocationZ))
	case "ListPlayers":
		cfgChat = append(cfgChat, commandPrefix+cfgCommand)
	case "SetFakeName":
		var nickName string
		if GlobalLogWatcher.Players[steamID].Prefix != "" {
			nickName = fmt.Sprintf("-★%s★-%s", GlobalLogWatcher.Players[steamID].Prefix, GlobalLogWatcher.Players[steamID].Name)
		} else {
			fmt.Println("CommandSelecter:")
			//fmt.Println(lw.Players[steamID])
			nickName = GlobalLogWatcher.Players[steamID].Name
		}
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID, nickName))
	case "Teleport":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "AchievementRewardItem":
		OnlinePlayers := GlobalLogWatcher.GetPlayers()
		for steamid, _ := range OnlinePlayers {
			if steamid == "" {
				continue
			}
			playerTitle, _ := GlobalTitleManager.PrefixGetActiveTitle(steamid)
			if playerTitle == "" {
				continue
			}
			for _, achievement := range *GlobalAchievements {

				if achievement.Name != playerTitle {
					continue
				}
				//fmt.Println("[DEBUG]RewardCommandLines:", achievement.RewardCommandLines)
				for _, line := range achievement.RewardCommandLines {
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
		OnlinePlayers := GlobalLogWatcher.GetPlayers()
		for steamid, _ := range OnlinePlayers {
			cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(output, steamid))
		}
	default:
		//fmt.Println("[ERROR-CommandSelecter]->Error:无法匹配命令 ", cmd)
		cfgChat = append(cfgChat, cfgCommand)
	}
	return cfgChat
}
