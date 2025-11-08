package main

import (
	"encoding/json"
	"fmt"
	"net"
)

// sequenceJson sequence dict to Json
func sequenceJson(execData *map[string]string) []byte {
	jsonByte, _ := json.Marshal(execData)
	return jsonByte
}

func commandSender(conn net.Conn, sendChan chan []byte, exitChan chan struct{}) {
	for {
		select {
		case <-exitChan:
			fmt.Println("[commandSender] 收到退出信号，停止发送")
			return

		case msg, ok := <-sendChan:
			if !ok {
				fmt.Println("[commandSender] sendChan 已关闭，退出")
				return
			}

			_, err := conn.Write(append(msg, '\n'))
			if err != nil {
				fmt.Println("[commandSender] 写入失败:", err)
				// 通常是网络断开或服务器关闭连接
				return
			}
		}
	}
}
