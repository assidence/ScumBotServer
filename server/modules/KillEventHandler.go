package modules

import (
	"fmt"
	"regexp"
)

type KillEvent struct {
	victimSteamID string
	killerSteamID string
	weaponName    string
	weaponType    string
}

func parseKillEvent(content string) *KillEvent {
	re := regexp.MustCompile(`Died: [^()]+\s\((\d+)\), Killer: [^()]+\s\((\d+)\) Weapon: ([^\[]+)\s\[(\w+)\]`)
	matches := re.FindStringSubmatch(content)
	if len(matches) != 5 {
		fmt.Println("[KillEvent-Error]Insufficient KillEvent matched:", len(matches))
		return nil
	}
	return &KillEvent{
		victimSteamID: matches[1],
		killerSteamID: matches[2],
		weaponName:    matches[3],
		weaponType:    matches[4],
	}
}

func KillHandler(regstring string, commch <-chan string, execch chan string) {
	re := regexp.MustCompile(regstring)
	execData := map[string]string{
		"steamID":     "000000",
		"nickName":    "System",
		"command":     "0",
		"commandArgs": "0",
	}
	var jsonByte []byte
	for line := range commch {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 0 {
			fmt.Println("[KillEvent-Error]Insufficient init elements matched:", len(matches))
			continue
		}

		result := parseKillEvent(line)
		if result == nil {
			continue
		}
		//send Killer event
		execData["command"] = "Killer"
		execData["commandArgs"] = fmt.Sprintf("%s-%s-1", result.killerSteamID, result.weaponName)
		jsonByte = sequenceJson(&execData)
		execch <- string(jsonByte)
		//send Died event
		execData["command"] = "Died"
		execData["commandArgs"] = fmt.Sprintf("%s-%s-1", result.victimSteamID, result.weaponType)
		jsonByte = sequenceJson(&execData)
		execch <- string(jsonByte)
		//fmt.Println("[KillEvent] Kill Event Send!")
	}
}
