package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/james226/collab-api/diagram"
	"log"
	"net/http"
	"os"
	"sync"
)

type ClientMessage struct {
	Message diagram.Message
}

func CreateBroker(id string, onClose func()) *Broker {
	log.Printf("Creating broker: %s", id)
	return NewBroker(id, onClose)
}

func setCors(h http.Handler) http.Handler {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		origin = "http://localhost:8080"
		log.Printf("defaulting to origin %s", origin)
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func main() {
	router := mux.NewRouter()

	brokers := make(map[string]*Broker)

	mu := sync.Mutex{}

	router.Handle("/health", healthController{})

	router.HandleFunc("/events/{id:[\\w\\d]+}", func(response http.ResponseWriter, request *http.Request) {
		id := mux.Vars(request)["id"]
		mu.Lock()
		broker, ok := brokers[id]
		if !ok {
			broker = CreateBroker(id, func() {
				delete(brokers, id)
			})
			brokers[id] = broker
		}
		mu.Unlock()
		broker.ServeHTTP(response, request)
	})

	router.HandleFunc("/update/{id:[\\w\\d]+}", func(response http.ResponseWriter, request *http.Request) {
		id := mux.Vars(request)["id"]
		mu.Lock()
		broker, ok := brokers[id]
		if !ok {
			broker = CreateBroker(id, func() {
				delete(brokers, id)
			})
			brokers[id] = broker
		}

		mu.Unlock()

		if request.Method == "OPTIONS" {
			return
		}

		var msg diagram.Message

		err := json.NewDecoder(request.Body).Decode(&msg)
		if err != nil {
			fmt.Print(err.Error())
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}

		broker.Notifier <- ClientMessage{Message: msg}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
		log.Printf("defaulting to port %s", port)
	}

	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, setCors(router)); err != nil {
		log.Fatal(err)
	}
}
