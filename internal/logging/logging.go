package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{ name string }

var ctxLoggerKey = ctxKey{"logger"}

func Init(
	logLevel zapcore.LevelEnabler,
	withSentry bool,
	sentryDSN string,
) (*zap.Logger, error) {
	encodeCfg := zap.NewDevelopmentEncoderConfig()
	encodeCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeCfg.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006/01/02 15:04:05"))
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encodeCfg)
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevel)

	base := zap.New(core)

	if withSentry {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDSN,
			Environment:      "production",
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
			EnableTracing:    true,
		}); err != nil {
			return nil, fmt.Errorf("sentry init: %w", err)
		}

		core, err := zapsentry.NewCore(
			zapsentry.Configuration{
				Level:             zapcore.ErrorLevel,
				EnableBreadcrumbs: true,
				BreadcrumbLevel:   zapcore.InfoLevel,
			},
			zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()),
		)
		if err != nil {
			return nil, fmt.Errorf("zapsentry new core: %w", err)
		}

		logger := zapsentry.AttachCoreToLogger(core, base)
		zap.ReplaceGlobals(logger)

		return logger, nil
	}

	zap.ReplaceGlobals(base)
	return base, nil
}

func Sync() {
	_ = zap.L().Sync()
	sentry.Flush(5 * time.Second)
}

func LoggerFromContext(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	if l, ok := ctx.Value(ctxLoggerKey).(*zap.Logger); ok && l != nil {
		return l
	}
	return fallback
}
