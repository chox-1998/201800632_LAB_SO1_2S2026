// Container 1 del Go Deployment 1: API REST.
// Recibe JSON de Rust y lo entrega al container gRPC client (localhost del pod).
package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	upstream := getenv("GRPC_CLIENT_URL", "http://localhost:8082/send")
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{MaxIdleConns: 200, MaxIdleConnsPerHost: 200},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /report", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, `{"error":"body invalido"}`, http.StatusBadRequest)
			return
		}
		resp, err := client.Post(upstream, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("grpc-client no responde: %v", err)
			http.Error(w, `{"error":"grpc-client no responde"}`, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Println("go api escuchando en :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
