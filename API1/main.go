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

	api2URL = "http://192.168.100.3:8082/health"
	api3URL = "http://192.168.100.4:8083/health"
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

func checkAPIHealth(url string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var health HealthResponse

	err = json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		return false
	}

	return health.Status == "UP"
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
	connection := checkAPIHealth(api2URL)

	var response CallResponse

	if connection {
		response = CallResponse{
			APIName:    "API2",
			Message:    "The API2 located on the VM1 is working",
			Connection: true,
			Carnet:     carnet,
		}
	} else {
		response = CallResponse{
			APIName:    "API2",
			Message:    "ERROR: The API2 located on the VM1 is not working",
			Connection: false,
			Carnet:     carnet,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func callAPI3Handler(w http.ResponseWriter, r *http.Request) {
	connection := checkAPIHealth(api3URL)

	var response CallResponse

	if connection {
		response = CallResponse{
			APIName:    "API3",
			Message:    "The API3 located on the VM2 is working",
			Connection: true,
			Carnet:     carnet,
		}
	} else {
		response = CallResponse{
			APIName:    "API3",
			Message:    "ERROR: The API3 located on the VM2 is not working",
			Connection: false,
			Carnet:     carnet,
		}
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