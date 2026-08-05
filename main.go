package main

import (
	"log"
	"net"

	pb "github.com/LimeOnTop/interverse-contracts/report/gen"
	"github.com/LimeOnTop/interverse-report/internal/config"
	"github.com/LimeOnTop/interverse-report/internal/database"
	"github.com/LimeOnTop/interverse-report/internal/handler"
	"github.com/LimeOnTop/interverse-report/internal/repository"
	"github.com/LimeOnTop/interverse-report/internal/service"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	reportRepo := repository.NewReportRepository(db)

	// Initialize service
	reportService := service.NewReportService(*reportRepo, cfg.UserServiceURL, cfg.InterviewServiceURL)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterReportServiceServer(grpcServer, handler.NewReportHandler(reportService))

	log.Printf("Report Service starting on port %s", cfg.Port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
