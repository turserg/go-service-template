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
	grpctransport "github.com/turserg/go-service-template/internal/transport/grpc"
	httptransport "github.com/turserg/go-service-template/internal/transport/http"
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

	state := grpctransport.NewInMemoryState()
	catalogService := grpctransport.NewCatalogService(log)
	bookingService := grpctransport.NewBookingService(log, state)
	ticketService := grpctransport.NewTicketService(log, state)

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
