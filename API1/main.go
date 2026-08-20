package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const (
	port   = ":8081"
	vm     = 1
	carnet = "201800632"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	VM        int       `json:"VM"`
	Carnet    string    `json:"carnet"`
}

type CallResponse struct {
	APIName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "UP",
		Message:   "API1 is Ready",
		Timestamp: time.Now().UTC(),
		VM:        vm,
		Carnet:    carnet,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func callAPI2Handler(w http.ResponseWriter, r *http.Request) {
	response := CallResponse{
		APIName:    "API2",
		Message:    "API2 located on VM1",
		Connection: false,
		Carnet:     carnet,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func callAPI3Handler(w http.ResponseWriter, r *http.Request) {
	response := CallResponse{
		APIName:    "API3",
		Message:    "API3 located on VM2",
		Connection: false,
		Carnet:     carnet,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/health", healthHandler)

	http.HandleFunc(
		"/api1/"+carnet+"/call-api2",
		callAPI2Handler,
	)

	http.HandleFunc(
		"/api1/"+carnet+"/call-api3",
		callAPI3Handler,
	)

	log.Printf("API1 running on port %s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}