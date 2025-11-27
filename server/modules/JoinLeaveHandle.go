package modules

import (
	"fmt"
	"regexp"
)

func JoinLeaveHandler(regstring string, commch <-chan string, execch chan string) {
	re := regexp.MustCompile(regstring)
	var jsonByte []byte
	execData := map[string]string{
		"steamID":     "000000",
		"nickName":    "System",
		"command":     "0",
		"commandArgs": "0",
	}
	for line := range commch {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 2 {
			/*
				for _, i := range matches {
					fmt.Println("JoinEventMatches:", i)
				}

			*/
			execData["command"] = "@" + matches[6]
			//execData["command"] = matches[3]
			execData["commandArgs"] = matches[4]

			if matches[6] == "in" {
				execData2 := map[string]string{
					"steamID":     matches[3],
					"nickName":    matches[4],
					"command":     "checkIn",
					"commandArgs": "0",
				}
				jsonByte = sequenceJson(&execData2)
				execch <- string(jsonByte)
			}

			jsonByte = sequenceJson(&execData)
			execch <- string(jsonByte)
		} else {
			fmt.Println("[JoinLeave-Error]未匹配到行为日志:")
			fmt.Println(line)
		}
	}
}
