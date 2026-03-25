package action

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"text/template"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// LogAction logs events to stdout using slog.
type LogAction struct {
	tmpl   *template.Template
	logger *slog.Logger
}

// NewLogAction creates a LogAction from config. Supported keys:
//   - message: Go text/template string (optional)
func NewLogAction(config map[string]any) (Action, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	a := &LogAction{logger: logger}

	if msg, ok := config["message"].(string); ok && msg != "" {
		tmpl, err := template.New("log").Parse(msg)
		if err != nil {
			return nil, fmt.Errorf("parsing log message template: %w", err)
		}
		a.tmpl = tmpl
	}
	return a, nil
}

// Execute logs the event.
func (a *LogAction) Execute(_ context.Context, event engine.Event) error {
	if a.tmpl != nil {
		var buf bytes.Buffer
		if err := a.tmpl.Execute(&buf, event.Data); err != nil {
			return fmt.Errorf("executing log template: %w", err)
		}
		a.logger.Info(buf.String(), slog.String("type", string(event.Type)))
		return nil
	}

	a.logger.Info("event",
		slog.String("type", string(event.Type)),
		slog.Any("data", event.Data),
	)
	return nil
}
