package rules

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sebrandon1/skylight-bridge/action"
	"github.com/sebrandon1/skylight-bridge/config"
	"github.com/sebrandon1/skylight-bridge/engine"
)

type compiledRule struct {
	name    string
	event   engine.EventType
	filters map[string]string
	actions []action.Action
}

// Engine matches events against rules and dispatches actions.
type Engine struct {
	rules  []compiledRule
	logger *slog.Logger
}

// NewEngine compiles rule configs into an Engine.
func NewEngine(configs []config.RuleConfig, factories map[string]action.Factory, logger *slog.Logger) (*Engine, error) {
	var rules []compiledRule
	for _, rc := range configs {
		var actions []action.Action
		for j, ac := range rc.Actions {
			factory, ok := factories[ac.Type]
			if !ok {
				return nil, fmt.Errorf("rule %q action %d: unknown type %q", rc.Name, j, ac.Type)
			}
			a, err := factory(ac.Config)
			if err != nil {
				return nil, fmt.Errorf("rule %q action %d (%s): %w", rc.Name, j, ac.Type, err)
			}
			actions = append(actions, a)
		}
		rules = append(rules, compiledRule{
			name:    rc.Name,
			event:   engine.EventType(rc.Event),
			filters: rc.Filters,
			actions: actions,
		})
	}
	return &Engine{rules: rules, logger: logger}, nil
}

// HandleEvent checks each rule against the event and executes matching actions.
func (e *Engine) HandleEvent(ctx context.Context, event engine.Event) {
	for _, r := range e.rules {
		if r.event != event.Type {
			continue
		}
		if !matchFilters(r.filters, event.Data) {
			continue
		}
		for _, a := range r.actions {
			if err := a.Execute(ctx, event); err != nil {
				e.logger.Error("action failed",
					slog.String("rule", r.name),
					slog.String("error", err.Error()),
				)
			}
		}
	}
}

func matchFilters(filters map[string]string, data map[string]any) bool {
	for key, want := range filters {
		got, ok := data[key]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", got) != want {
			return false
		}
	}
	return true
}
