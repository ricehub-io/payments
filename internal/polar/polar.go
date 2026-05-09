package polar

import (
	polargo "github.com/polarsource/polar-go"
	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
	"go.uber.org/zap"
)

type Polar struct {
	cfg *config.Config
	db  *db.Database
	sdk *polargo.Polar
}

func NewPolar(cfg *config.Config, db *db.Database) *Polar {
	opts := []polargo.SDKOption{polargo.WithSecurity(cfg.PolarToken)}
	if cfg.PolarSandbox {
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
		zap.L().Warn("Using Polar in sandbox mode!")
	}

	sdk := polargo.New(opts...)

	return &Polar{cfg, db, sdk}
}
