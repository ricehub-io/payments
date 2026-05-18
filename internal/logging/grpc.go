package logging

import (
	"context"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	extLogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ZapUnaryServerInterceptor(zapLogger *zap.Logger) grpc.UnaryServerInterceptor {
	return extLogging.UnaryServerInterceptor(
		interceptorLogger(zapLogger),
		extLogging.WithLogOnEvents(extLogging.StartCall, extLogging.FinishCall),
	)
}

func SentryUnaryServerInterceptor(baseLogger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("grpc.method", info.FullMethod)
		ctx = sentry.SetHubOnContext(ctx, hub)

		reqLogger := NewScopedLogger(baseLogger, hub).With(
			zap.String("grpc.method", info.FullMethod),
		)
		ctx = context.WithValue(ctx, ctxLoggerKey, reqLogger)

		defer func() {
			if r := recover(); r != nil {
				hub.RecoverWithContext(ctx, r)
				hub.Flush(5 * time.Second)
				panic(r)
			}
		}()

		resp, err = handler(ctx, req)
		if err != nil && shouldReport(err) {
			hub.CaptureException(err)
		}

		return resp, err
	}
}

// NewScopedLogger returns a child logger whose Error/Fatal calls
// report to given request-scoped hub instead of the global one.
func NewScopedLogger(base *zap.Logger, hub *sentry.Hub) *zap.Logger {
	core, err := zapsentry.NewCore(
		zapsentry.Configuration{
			Level:             zapcore.ErrorLevel,
			EnableBreadcrumbs: true,
			BreadcrumbLevel:   zapcore.InfoLevel,
		},
		zapsentry.NewSentryClientFromClient(hub.Client()),
	)
	if err != nil {
		return base // fallback
	}

	return zapsentry.AttachCoreToLogger(core, base)
}

// -- HELPERS --
func interceptorLogger(l *zap.Logger) extLogging.Logger {
	return extLogging.LoggerFunc(func(
		_ context.Context,
		lvl extLogging.Level,
		msg string,
		fields ...any,
	) {
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
		case extLogging.LevelDebug:
			logger.Debug(msg)
		case extLogging.LevelInfo:
			logger.Info(msg)
		case extLogging.LevelWarn:
			logger.Warn(msg)
		case extLogging.LevelError:
			logger.Error(msg)
		default:
			l.Sugar().Errorf("Unknown log level: %v", lvl)
		}
	})
}

func shouldReport(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return true
	}

	switch st.Code() {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.Unauthenticated, codes.PermissionDenied, codes.Canceled:
		return false
	default:
		return true
	}
}
