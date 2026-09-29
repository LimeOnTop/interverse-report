package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	pb "github.com/LimeOnTop/interverse-contracts/report/gen"
	"github.com/LimeOnTop/interverse-report/cmd/config"
	"github.com/LimeOnTop/interverse-report/internal/apperr"
	"github.com/LimeOnTop/interverse-report/internal/client"
	"github.com/LimeOnTop/interverse-report/internal/consumer"
	"github.com/LimeOnTop/interverse-report/internal/controller"
	"github.com/LimeOnTop/interverse-report/internal/keycache"
	"github.com/LimeOnTop/interverse-report/internal/producer"
	"github.com/LimeOnTop/interverse-report/internal/repository"
	"github.com/LimeOnTop/interverse-report/internal/service"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const shutdownTimeout = 15 * time.Second

func main() {
	cfg := config.Load()

	apperr.Configure(cfg.DevMode)
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		panic("open database: " + err.Error())
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(30)
	db.SetConnMaxIdleTime(30 * time.Minute)

	defer db.Close()

	reportRepository := repository.NewReportRepository(db)
	outboxRepository := repository.NewOutboxRepository(db)
	reportService := service.NewReportService(reportRepository)

	interviewClient, err := client.NewInterviewClient(cfg.InterviewServiceURL)
	if err != nil {
		panic("interview client: " + err.Error())
	}

	questionClient, err := client.NewQuestionClient(cfg.QuestionServiceURL)
	if err != nil {
		panic("question client: " + err.Error())
	}

	if cfg.GeminiHTTPProxy != "" {
		log.Printf("gemini http proxy enabled: %s", cfg.GeminiHTTPProxy)
	}
	if cfg.GeminiAPIBase != "" && cfg.GeminiAPIBase != "https://generativelanguage.googleapis.com" {
		log.Printf("gemini api base override: %s", cfg.GeminiAPIBase)
	}

	geminiClient := client.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiHTTPProxy, cfg.GeminiAPIBase)
	deepseekClient := client.NewDeepSeekClient(cfg.DeepSeekAPIKey, cfg.DeepSeekModel, cfg.DeepSeekAPIBase)
	llmAnalyzer := buildAnalyzer(cfg.LLMProvider, geminiClient, deepseekClient)

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   cfg.RedisAnswersDB,
	})
	defer redisClient.Close()

	addresses := producer.SplitAddresses(cfg.KafkaBrokers)

	messageBroker, err := producer.NewMessageBroker(producer.BrokerConfig{Addresses: addresses})
	if err != nil {
		panic("message broker: " + err.Error())
	}
	defer messageBroker.Close()

	reportProducer := producer.NewOutboxPublisher(outboxRepository, cfg.KafkaTopicReports)
	var _ usecase.ReportProducer = reportProducer

	outboxRelay := producer.NewOutboxRelay(outboxRepository, messageBroker, producer.RelayConfig{
		BatchSize:    50,
		PollInterval: time.Second,
		MaxAttempts:  10,
	})

	idempotency := keycache.NewStore(redisClient, 72*time.Hour)
	trainingAnalyzer := service.NewTrainingReportAnalyzer()

	reportConsumer, err := consumer.NewReportConsumer(consumer.Config{
		Brokers:  addresses,
		Topic:    cfg.KafkaTopicReports,
		DLQTopic: cfg.KafkaTopicReportsDLQ,
		GroupID:  cfg.KafkaGroupID,
	}, trainingAnalyzer, idempotency)
	if err != nil {
		panic("kafka consumer: " + err.Error())
	}
	defer reportConsumer.Close()

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := outboxRelay.Run(rootCtx); err != nil {
			log.Printf("outbox relay stopped: %v", err)
		}
	}()

	go func() {
		if err := reportConsumer.Run(rootCtx); err != nil {
			log.Printf("kafka consumer stopped: %v", err)
		}
	}()

	answerCache := service.NewAnswerCache(redisClient)
	analysisService := service.NewAnalysisService(reportRepository, interviewClient, questionClient, llmAnalyzer, answerCache)
	reportController := controller.NewReportController(reportService, analysisService)

	lis, err := net.Listen("tcp", net.JoinHostPort("", cfg.Port))
	if err != nil {
		panic("listen: " + err.Error())
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryUnary()),
	)

	pb.RegisterReportServiceServer(grpcServer, reportController)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("report service listening on %s", lis.Addr())
		errCh <- grpcServer.Serve(lis)
	}()

	defer close(errCh)

	waitForShutdown(cancel, grpcServer, healthServer, errCh)
}

func recoveryUnary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf(
					"panic recovered: method=%s panic=%v\n%s",
					info.FullMethod,
					r,
					debug.Stack(),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func waitForShutdown(
	cancel context.CancelFunc,
	grpcServer *grpc.Server,
	healthServer *health.Server,
	errCh <-chan error,
) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig)
	case err := <-errCh:
		if err != nil && err != grpc.ErrServerStopped {
			panic("grpc serve: " + err.Error())
		}
		cancel()
		return
	}

	cancel()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Print("grpc server stopped")
	case <-time.After(shutdownTimeout):
		log.Print("shutdown timed out, forcing stop")
		grpcServer.Stop()
	}
}

func buildAnalyzer(provider string, gemini *client.GeminiClient, deepseek *client.DeepSeekClient) client.Analyzer {
	provider = strings.ToLower(strings.TrimSpace(provider))

	deepseekFirst := func() client.Analyzer {
		log.Printf("llm provider: deepseek primary (%d attempts), gemini fallback", client.DefaultPrimaryAttempts())
		return &client.FallbackAnalyzer{
			Primary:         deepseek,
			Secondary:       gemini,
			NamePrim:        "deepseek",
			NameSec:         "gemini",
			PrimaryAttempts: client.DefaultPrimaryAttempts(),
		}
	}

	switch provider {
	case "gemini":
		log.Printf("llm provider: gemini only")
		return gemini
	case "deepseek", "auto", "":
		if deepseek.Enabled() {
			return deepseekFirst()
		}
		log.Printf("llm provider: deepseek key missing; using gemini only")
		return gemini
	default:
		if deepseek.Enabled() {
			return deepseekFirst()
		}
		log.Printf("llm provider %q unknown and deepseek unavailable; using gemini", provider)
		return gemini
	}
}
