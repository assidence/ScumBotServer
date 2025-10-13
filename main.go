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
	addr := "127.0.0.1:20500"
	online := make(chan struct{})
	ch := make(chan string)
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
	go IMServer.HttpClient(addr, ch)
	for line := range *PlayerCommand {
		fmt.Println("[聊天]", line.Text)
		ch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}
