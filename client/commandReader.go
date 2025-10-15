package main

import (
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
