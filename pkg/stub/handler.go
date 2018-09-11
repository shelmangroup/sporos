package stub

import (
	"context"

	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	"github.com/shelmangroup/sporos/pkg/sporos"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct{}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *api.Sporos:
		return sporos.Reconcile(o)
	}
	return nil
}
