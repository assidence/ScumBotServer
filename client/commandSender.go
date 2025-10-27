package main

import (
	"encoding/json"
	"net"
)

// sequenceJson sequence dict to Json
func sequenceJson(execData *map[string]string) []byte {
	jsonByte, _ := json.Marshal(execData)
	return jsonByte
}

func commandSender(conn net.Conn, sendChan chan []byte) {
	for msg := range sendChan {
		//fmt.Println("[广播]:", string(msg))
		conn.Write(append(msg, '\n'))
	}
}
