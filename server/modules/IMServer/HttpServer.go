package IMServer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
)

// Token list
var validTokens = map[string]string{
	"token123": "GameOP0",
	"token456": "GameOP1",
	"token789": "Broadcaster",
}

type Client struct {
	conn     net.Conn
	username string
}

type Message struct {
	From    string `json:"from"`
	Content string `json:"content"`
}

var (
	clients = make(map[net.Conn]Client)
	mu      sync.Mutex
)

// StartServer 启动 IM 服务器
func StartServer(address string, incoming chan Message, online chan struct{}) error {
	ln, err := net.Listen("tcp4", address)
	if err != nil {
		return err
	}
	fmt.Println("[IMServer] Listening on", address)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println("[IMServer] accept error:", err)
				continue
			}
			go handleClient(conn, incoming)
		}
	}()
	close(online)
	return nil
}

// 广播消息给其他客户端，保持消息原样（JSON 格式）
func broadcast(sender *Client, msg Message) {
	mu.Lock()
	defer mu.Unlock()
	//fmt.Println("已进入Broadcast函数")
	//fmt.Println("msg的内容：", msg)
	jsonBytes, _ := json.Marshal(msg)
	//fmt.Println("jsonBtyes", jsonBytes)
	//fmt.Println("Broadcast函数收到的json:", json.Unmarshal(jsonBytes, sender))
	for _, c := range clients {
		//fmt.Println("broadcast客户端比较：", c.conn, sender.conn)
		if c.conn != sender.conn {
			//fmt.Println("检测到其他客户端，广播数据：", string(jsonBytes), '\n')
			_, err := c.conn.Write(append(jsonBytes, '\n'))
			if err != nil {
				fmt.Println("[IMServer] 广播失败:", err)
			}
		}
	}
}

// 处理单个客户端
func handleClient(conn net.Conn, incoming chan Message) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	token, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("[IMServer] failed to read token:", err)
		return
	}

	token = string(token[:len(token)-1])
	username, ok := validTokens[token]
	if !ok {
		conn.Write([]byte("Invalid token\n"))
		return
	}

	client := Client{conn: conn, username: username}

	mu.Lock()
	clients[conn] = client
	mu.Unlock()

	fmt.Printf("[IMServer] %s connected\n", username)
	conn.Write([]byte("Welcome " + username + "!\n"))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("[IMServer]", username, "disconnected:", err)
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			return
		}

		line = strings.TrimSpace(line) // 只去掉前后空白，不截断 JSON
		if line == "" {
			continue
		}

		//fmt.Println("[IMServer] 收到消息原文:", line)

		var msg Message
		err = json.Unmarshal([]byte(line), &msg)
		if err != nil {
			fmt.Println("[IMServer] 解析失败:", err)
			continue
		}

		msg.From = username
		msg.Content = line

		// 发给外部通道
		incoming <- msg

		// 广播给其他客户端
		broadcast(&client, msg)
	}

}
