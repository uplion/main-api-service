package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	// Initialize Pulsar client
	pulsarClient := initPulsarClient()
	defer pulsarClient.Close()

	// Initialize producer cache
	producerCache := NewProducerCache(&pulsarClient)

	// Map to store clients
	clientsMap := make(map[string](chan []byte))
	clientsStreamMap := make(map[string](chan []byte))
	var clientsMapMutex sync.RWMutex
	var clientsStreamMapMutex sync.RWMutex

	hostname := fqdn()
	log.Println("FQDN:", hostname)

	timeout := 5 * time.Minute
	if os.Getenv("TIMEOUT") != "" {
		timeout_env, err := time.ParseDuration(os.Getenv("TIMEOUT"))
		if err != nil {
			log.Printf(
				"Could not parse TIMEOUT environment variable: %s\n"+
					"\tTIMEOUT = %s\n"+
					"Using default timeout: %s\n",
				err,
				os.Getenv("TIMEOUT"),
				timeout,
			)
		} else {
			timeout = timeout_env
		}
	}

	var base_url string

	// Initialize Gin
	if os.Getenv("DEBUG") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.POST("/api/v1/chat/completions", func(c *gin.Context) {
		var rawBody []byte
		var request ClientRequest

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")

		if body, err := c.GetRawData(); err != nil {
			log.Println("/api/v1/chat/completions - could not read request body: ", err)
			c.JSON(http.StatusBadRequest, InvalidRequestError.Response)
			return
		} else {
			rawBody = body
			if err := json.Unmarshal(body, &request); err != nil {
				log.Println("/api/v1/chat/completions - failed to parse JSON: ", err)
				c.JSON(http.StatusBadRequest, InvalidRequestError.Response)
				return
			}
		}

		stream := false
		if request.Stream != nil && *request.Stream {
			stream = true
		}

		requestID := uuid.New().String()

		task := Task{
			RequestID: requestID,
			Stream:    stream,
			Data:      rawBody,
			EndPoint:  "http://" + base_url + "/res",
			MetaData: &MetaData{
				Headers: &c.Request.Header,
			},
		}

		if stream {
			task.EndPoint = "ws://" + base_url + "/res/ws"
		}

		producer, err := producerCache.GetProducer("model-" + request.Model)

		if err != nil {
			log.Println("/api/v1/chat/completions - could not get producer: ", err)
			c.JSON(http.StatusInternalServerError, InternalServerError.Response)
		}

		bodyBytes, err := json.Marshal(task)
		if err != nil {
			log.Println("/api/v1/chat/completions - could not marshal JSON to task: ", err)
			c.JSON(http.StatusInternalServerError, InternalServerError.Response)
			return
		}

		_, err = producer.Send(context.Background(), &pulsar.ProducerMessage{
			Payload: bodyBytes,
		})

		if err != nil {
			log.Println("/api/v1/chat/completions - could not send message to Pulsar: ", err)
			c.JSON(http.StatusInternalServerError, InternalServerError.Response)
			return
		}

		receive := make(chan []byte)

		if stream {
			clientsStreamMapMutex.Lock()
			clientsStreamMap[requestID] = receive
			clientsStreamMapMutex.Unlock()
			defer func() {
				clientsStreamMapMutex.Lock()
				delete(clientsStreamMap, requestID)
				clientsStreamMapMutex.Unlock()
			}()

			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")

			c.Stream(func(w io.Writer) bool {
				for {
					var buffer bytes.Buffer
					buffer.Write([]byte("data: "))

					select {
					case msg, ok := <-receive:
						if !ok {
							_, err := w.Write([]byte("data: [DONE]\n\n"))
							if err != nil {
								return false
							}
							w.(http.Flusher).Flush()
							return false
						}
						buffer.Write(msg)
						buffer.Write([]byte("\n\n"))
						_, err := w.Write(buffer.Bytes())
						if err != nil {
							return true
						}
					case <-time.After(30 * time.Second):
						_, err := w.Write([]byte(": keep-alive\n\n"))
						if err != nil {
							return true
						}
					}

					w.(http.Flusher).Flush()
				}
			})
		} else {
			clientsMapMutex.Lock()
			clientsMap[requestID] = receive
			clientsMapMutex.Unlock()
			defer func() {
				clientsMapMutex.Lock()
				delete(clientsMap, requestID)
				clientsMapMutex.Unlock()
			}()

			select {
			case msg := <-receive:
				c.Data(http.StatusOK, "application/json", msg)
			case <-time.After(timeout):
				c.JSON(http.StatusRequestTimeout, RequestTimeoutError.Response)
			}
		}
	})
	r.POST("/res", func(c *gin.Context) {
		var taskResult TaskResult
		if err := c.ShouldBindJSON(&taskResult); err != nil {
			log.Println("/res - failed to parse JSON: ", err)
			c.JSON(http.StatusBadRequest, InvalidRequestError.Response)
			return
		}

		clientsMapMutex.RLock()
		receive, ok := clientsMap[taskResult.RequestID]
		clientsMapMutex.RUnlock()
		if !ok {
			log.Println("/res - request ID not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Request ID not found"})
			return
		}

		receive <- taskResult.Data

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	r.GET("/res/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("/res/ws - failed to set websocket upgrade: ", err)
			return
		}
		defer conn.Close()
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if msgType == websocket.CloseMessage {
				break
			}
			if msgType != websocket.TextMessage {
				log.Println("/res/ws - received non-text message, ignoring")
				continue
			}

			var taskResult TaskStreamResult
			if err := json.Unmarshal(msg, &taskResult); err != nil {
				log.Println("/res/ws - failed to unmarshal message: ", err)
				continue
			}

			clientsStreamMapMutex.RLock()
			receive, ok := clientsStreamMap[taskResult.RequestID]
			clientsStreamMapMutex.RUnlock()
			if !ok {
				log.Println("Request ID not found")
				continue
			}

			if taskResult.Data != nil {
				receive <- *taskResult.Data
			}

			if taskResult.End {
				close(receive)
			}
		}
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	base_url = hostname + ":" + port

	log.Println("Server started on port", port)
	log.Printf("\tWorker Nodes should send task results to `%s/res` or `%s/res/ws`\n", base_url, base_url)

	err := r.Run(":" + port)
	if err != nil {
		log.Fatalln("Failed to start server", err)
		return
	}
}
