package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/james226/collab-api/diagram"
	"log"
	"net/http"
)

type Client struct {
	Id          string
	messageChan chan diagram.Message
	notify      <-chan struct{}
}

func NewClient(id string) *Client {
	return &Client{
		Id: id,
	}
}

func (c *Client) ServeHTTP(rw http.ResponseWriter, req *http.Request, rdb *redis.Client) {
	flusher, ok := rw.(http.Flusher)

	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")

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

	c.messageChan = make(chan diagram.Message)

	fmt.Fprint(rw, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	c.MessageLoop(rw, flusher)

	log.Printf("Closing client %s", c.Id)

}

func (c *Client) MessageLoop(rw http.ResponseWriter, flusher http.Flusher) {
	for {
		select {
		case m := <-c.messageChan:
			bytes, _ := json.Marshal(m)
			fmt.Fprintf(rw, "data: %s\n\n", bytes)
			flusher.Flush()

		case <-c.notify:
			return
		}
	}
}
