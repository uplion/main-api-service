package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	clients = make(map[string]chan string)
	mu      sync.Mutex
)

func wsHandler(c *gin.Context) {
	id := c.Query("id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set websocket upgrade: " + err.Error()})
		return
	}
	defer conn.Close()

	if id == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("id is required"))
		return
	}

	for {
		_, msgBytes, err := conn.ReadMessage()
		msg := strings.TrimSpace(string(msgBytes))

		if err != nil {
			break
		}

		// If message is "close", close the SSE connection
		if msg == "close" {
			closeSSE(id)
			break
		}

		sendMessageToUUID(id, msg)
	}
}

func sseHandler(c *gin.Context) {
	sseChan := make(chan string)
	id := uuid.New().String()

	mu.Lock()
	clients[id] = sseChan
	mu.Unlock()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Content-Type")

	fmt.Fprintf(c.Writer, "data: %s\n\n", id)
	c.Writer.(http.Flusher).Flush()

	c.Stream(func(w io.Writer) bool {
		//fmt.Fprintf(w, "data: %s\n\n", id)
		for {
			select {
			case msg, ok := <-sseChan:
				if !ok {
					return false
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				w.(http.Flusher).Flush()
			case <-time.After(30 * time.Second):
				fmt.Fprintf(w, ": keep-alive\n\n")
				w.(http.Flusher).Flush()
			}
		}
	})
}

func sendMessageToUUID(uuid string, message string) {
	mu.Lock()
	defer mu.Unlock()
	if client, ok := clients[uuid]; ok {
		client <- message
	}
}

func closeSSE(uuid string) {
	mu.Lock()
	defer mu.Unlock()
	if client, ok := clients[uuid]; ok {
		close(client)
		delete(clients, uuid)
	}
}

func main() {
	r := gin.Default()

	// Route for WebSocket
	r.GET("/ws", wsHandler)

	// Route for SSE
	r.GET("/sse", sseHandler)

	r.Run(":8080")
}
