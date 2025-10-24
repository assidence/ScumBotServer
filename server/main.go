package main

import (
	modules2 "ScumBotServer/server/modules"
	IMServer2 "ScumBotServer/server/modules/IMServer"
	Utf16tail "ScumBotServer/server/modules/tail"
	"fmt"
	"os"
)

func PlayerCommandSender(PlayerCommand *chan *Utf16tail.Line, commch chan string) {
	for line := range *PlayerCommand {
		commch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

func LoginCommandSender(LoginCommand *chan *Utf16tail.Line, commch chan string) {
	for line := range *LoginCommand {
		commch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

func EconomyCommandSender(EconomyCommand *chan *Utf16tail.Line, ecoch chan string) {
	for line := range *EconomyCommand {
		ecoch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

func main() {
	//found newset ChatLog
	filePath, newsetTime, err := modules2.FindNewestChatLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Chat log founded!%s(%s)\n", filePath, newsetTime)
	var PlayerCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]Chat log Loaded!\n")

	//found newset LoginLog
	filePath, newsetTime, err = modules2.FindNewestLoginLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]login log founded!%s(%s)\n", filePath, newsetTime)
	var LoginCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]login log Loaded!\n")

	//found newset EconomyLog
	filePath, newsetTime, err = modules2.FindNewestEconomyLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Economy log founded!%s(%s)\n", filePath, newsetTime)
	var EconomyCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]Economy log Loaded!\n")

	addr := fmt.Sprintf("0.0.0.0:%s", os.Args[1])
	online := make(chan struct{})
	execch := make(chan string)
	go IMServer2.StartHttpServer(addr, online)
	for {
		select {
		case <-online:
			break
		default:
			continue
		}
		break
	}
	go IMServer2.HttpClient(addr, execch)
	//玩家聊天框
	commch := make(chan string)
	reg := `'(\d+):([^']+)'\s+'.*?:\s*(@[^']+)'`
	go PlayerCommandSender(PlayerCommand, commch)
	go modules2.CommandHandler(reg, commch, execch)
	//玩家进入退出
	logch := make(chan string)
	reg = `^(\d{4}\.\d{2}\.\d{2}-\d{2}\.\d{2}\.\d{2}): '([\d\.]+) (\d+):([^()']+)\((\d+)\)' logged (in|out) at: X=([-\d\.]+) Y=([-\d\.]+) Z=([-\d\.]+)(?: \(as drone\))?$`
	go LoginCommandSender(LoginCommand, logch)
	go modules2.JoinLeaveHandler(reg, logch, execch)
	//玩家经济行为
	ecoch := make(chan string)
	//reg = `^(\d{4}\.\d{2}\.\d{2}-\d{2}\.\d{2}\.\d{2}): '([\d\.]+) (\d+):([^()']+)\((\d+)\)' logged (in|out) at: X=([-\d\.]+) Y=([-\d\.]+) Z=([-\d\.]+)(?: \(as drone\))?$`
	go EconomyCommandSender(EconomyCommand, ecoch)
	go modules2.EconomyHandler(ecoch, execch)

	select {}
}
