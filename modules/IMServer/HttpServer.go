package IMServer

import (
	"bufio"
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

// Client Save online client
type Client struct {
	conn     net.Conn
	username string
}

var clients = make(map[net.Conn]Client)
var mu sync.Mutex

func broadcast(sender *Client, msg string) {
	mu.Lock()
	defer mu.Unlock()
	for _, c := range clients {
		if c.conn != sender.conn {
			c.conn.Write([]byte(fmt.Sprintf("[%s]: %s\n", sender.username, msg)))
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.Write([]byte("Welcome! Please provide your token:\n"))

	reader := bufio.NewReader(conn)
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	username, ok := validTokens[token]
	if !ok {
		conn.Write([]byte("Invalid token. Connection closed.\n"))
		return
	}

	client := &Client{conn: conn, username: username}
	mu.Lock()
	clients[client.conn] = *client
	mu.Unlock()

	conn.Write([]byte("Authentication successful! You can start chatting.\n"))
	fmt.Printf("[Network] %s connected from %s\n", username, conn.RemoteAddr())

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[Network] %s disconnected.\n", username)
			break
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		fmt.Printf("[Network][%s]: %s\n", username, msg)
		broadcast(client, msg)
	}
	mu.Lock()
	delete(clients, conn)
	mu.Unlock()
}

func StartHttpServer(address string, online chan struct{}) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Printf("[Network] Server listening on %s ...\n", address)
	close(online)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("[Network] Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}
