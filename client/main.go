package main

import (
	"ScumBotServer/client/execModules"
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
)

func main() {
	address := ""
	reader := bufio.NewScanner(os.Stdin)
	fmt.Print("请输入ScumBot-服务端的地址: ")
	reader.Scan() // 阻塞直到用户输入回车
	address = reader.Text()
	if address == "" {
		fmt.Println("用户直接按了回车，使用默认值")
		address = "0.0.0.0:20500"
	}
	var execCommand = make(chan map[string]interface{})
	re := regexp.MustCompile(`\{[^}]*\}`)
	go commandExecuter(execCommand)
	go execModules.FocusWindows("SCUM  ")
	//go execModules.AutoReConnect()
	for {
		NetworkSignal := make(chan struct{})
		conn := HttpClient(address)
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				panic(err)
			}
		}(conn)
		go commandReader(re, conn, execCommand, NetworkSignal)
		<-NetworkSignal
	}
}
