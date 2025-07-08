package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/LucianoGiope/openTelemetry/search-weather/internal/web"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InitProvider inicializa o TracerProvider com exporter OTLP via gRPC
func InitProvider(serviceName, collectorURL string) (func(context.Context) error, error) {
	// Contexto com timeout para a inicialização
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Cria resource com nome do serviço
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Abre conexão gRPC com o collector OTLP
	conn, err := grpc.DialContext(ctx, collectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	// Cria exporter OTLP usando a conexão gRPC
	tracerExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Cria span processor batch e o tracer provider
	bsp := sdktrace.NewBatchSpanProcessor(tracerExporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sempre amostra para testes
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// Define o tracer provider global
	otel.SetTracerProvider(tp)

	// Define o propagator para TraceContext + Baggage (padrão OpenTelemetry)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Retorna função para shutdown do provider
	return tp.Shutdown, nil
}

func main() {
	// Canal para capturar CTRL+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Inicializa o provider de tracing
	shutdown, err := InitProvider("search-weather", "otel-collector:4317")
	if err != nil {
		log.Fatalf("Erro ao iniciar provider: %v", err)
	}
	defer func() {
		// Timeout para garantir encerramento correto
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Fatalf("failed to shutdown TracerProvider: %v", err)
		}
	}()

	println("\nIniciando serviço de busca do clima na porta 8081 e aguardando requisições")
	go func() {
		routers := web.CreateNewServer()
		if err := http.ListenAndServe(":8081", routers); err != nil {
			log.Fatalf("Erro no servidor HTTP: %v", err)
		}
	}()

	// Aguarda sinal para encerrar
	select {
	case <-sigCh:
		log.Println("Shutting down gracefully, CTRL+C pressed...")
	case <-ctx.Done():
		log.Println("Shutting down due other reason...")
	}

	println("\nFinalizando serviço")
}
