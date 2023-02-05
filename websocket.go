package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"

	"github.com/james226/new-world-api/diagram"
)

type Websocket struct {
	Id           string
	messageChan  chan []byte
	notify       <-chan struct{}
	stateManager *StateManager
}

func NewWebsocket(id string, manager *StateManager) *Websocket {
	return &Websocket{
		Id:           id,
		stateManager: manager,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (c *Websocket) ServeHTTP(rw http.ResponseWriter, req *http.Request, rdb *redis.Client) {

	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Println(err)
		http.Error(rw, "Failed to upgrade connection", 500)
	}

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	c.notify = ctx.Done()

	location := rdb.Subscribe(ctx, "location-1.1.1")

	defer location.Close()

	go func() {
		for {
			// read in a message
			_, p, err := ws.ReadMessage()
			if err != nil {
				log.Println("Websocket read error", err)
				return
			}

			var message diagram.Message
			json.Unmarshal(p, &message)

			message.ClientId = c.Id

			bytes, err := json.Marshal(message)

			err = rdb.Publish(req.Context(), "location-1.1.1", bytes).Err()
			if err != nil {
				ws.Close()
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"foo": "bar",
				"nbf": time.Now().Unix(),
			})

			tokenString, err := token.SignedString([]byte("my_secret_key"))

			c.messageChan <- []byte(fmt.Sprintf("{\"state\": \"%s\"}", tokenString))
		}
	}()

	go func() {
		ch := location.Channel()

		for msg := range ch {
			c.messageChan <- []byte(msg.Payload)
		}
	}()

	c.messageChan = make(chan []byte)

	//token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	//	"position": map[string]int{
	//		"x": 2005000,
	//		"y": 0,
	//		"z": 5000,
	//	},
	//	"nbf": time.Now().Unix(),
	//})
	//
	//tokenString, err := token.SignedString([]byte("my_secret_key"))

	tokenString, err := c.stateManager.Serialize(State{
		Position: Point{
			X: 1995000,
			Y: 300,
			Z: 1000,
		},
	})

	err = ws.WriteMessage(1, []byte(fmt.Sprintf("{\"type\":\"connected\", \"clientId\": \"%s\", \"state\":\"%s\"}", c.Id, tokenString)))
	if err != nil {
		log.Println(err)
	}

	message := fmt.Sprintf("{\"type\":\"client_connected\", \"clientId\": \"%s\"}", c.Id)
	err = rdb.Publish(req.Context(), "location-1.1.1", []byte(message)).Err()
	if err != nil {
		panic(err)
	}

	c.MessageLoop(rw, ws)

	log.Printf("Closing client %s", c.Id)

}

func (c *Websocket) MessageLoop(rw http.ResponseWriter, ws *websocket.Conn) {
	for {
		select {
		case m := <-c.messageChan:
			err := ws.WriteMessage(1, m)
			if err != nil {
				log.Println("Websocket write error", err)
				return
			}

		case <-c.notify:
			return
		}
	}
}
