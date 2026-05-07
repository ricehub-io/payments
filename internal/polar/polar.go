package polar

import (
	"log"

	polargo "github.com/polarsource/polar-go"
)

type Polar struct {
	sdk *polargo.Polar
}

func NewPolar(token string, sandbox bool) *Polar {
	opts := []polargo.SDKOption{polargo.WithSecurity(token)}
	if sandbox {
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
		log.Println("[WARNING] Using Polar in sandbox mode.")
	}

	sdk := polargo.New(opts...)

	return &Polar{sdk}
}
