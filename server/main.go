package main

import (
	modules2 "ScumBotServer/server/modules"
	IMServer2 "ScumBotServer/server/modules/IMServer"
	"fmt"
	"os"
)

func main() {
	//found newset ChatLog
	filePath, newsetTime, err := modules2.FindNewestChatLog(os.Args[2])

	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Chat log founded!%s(%s)\n", filePath, newsetTime)
	var PlayerCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]Chat log Loaded!\n")
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

	commch := make(chan string)
	reg := `'(\d+):([^']+)'\s+'.*?:\s*(@[^']+)'`
	go modules2.CommandHandler(reg, commch, execch)
	for line := range *PlayerCommand {
		commch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}
