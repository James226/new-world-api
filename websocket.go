package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"

	"github.com/james226/new-world-api/diagram"
)

type Websocket struct {
	Id          string
	messageChan chan diagram.Message
	notify      <-chan struct{}
}

func NewWebsocket(id string) *Websocket {
	return &Websocket{
		Id: id,
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
		ch := location.Channel()

		for msg := range ch {
			var message diagram.Message
			json.Unmarshal([]byte(msg.Payload), &message)

			c.messageChan <- message
		}
	}()

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
				panic(err)
			}
		}
	}()

	c.messageChan = make(chan diagram.Message)

	err = ws.WriteMessage(1, []byte(fmt.Sprintf("{\"type\":\"connected\", \"clientId\": \"%s\"}", c.Id)))
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
			bytes, _ := json.Marshal(m)
			err := ws.WriteMessage(1, bytes)
			if err != nil {
				log.Println("Websocket write error", err)
				return
			}

		case <-c.notify:
			return
		}
	}
}
