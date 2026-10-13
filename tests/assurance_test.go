//go:build wasm

package cloudflare_test

import (
	"testing"

	"webtyp.com/cloudflare/edge"
	"webtyp.com/model"
	"webtyp.com/router"
)

func TestAssuranceAnonymous(t *testing.T) {
	r := edge.NewRouter(edge.Config{
		Authn: func(next router.HandlerFunc) router.HandlerFunc {
			return func(ctx router.Context) {
				// Authn sets MultiFactor but leaves UserID empty (anonymous)
				ctx.SetAssurance(model.AssuranceMultiFactor)
				next(ctx)
			}
		},
	})

	r.Get("/secret", func(ctx router.Context) {
		ctx.WriteStatus(200)
	}).Authenticated().RequiresMFA()

	ctx := &conformanceCtx{method: "GET", path: "/secret"}
	edge.Dispatch(r, ctx)

	if ctx.status != 403 {
		t.Errorf("expected 403, got %d", ctx.status)
	}

	if ctx.Assurance() != model.AssuranceNone {
		t.Errorf("expected None assurance on anonymous request, got %v", ctx.Assurance())
	}
}
