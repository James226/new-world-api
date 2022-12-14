package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/james226/collab-api/diagram"
	"github.com/segmentio/ksuid"
)

type Broker struct {
	Notifier chan ClientMessage

	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]ksuid.KSUID

	editor     *diagram.Editor

	onClose func()
}

func NewBroker(id string, onClose func()) (broker *Broker) {
	editor := diagram.Editor{
		Id: id,
	}

	broker = &Broker{
		Notifier:       make(chan ClientMessage, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]ksuid.KSUID),
		editor:         &editor,
		onClose:        onClose,
	}

	go broker.listen()

	return
}

func (broker *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	flusher, ok := rw.(http.Flusher)

	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")

	messageChan := make(chan []byte)

	broker.newClients <- messageChan

	defer func() {
		broker.closingClients <- messageChan
	}()

	notify := req.Context().Done()

	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	for {
		fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)
		flusher.Flush()
	}

}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:
			numClients := len(broker.clients)
			clients := make([]string, 0, numClients)
			for _, id := range broker.clients {
				clients = append(clients, id.String())
			}

			clientId := ksuid.New()

			for clientMessageChan := range broker.clients {
				bytes, err := json.Marshal(diagram.InitialMessage{Type: "client-connected", ClientId: clientId.String()})
				if err != nil {
					panic(err)
				}
				clientMessageChan <- bytes
			}

			broker.clients[s] = clientId
			bytes, err := json.Marshal(diagram.InitialMessage{Type: "connected", Clients: clients, ClientId: broker.clients[s].String()})
			if err != nil {
				panic(err)
			}
			s <- bytes
			log.Printf("Client added. %d registered clients", len(broker.clients))

		case s := <-broker.closingClients:
			clientId := broker.clients[s].String()
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))

			if len(broker.clients) == 0 {
				log.Printf("Closing broker: %s", broker.editor.Id)
				broker.onClose()
				return
			}

			for clientMessageChan := range broker.clients {
				bytes, err := json.Marshal(diagram.InitialMessage{Type: "client-disconnected", ClientId: clientId})
				if err != nil {
					panic(err)
				}
				clientMessageChan <- bytes
			}

		case event := <-broker.Notifier:
			if event.Message.Type == "offer" || event.Message.Type == "answer" || event.Message.Type == "ice" {
				for client, id := range broker.clients {
					if id.String() == event.Message.For {
						bytes, _ := json.Marshal(event.Message)
						client <- bytes
						continue
					}
				}
				continue
			}

			broadcastMessages := broker.editor.Process(&event.Message)

			for _, message := range broadcastMessages {
				for clientMessageChan := range broker.clients {
					bytes, err := json.Marshal(message)
					if err != nil {
						panic(err)
					}
					clientMessageChan <- bytes
				}
			}
		}
	}
}
