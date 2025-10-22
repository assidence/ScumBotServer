package modules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// sequenceJson sequence dict to Json
func sequenceJson(execData *map[string]string) []byte {
	jsonByte, _ := json.Marshal(execData)
	return jsonByte
}

func CommandHandler(regstring string, commch <-chan string, execch chan string) {
	re := regexp.MustCompile(regstring)
	var jsonByte []byte
	execData := map[string]string{
		"steamID":     "0",
		"nickName":    "0",
		"command":     "0",
		"commandArgs": "0",
	}
	for line := range commch {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 2 {
			execData["steamID"] = matches[1]
			execData["nickName"] = matches[2]
			com := strings.Split(matches[3], "-")
			execData["command"] = com[0]
			//execData["command"] = matches[3]
			execData["commandArgs"] = strings.Join(com[1:], "-")
			jsonByte = sequenceJson(&execData)
			execch <- string(jsonByte)
		} else {
			fmt.Printf("[玩家聊天]%s\n", line)
		}
	}
}
