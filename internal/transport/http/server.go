package httptransport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	bookingv1 "github.com/turserg/go-service-template/gen/go/booking/v1"
	catalogv1 "github.com/turserg/go-service-template/gen/go/catalog/v1"
	ticketingv1 "github.com/turserg/go-service-template/gen/go/ticketing/v1"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	// Developer portal.
	httpMux.HandleFunc("/", developerHomeHandler)
	httpMux.HandleFunc("/swagger", swaggerRedirectHandler)
	httpMux.HandleFunc("/swagger/", swaggerUIHandler)
	httpMux.Handle("/swagger/specs/", http.StripPrefix("/swagger/specs/", http.FileServer(http.Dir("gen/openapiv2"))))

	// Operational endpoints.
	httpMux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	httpMux.Handle("/metrics", promhttp.Handler())
	httpMux.HandleFunc("/debug/pprof/", pprof.Index)
	httpMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	httpMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	httpMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	httpMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Public API.
	httpMux.Handle("/v1/", gatewayMux)

	withMetrics := metricsMiddleware(httpMux)
	withTelemetry := otelhttp.NewHandler(
		withMetrics,
		"http-server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + normalizeRoute(r.URL.Path)
		}),
	)
	return withTelemetry, nil
}

func developerHomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Service Template Portal</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 32px; color: #1f2937; }
    h1 { margin-bottom: 8px; }
    p { color: #4b5563; }
    ul { line-height: 1.9; }
    a { color: #0f766e; text-decoration: none; }
    a:hover { text-decoration: underline; }
    code { background: #f3f4f6; padding: 2px 6px; border-radius: 6px; }
  </style>
</head>
<body>
  <h1>Service Template Portal</h1>
  <p>Entry point for local API and observability endpoints.</p>
  <ul>
    <li><a href="/swagger/">Swagger UI (all APIs)</a> <code>/swagger/</code></li>
    <li><a href="/swagger/specs/public.swagger.json">OpenAPI spec (merged)</a></li>
    <li><a href="/metrics">Metrics</a> <code>/metrics</code></li>
    <li><a href="/debug/pprof/">pprof</a> <code>/debug/pprof/</code></li>
    <li><a href="/healthz">Health</a> <code>/healthz</code></li>
    <li><a href="/v1/catalog/events">Sample API endpoint</a> <code>/v1/catalog/events</code></li>
  </ul>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func swaggerRedirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
}

func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/swagger/" {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
		return
	}

	html := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>API Swagger</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/swagger/specs/public.swagger.json",
        dom_id: '#swagger-ui'
      });
    };
  </script>
</body>
</html>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
