// Dapr Publisher ("dalno"): recibe HTTP de Rust y publica el evento con el
// SDK de Dapr (pub/sub). El sidecar lo entrega a RabbitMQ.
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

const (
	pubsubName = "rabbitmq-pubsub"
	topic      = "war-reports"
)

func main() {
	var client dapr.Client
	var err error
	for i := 0; i < 30; i++ {
		if client, err = dapr.NewClient(); err == nil {
			break
		}
		log.Printf("esperando sidecar de Dapr (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("dapr client: %v", err)
	}
	defer client.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /publish", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil || !json.Valid(body) {
			http.Error(w, "json invalido", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := client.PublishEvent(ctx, pubsubName, topic, body,
			dapr.PublishEventWithContentType("application/json")); err != nil {
			log.Printf("publish: %v", err)
			http.Error(w, "publish fallo", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	log.Println("dapr publisher escuchando en :8090")
	log.Fatal(http.ListenAndServe(":8090", mux))
}
