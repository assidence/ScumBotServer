package main

import (
	"ScumBotServer/modules"
	"ScumBotServer/modules/IMServer"
	"fmt"
	"os"
)

func main() {
	//found newset ChatLog
	filePath, newsetTime, err := modules.FindNewestChatLog(os.Args[1])

	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Chat log founded!%s(%s)\n", filePath, newsetTime)
	var PlayerCommand = modules.ReadCommand(filePath)
	fmt.Printf("[Success]Chat log Loaded!\n")
	addr := "0.0.0.0:20500"
	online := make(chan struct{})
	execch := make(chan string)
	go IMServer.StartHttpServer(addr, online)
	for {
		select {
		case <-online:
			break
		default:
			continue
		}
		break
	}
	go IMServer.HttpClient(addr, execch)

	commch := make(chan string)
	reg := `'(\d+):([^']+)'\s+'.*?:\s*(@[^']+)'`
	go modules.CommandHandler(reg, commch, execch)
	for line := range *PlayerCommand {
		commch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}
