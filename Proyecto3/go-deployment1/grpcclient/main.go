// Container 2 del Go Deployment 1: cliente gRPC.
// Expone HTTP solo en localhost:8082 y llama WarReportService.SendReport.
// Si GRPC_SERVER_ADDR esta vacio -> modo stub.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pb "munk8s/go-deployment1/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const serviceConfig = `{"loadBalancingConfig":[{"round_robin":{}}]}`

type report struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func writeJSON(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(body))
}

func main() {
	var client pb.WarReportServiceClient
	if addr := os.Getenv("GRPC_SERVER_ADDR"); addr != "" {
		// Service headless + dns:/// + round_robin => reparte entre replicas del server.
		conn, err := grpc.NewClient("dns:///"+addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(serviceConfig))
		if err != nil {
			log.Fatalf("grpc dial: %v", err)
		}
		defer conn.Close()
		client = pb.NewWarReportServiceClient(conn)
		log.Printf("gRPC client -> %s (round_robin)", addr)
	} else {
		log.Println("GRPC_SERVER_ADDR vacio: modo STUB")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		var rep report
		if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
			writeJSON(w, http.StatusBadRequest, `{"error":"json invalido"}`)
			return
		}
		c, ok := pb.Countries_value[strings.ToLower(rep.Country)]
		if !ok {
			writeJSON(w, http.StatusBadRequest, `{"error":"country desconocido"}`)
			return
		}
		if client == nil {
			log.Printf("[STUB] %+v", rep)
			writeJSON(w, http.StatusAccepted, `{"status":"stub"}`)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		resp, err := client.SendReport(ctx, &pb.WarReportRequest{
			Country:         pb.Countries(c),
			WarplanesInAir:  rep.WarplanesInAir,
			WarshipsInWater: rep.WarshipsInWater,
			Timestamp:       rep.Timestamp,
		})
		if err != nil {
			log.Printf("SendReport: %v", err)
			writeJSON(w, http.StatusBadGateway, `{"error":"grpc server fallo"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"status":"`+resp.Status+`"}`)
	})

	log.Println("grpc-client escuchando en :8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}
