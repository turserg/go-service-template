package observability

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Config struct {
	ServiceName  string
	OTLPEndpoint string
	OTLPInsecure bool
}

type Telemetry struct {
	enabled  bool
	shutdown func(context.Context) error
}

func New(ctx context.Context, cfg Config) (*Telemetry, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	endpoint, inferredInsecure := normalizeEndpoint(cfg.OTLPEndpoint)
	if endpoint == "" {
		return &Telemetry{
			enabled:  false,
			shutdown: func(context.Context) error { return nil },
		}, nil
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if cfg.OTLPInsecure || inferredInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("", attribute.String("service.name", cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(traceProvider)

	return &Telemetry{
		enabled:  true,
		shutdown: traceProvider.Shutdown,
	}, nil
}

func (t *Telemetry) Enabled() bool {
	return t != nil && t.enabled
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil || t.shutdown == nil {
		return nil
	}
	return t.shutdown(ctx)
}

func normalizeEndpoint(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return raw, false
		}
		return parsed.Host, parsed.Scheme == "http"
	}

	return raw, false
}
