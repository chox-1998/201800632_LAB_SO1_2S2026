// Container 2 del Go Deployment 2: Writer de RabbitMQ.
// Expone HTTP en localhost:8083 y publica cada reporte en la cola durable.
// Concurrencia: pool de channels AMQP (un channel no es thread-safe).
package main

import (
	"context"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const queueName = "war_reports"

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:5672/",
		url.QueryEscape(getenv("RABBITMQ_USER", "guest")),
		url.QueryEscape(getenv("RABBITMQ_PASS", "guest")),
		getenv("RABBITMQ_HOST", "rabbitmq"))

	var conn *amqp.Connection
	var err error
	for i := 0; i < 30; i++ {
		if conn, err = amqp.Dial(amqpURL); err == nil {
			break
		}
		log.Printf("esperando RabbitMQ (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("no se pudo conectar a RabbitMQ: %v", err)
	}
	go func() {
		e := <-conn.NotifyClose(make(chan *amqp.Error, 1))
		log.Fatalf("conexion AMQP cerrada: %v (el pod se reinicia)", e)
	}()

	poolSize, _ := strconv.Atoi(getenv("POOL_SIZE", "8"))
	pool := make(chan *amqp.Channel, poolSize)
	for i := 0; i < poolSize; i++ {
		ch, err := conn.Channel()
		if err != nil {
			log.Fatalf("channel: %v", err)
		}
		if i == 0 {
			if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
				log.Fatalf("queue declare: %v", err)
			}
		}
		pool <- ch
	}
	log.Printf("conectado a RabbitMQ, cola=%s, pool=%d", queueName, poolSize)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /publish", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil || !json.Valid(body) {
			http.Error(w, "json invalido", http.StatusBadRequest)
			return
		}
		ch := <-pool
		defer func() { pool <- ch }()
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		err = ch.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})
		if err != nil {
			log.Printf("publish: %v", err)
			http.Error(w, "publish fallo", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	log.Println("writer escuchando en :8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}
