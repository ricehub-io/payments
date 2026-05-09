package polar

import (
	polargo "github.com/polarsource/polar-go"
	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
	"go.uber.org/zap"
)

type Polar struct {
	logger *zap.Logger
	cfg    *config.Config
	db     *db.Database
	sdk    *polargo.Polar
}

func NewPolar(
	logger *zap.Logger,
	cfg *config.Config,
	db *db.Database,
) *Polar {
	opts := []polargo.SDKOption{polargo.WithSecurity(cfg.PolarToken)}
	if cfg.PolarSandbox {
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
		logger.Warn("Using Polar in sandbox mode!")
	}

	sdk := polargo.New(opts...)

	return &Polar{logger, cfg, db, sdk}
}
