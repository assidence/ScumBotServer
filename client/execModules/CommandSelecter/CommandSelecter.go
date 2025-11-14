package CommandSelecter

import (
	"ScumBotServer/client/execModules/Public"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Selecter(steamID string, cfgCommand string) []string {
	commandPrefix := "#"
	re := regexp.MustCompile(`^\w+`)
	cmd := re.FindString(cfgCommand)
	var cfgChat []string
	switch cmd {
	case "DestroyDiDi":
		if Public.LogWatcherInterface.Vehicles["BPC_Dirtbike"] == nil {
			cfgChat = append(cfgChat, fmt.Sprintf("找不到%s车辆类型的id列表", "BPC_Dirtbike"))

		}
		for _, vehicleID := range Public.LogWatcherInterface.Vehicles["BPC_Dirtbike"] {
			cfgChat = append(cfgChat, fmt.Sprintf("%sDestroyVehicle %s", commandPrefix, vehicleID))
		}
	case "SpawnItem":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "ChangeCurrencyBalance":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "SpawnVehicle":
		PLocationX := Public.LogWatcherInterface.Players[steamID].LocationX
		PLocationY := Public.LogWatcherInterface.Players[steamID].LocationY
		PLocationZ := Public.LogWatcherInterface.Players[steamID].LocationZ
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, PLocationX, PLocationY, PLocationZ))
	case "ListPlayers":
		cfgChat = append(cfgChat, commandPrefix+cfgCommand)
	case "SetFakeName":
		var nickName string
		if Public.LogWatcherInterface.Players[steamID].Prefix != "" {
			nickName = fmt.Sprintf("-★%s★-%s", Public.LogWatcherInterface.Players[steamID].Prefix, Public.LogWatcherInterface.Players[steamID].Name)
		} else {
			fmt.Println("CommandSelecter:")
			//fmt.Println(lw.Players[steamID])
			nickName = Public.LogWatcherInterface.Players[steamID].Name
		}
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID, nickName))
	case "Teleport":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "AchievementRewardItem":
		OnlinePlayers := Public.LogWatcherInterface.GetPlayers()
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
		OnlinePlayers := Public.LogWatcherInterface.GetPlayers()
		for steamid, _ := range OnlinePlayers {
			cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(output, steamid))
		}
	case "SpawnInventoryFullOf":
		cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf(cfgCommand, steamID))
	case "SpawnRandomItem":
		parts := strings.Split(cfgCommand, " ")
		// 跳过第一个 "SpawnRandomItem"
		itemsPart := parts[1]
		countStr := parts[len(parts)-1]
		count, _ := strconv.Atoi(countStr)
		// 分割物品名
		items := strings.Split(itemsPart, "::")
		// 随机选择
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < count; i++ {
			cfgChat = append(cfgChat, commandPrefix+fmt.Sprintf("SpawnItem %s 1 Location %s", items[rand.Intn(len(items))], steamID))
		}
	case "Sun":
		cfgChat = append(cfgChat, commandPrefix+"SetWeather 0")
	case "Rain":
		cfgChat = append(cfgChat, commandPrefix+"SetWeather 1")
	case "RewardEcho":
		EchoString := strings.Split(cfgCommand, " ")
		cfgChat = append(cfgChat, fmt.Sprintf(strings.Join(EchoString[1:], " "), Public.LogWatcherInterface.Players[steamID].Name))
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
