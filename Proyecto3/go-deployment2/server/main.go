// Container 1 del Go Deployment 2: gRPC Server.
// Recibe SendReport y lo entrega al container writer (localhost del pod).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	pb "munk8s/go-deployment2/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedWarReportServiceServer
	writerURL string
	http      *http.Client
}

type report struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func (s *server) SendReport(ctx context.Context, req *pb.WarReportRequest) (*pb.WarReportResponse, error) {
	if req.Country == pb.Countries_countries_unknown {
		return nil, status.Error(codes.InvalidArgument, "country desconocido")
	}
	body, _ := json.Marshal(report{
		Country:         strings.ToUpper(req.Country.String()),
		WarplanesInAir:  req.WarplanesInAir,
		WarshipsInWater: req.WarshipsInWater,
		Timestamp:       req.Timestamp,
	})
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.writerURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(httpReq)
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("writer no responde: %v", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, status.Error(codes.Internal, fmt.Sprintf("writer respondio %d", resp.StatusCode))
	}
	return &pb.WarReportResponse{Status: "published"}, nil
}

func main() {
	writerURL := os.Getenv("WRITER_URL")
	if writerURL == "" {
		writerURL = "http://localhost:8083/publish"
	}
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	// MaxConnectionAge fuerza reconexiones periodicas: asi el cliente vuelve a
	// resolver DNS y reparte carga cuando se escalan las replicas.
	gs := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 10 * time.Second,
	}))
	pb.RegisterWarReportServiceServer(gs, &server{
		writerURL: writerURL,
		http: &http.Client{
			Timeout:   3 * time.Second,
			Transport: &http.Transport{MaxIdleConns: 200, MaxIdleConnsPerHost: 200},
		},
	})
	log.Println("gRPC server escuchando en :50051")
	log.Fatal(gs.Serve(lis))
}
