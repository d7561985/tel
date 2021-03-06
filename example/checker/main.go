package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/otlplog/logskd"
	"github.com/d7561985/tel/v2/otlplog/otlploggrpc"
	"github.com/d7561985/tel/v2/pkg/logtransform"
	_ "github.com/joho/godotenv/autoload"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.uber.org/zap/zapcore"
)

var addr = "0.0.0.0:4317"
var insecure bool

func load() {
	flag.BoolVar(&insecure, "insecure", true, "do it")
	flag.StringVar(&addr, "addr", "0.0.0.0:4317", "grpc addr")

	if v, ok := os.LookupEnv("OTEL_COLLECTOR_GRPC_ADDR"); ok {
		addr = v
	}

	flag.Parse()
	fmt.Println("addr", addr)
}

func main() {
	load()

	ctx := context.Background()

	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(addr)}
	if insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	client := otlploggrpc.NewClient(opts...)
	if err := client.Start(ctx); err != nil {
		tel.Global().Fatal("start client", tel.Error(err))
	}

	defer func() {
		_ = client.Stop(ctx)
	}()

	res, _ := resource.New(ctx, resource.WithAttributes(
		// the service name used to display traces in backends
		// key: service.name
		semconv.ServiceNameKey.String("PING"),
		// key: service.namespace
		semconv.ServiceNamespaceKey.String("TEST"),
		// key: service.version
		semconv.ServiceVersionKey.String("TEST"),
		semconv.ServiceInstanceIDKey.String("LOCAL"),
	))

	if err := client.UploadLogs(ctx, logtransform.Trans(res, []logskd.Log{logg()})); err != nil {
		tel.Global().Fatal("test upload logs", tel.Error(err))
	}

	tel.Global().Info("OK")
}

func logg() logskd.Log {
	return logskd.NewLog(zapcore.Entry{
		Level:      zapcore.InfoLevel,
		Time:       time.Now(),
		LoggerName: "XXX",
		Message:    "XXX",
	}, []byte("HELLO=WORLD"))
}
