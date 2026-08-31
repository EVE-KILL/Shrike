package api

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/go-chi/chi/v5"
)

func TestLabelRouteIsPublicAndDocumented(t *testing.T) {
	a := humachi.New(chi.NewRouter(), huma.DefaultConfig("labels-test", "test"))
	registerLabelRoutes(a, Options{})
	op := a.OpenAPI().Paths["/labels"]
	if op == nil || op.Get == nil {
		t.Fatal("GET /labels was not registered")
	}
	if op.Get.OperationID != "killmail-labels" {
		t.Errorf("operation = %q", op.Get.OperationID)
	}
	if got := op.Get.Extensions["x-audience"]; got != "public" {
		t.Errorf("audience = %#v, want public", got)
	}
}

func TestPublicLabelIDsFollowCatalogueOrder(t *testing.T) {
	ids := publicLabelIDs()
	if len(ids) != len(killtype.Labels) {
		t.Fatalf("got %d ids, want %d", len(ids), len(killtype.Labels))
	}
	for i, label := range killtype.Labels {
		if ids[i] != label.ID {
			t.Errorf("ids[%d] = %q, want %q", i, ids[i], label.ID)
		}
	}
}
