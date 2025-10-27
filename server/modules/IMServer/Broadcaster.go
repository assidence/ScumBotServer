package IMServer

import (
	"fmt"
	"net"
	"strings"
)

func HttpClient(address string, msgChan chan string) {
	fmt.Println("[Network] Broadcaster Initiating")
	conn, err := net.Dial("tcp", address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 发送 token 登录
	conn.Write([]byte("token789\n"))
	fmt.Println("[Network] Broadcaster is now online")

	for l := range msgChan {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}

		// 如果 l 已经是 JSON 字符串，直接发送，不要再 Marshal
		_, err := conn.Write([]byte(l + "\n"))
		if err != nil {
			fmt.Println("[Network] Send failed:", err)
			break
		}

		fmt.Println("[Network] Sent:", l)
	}
}
