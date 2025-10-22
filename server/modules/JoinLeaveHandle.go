package modules

import (
	"fmt"
	"regexp"
)

func JoinLeaveHandler(regstring string, commch <-chan string, execch chan string) {
	re := regexp.MustCompile(regstring)
	var jsonByte []byte
	execData := map[string]string{
		"steamID":     "0",
		"nickName":    "System",
		"command":     "0",
		"commandArgs": "0",
	}
	for line := range commch {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 2 {
			execData["command"] = "@" + matches[6]
			//execData["command"] = matches[3]
			execData["commandArgs"] = matches[4]
			jsonByte = sequenceJson(&execData)
			execch <- string(jsonByte)
		} else {
			fmt.Printf("[玩家聊天]%s\n", line)
		}
	}
}
