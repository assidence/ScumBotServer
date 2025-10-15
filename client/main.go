package main

import (
	"ScumBotServer/client/execModules"
	"net"
	"regexp"
)

func main() {
	address := "0.0.0.0:20500"
	var execCommand = make(chan map[string]interface{})
	re := regexp.MustCompile(`\{[^}]*\}`)
	go commandExecuter(execCommand)
	go execModules.FocusWindows("SCUM  ")
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
