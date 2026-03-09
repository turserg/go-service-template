package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/turserg/go-service-template/internal/platform/config"
	"github.com/turserg/go-service-template/internal/platform/logger"
	"github.com/turserg/go-service-template/internal/platform/observability"
	postgresplatform "github.com/turserg/go-service-template/internal/platform/postgres"
	memoryrepo "github.com/turserg/go-service-template/internal/repository/memory"
	postgresrepo "github.com/turserg/go-service-template/internal/repository/postgres"
	grpctransport "github.com/turserg/go-service-template/internal/transport/grpc"
	httptransport "github.com/turserg/go-service-template/internal/transport/http"
	bookingusecase "github.com/turserg/go-service-template/internal/usecase/booking"
	catalogusecase "github.com/turserg/go-service-template/internal/usecase/catalog"
	ticketingusecase "github.com/turserg/go-service-template/internal/usecase/ticketing"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := config.Load()
	log := logger.NewJSON(cfg.ServiceName)

	if err := run(cfg, log); err != nil {
		log.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	telemetry, err := observability.New(rootCtx, observability.Config{
		ServiceName:  cfg.ServiceName,
		OTLPEndpoint: cfg.OTLPEndpoint,
		OTLPInsecure: cfg.OTLPInsecure,
	})
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := telemetry.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Error("shutdown telemetry", "error", shutdownErr)
		}
	}()

	log.Info("telemetry initialized", "tracing_enabled", telemetry.Enabled(), "otlp_endpoint", cfg.OTLPEndpoint)

	if cfg.PostgresDSN == "" {
		return errors.New("POSTGRES_DSN is required")
	}

	pgClient, err := postgresplatform.NewClient(rootCtx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pgClient.Close()

	if err = postgresplatform.ApplyMigrations(rootCtx, cfg.PostgresDSN, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("apply postgres migrations: %w", err)
	}

	if err = observability.RegisterPostgresPoolMetrics(pgClient.Pool()); err != nil {
		return fmt.Errorf("register postgres pool metrics: %w", err)
	}

	bookingRepo := bookingusecase.Repository(postgresrepo.NewBookingRepository(pgClient.Pool()))
	log.Info("booking repository backend selected", "backend", "postgres")

	catalogRepo := memoryrepo.NewCatalogRepository()
	catalogUsecase := catalogusecase.NewService(catalogRepo)

	bookingUsecase := bookingusecase.NewService(bookingRepo)
	ticketRepo := memoryrepo.NewTicketRepository()
	ticketUsecase := ticketingusecase.NewService(ticketRepo)

	catalogService := grpctransport.NewCatalogService(log, catalogUsecase)
	bookingService := grpctransport.NewBookingService(log, bookingUsecase)
	ticketService := grpctransport.NewTicketService(log, ticketUsecase)

	grpcServer := grpctransport.NewServer(catalogService, bookingService, ticketService)

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	httpHandler, err := httptransport.NewHandler(rootCtx, catalogService, bookingService, ticketService)
	if err != nil {
		return fmt.Errorf("build http handler: %w", err)
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpHandler,
	}

	group, groupCtx := errgroup.WithContext(rootCtx)

	group.Go(func() error {
		log.Info("gRPC server started", "addr", cfg.GRPCAddr)
		if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
			return fmt.Errorf("serve grpc: %w", serveErr)
		}
		return nil
	})

	group.Go(func() error {
		log.Info("HTTP gateway started", "addr", cfg.HTTPAddr)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", serveErr)
		}
		return nil
	})

	group.Go(func() error {
		<-groupCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
		case <-shutdownCtx.Done():
			grpcServer.Stop()
		}

		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil && !errors.Is(shutdownErr, context.Canceled) {
			return fmt.Errorf("shutdown http server: %w", shutdownErr)
		}
		return nil
	})

	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	log.Info("service stopped")
	return nil
}
