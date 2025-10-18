package main

import (
	"log"
	"net"

	pb "github.com/inter-verse/services/report-servic./gen"
	"github.com/inter-verse/services/report-service/internal/config"
	"github.com/inter-verse/services/report-service/internal/database"
	"github.com/inter-verse/services/report-service/internal/handler"
	"github.com/inter-verse/services/report-service/internal/repository"
	"github.com/inter-verse/services/report-service/internal/service"
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
