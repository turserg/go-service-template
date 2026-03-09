package grpctransport

import (
	bookingv1 "github.com/turserg/go-service-template/gen/go/booking/v1"
	catalogv1 "github.com/turserg/go-service-template/gen/go/catalog/v1"
	ticketingv1 "github.com/turserg/go-service-template/gen/go/ticketing/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewServer(
	catalogService catalogv1.CatalogServiceServer,
	bookingService bookingv1.BookingServiceServer,
	ticketService ticketingv1.TicketServiceServer,
) *grpc.Server {
	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	catalogv1.RegisterCatalogServiceServer(server, catalogService)
	bookingv1.RegisterBookingServiceServer(server, bookingService)
	ticketingv1.RegisterTicketServiceServer(server, ticketService)
	reflection.Register(server)

	return server
}
