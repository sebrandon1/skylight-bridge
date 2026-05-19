package action

import (
	"context"
	"log/slog"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// Action executes a side effect in response to an event.
type Action interface {
	Execute(ctx context.Context, event engine.Event) error
}

// Factory constructs an Action from a config map.
type Factory func(config map[string]any) (Action, error)

type dryRunAction struct {
	actionType string
	logger     *slog.Logger
}

func (d *dryRunAction) Execute(_ context.Context, event engine.Event) error {
	d.logger.Info("dry-run: would execute action",
		slog.String("action_type", d.actionType),
		slog.String("event_type", string(event.Type)),
	)
	return nil
}

// WrapDryRun returns a Factory that validates config but produces no-op actions.
func WrapDryRun(actionType string, f Factory, logger *slog.Logger) Factory {
	return func(cfg map[string]any) (Action, error) {
		if _, err := f(cfg); err != nil {
			return nil, err
		}
		return &dryRunAction{actionType: actionType, logger: logger}, nil
	}
}
