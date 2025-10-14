package main

import (
	"ScumBotServer/client/execModules"
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"time"
)

func HttpClient(address string) net.Conn {
	for {
		fmt.Println("[Network] Client Initiating")
		conn, err := net.Dial("tcp", address)
		if err != nil {
			fmt.Println("[Error] connServer is unreachable:" + err.Error())
			time.Sleep(3 * time.Second)
		} else {
			conn.Write([]byte("token123\n"))
			fmt.Println("[Network] ClientOP is now online")
			return conn
		}
	}
}

// commandReader unmarshal JSON to key:value map
func commandReader(re *regexp.Regexp, conn net.Conn, execCommand chan map[string]interface{}, networkSignal chan struct{}) {
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n') //each JSON end with \n
		if err != nil {
			fmt.Println("Disconnect from server:" + err.Error())
			//close(execCommand)
			close(networkSignal)
			return
		}
		match := re.FindString(line)
		var msg map[string]interface{}
		err = json.Unmarshal([]byte(match), &msg)
		if err != nil {
			//fmt.Println(line)
			fmt.Println(line)
			continue
		}
		//fmt.Println("[Info] msg[" + msg["command"].(string) + "]")

		execCommand <- msg
	}
}

// commandExecuter aka commandExecuter
func commandExecuter(execCommand chan map[string]interface{}) {
	var exec = ""
	for command := range execCommand {
		switch command["command"].(string) {
		case "@滴滴车":
			exec = "#SpawnVehicle " + "BP_Dirtbike_E " + "1 " + "Location " + command["steamID"].(string) + " " + "Modifier " + "MinimalFunctional"
			fmt.Println(exec)
			fmt.Printf("[Execute] command: %s from [%s]%s\n", command["command"].(string), command["nickName"].(string), command["steamID"].(string))
		default:
			fmt.Printf("[Error]Invalidate command: %s from [%s]%s\n", command["command"].(string), command["nickName"].(string), command["steamID"].(string))
		}
	}
}

func main() {
	address := "0.0.0.0:20500"
	var execCommand = make(chan map[string]interface{})
	re := regexp.MustCompile(`\{[^}]*\}`)
	go commandExecuter(execCommand)
	go execModules.FocusWindows("SCUM")
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
