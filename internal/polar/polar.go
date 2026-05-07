package polar

import (
	"log"

	polargo "github.com/polarsource/polar-go"
	"github.com/ricehub-io/payments/internal/config"
	"github.com/ricehub-io/payments/internal/db"
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
		log.Println("[WARNING] Using Polar in sandbox mode.")
	}

	sdk := polargo.New(opts...)

	return &Polar{cfg, db, sdk}
}
