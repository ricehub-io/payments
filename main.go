package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
	"github.com/ricehub-io/payments/internal/polar"
	paymentv1 "github.com/ricehub-io/proto/gen/go/payment/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := initZap(zap.InfoLevel)
	defer func() { _ = logger.Sync() }()

	if err := run(logger); err != nil {
		logger.Fatal("run failed", zap.Error(err))
	}
}

func run(logger *zap.Logger) error {
	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	db, err := db.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("new database: %w", err)
	}
	defer db.Close()

	polar := polar.NewPolar(cfg, db)
	go polar.StartWebhookHandler()
	go polar.StartSyncThread()

	port := fmt.Sprintf(":%d", cfg.Port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("net listen: %w", err)
	}

	logOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	paymentServer := NewPaymentServer(db, polar)
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(interceptorLogger(logger), logOpts...),
	))
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentServer)

	if cfg.Reflection {
		logger.Warn("Server reflection is enabled!")
		reflection.Register(grpcServer)
	}

	logger.Info("gRPC server ready!", zap.String("address", "127.0.0.1"+port))
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}

	return nil
}

func initZap(logLevel zapcore.LevelEnabler) *zap.Logger {
	encodeCfg := zap.NewDevelopmentEncoderConfig()
	encodeCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeCfg.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006/01/02 15:04:05"))
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encodeCfg)
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevel)

	logger := zap.New(core)
	zap.ReplaceGlobals(logger)

	return logger
}

func interceptorLogger(l *zap.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		f := make([]zap.Field, 0, len(fields)/2)
		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			value := fields[i+1]

			switch v := value.(type) {
			case string:
				f = append(f, zap.String(key.(string), v))
			case int:
				f = append(f, zap.Int(key.(string), v))
			case bool:
				f = append(f, zap.Bool(key.(string), v))
			default:
				f = append(f, zap.Any(key.(string), v))
			}
		}

		logger := l.WithOptions(zap.AddCallerSkip(1)).With(f...)

		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg)
		case logging.LevelInfo:
			logger.Info(msg)
		case logging.LevelWarn:
			logger.Warn(msg)
		case logging.LevelError:
			logger.Error(msg)
		default:
			l.Sugar().Errorf("Unknown log level: %v", lvl)
		}
	})
}
