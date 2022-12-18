package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/james226/collab-api/diagram"
	"github.com/segmentio/ksuid"
	"log"
	"net/http"
	"os"
)

type ClientMessage struct {
	Message diagram.Message
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-14146.c268.eu-west-1-2.ec2.cloud.redislabs.com:14146",
		Password: os.Getenv("REDIS_PASSWORD"), // no password set
		DB:       0,                           // use default DB
	})

	defer rdb.Close()

	router.Handle("/health", healthController{})

	router.HandleFunc("/events/{id:[\\w\\d]+}", func(response http.ResponseWriter, request *http.Request) {
		id := mux.Vars(request)["id"]
		c := NewClient(id)

		c.ServeHTTP(response, request, rdb)
	})

	router.HandleFunc("/update/{id:[\\w\\d]+}", func(response http.ResponseWriter, request *http.Request) {
		//id := mux.Vars(request)["id"]

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

		data, _ := json.Marshal(msg)

		err = rdb.Publish(request.Context(), "location-1.1.1", data).Err()
		if err != nil {
			panic(err)
		}
	})

	router.HandleFunc("/client", func(response http.ResponseWriter, request *http.Request) {
		id := ksuid.New().String()
		log.Printf("Client Connected %s", id)
		c := NewWebsocket(id)

		c.ServeHTTP(response, request, rdb)
		log.Printf("Client closed %s", id)

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
