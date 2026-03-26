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
	name        string
	event       engine.EventType
	filters     map[string]string
	actions     []action.Action
	actionTypes []string // parallel to actions, for introspection
}

// RuleInfo is the public representation of a compiled rule, used by GET /rules.
type RuleInfo struct {
	Name        string            `json:"name"`
	Event       string            `json:"event"`
	Filters     map[string]string `json:"filters,omitempty"`
	ActionTypes []string          `json:"actions"`
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
		var actionTypes []string
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
			actionTypes = append(actionTypes, ac.Type)
		}
		rules = append(rules, compiledRule{
			name:        rc.Name,
			event:       engine.EventType(rc.Event),
			filters:     rc.Filters,
			actions:     actions,
			actionTypes: actionTypes,
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

// GetRules returns a snapshot of the compiled rules for introspection.
func (e *Engine) GetRules() []RuleInfo {
	out := make([]RuleInfo, len(e.rules))
	for i, r := range e.rules {
		out[i] = RuleInfo{
			Name:        r.name,
			Event:       string(r.event),
			Filters:     r.filters,
			ActionTypes: r.actionTypes,
		}
	}
	return out
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
