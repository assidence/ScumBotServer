package modules

import (
	"fmt"
	"regexp"
)

func commandSwitcher(player string, nickName string, command string) string {
	fmt.Printf("[玩家命令] 玩家：%s | 命令：%s\n", player, command)
	switch command {
	case "@滴滴车":
		fmt.Println("")
	case "@新手礼包":
		fmt.Println("")
	default:
		fmt.Println("[玩家命令] 未知命令:", command)
	}
	return ``
}

func CommandHandler(regstring string, commch <-chan string, execch chan string) {
	re := regexp.MustCompile(regstring)
	for line := range commch {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 2 {
			steamID := matches[1]
			nickName := matches[2]
			command := matches[3]
			ExecutableCommand := commandSwitcher(steamID, nickName, command)
			execch <- ExecutableCommand
		} else {
			fmt.Printf("[玩家聊天]%s\n", line)
		}
	}
}
