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
	conn.Write([]byte("token789\n"))
	fmt.Println("[Network] Broadcaster is now online")
	for l := range msgChan {
		l = strings.TrimSpace(l)
		conn.Write([]byte(l + "\n"))
	}
}
