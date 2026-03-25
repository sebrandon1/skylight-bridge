package action

import (
	"context"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// Action executes a side effect in response to an event.
type Action interface {
	Execute(ctx context.Context, event engine.Event) error
}

// Factory constructs an Action from a config map.
type Factory func(config map[string]any) (Action, error)
