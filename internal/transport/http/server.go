package httptransport

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	bookingv1 "github.com/turserg/go-service-template/gen/go/booking/v1"
	catalogv1 "github.com/turserg/go-service-template/gen/go/catalog/v1"
	ticketingv1 "github.com/turserg/go-service-template/gen/go/ticketing/v1"
)

func NewHandler(
	ctx context.Context,
	catalogService catalogv1.CatalogServiceServer,
	bookingService bookingv1.BookingServiceServer,
	ticketService ticketingv1.TicketServiceServer,
) (http.Handler, error) {
	gatewayMux := runtime.NewServeMux()

	if err := catalogv1.RegisterCatalogServiceHandlerServer(ctx, gatewayMux, catalogService); err != nil {
		return nil, fmt.Errorf("register catalog gateway handlers: %w", err)
	}
	if err := bookingv1.RegisterBookingServiceHandlerServer(ctx, gatewayMux, bookingService); err != nil {
		return nil, fmt.Errorf("register booking gateway handlers: %w", err)
	}
	if err := ticketingv1.RegisterTicketServiceHandlerServer(ctx, gatewayMux, ticketService); err != nil {
		return nil, fmt.Errorf("register ticketing gateway handlers: %w", err)
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	httpMux.Handle("/", gatewayMux)

	return httpMux, nil
}
