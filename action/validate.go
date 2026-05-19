package action

import (
	"fmt"
	"net/url"
	"time"
)

func validateHTTPURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL %q must use http or https scheme", rawURL)
	}
	if u.Host == "" {
		return fmt.Errorf("URL %q is missing host", rawURL)
	}
	return nil
}

// parseTimeout reads an optional "timeout" duration string from config.
// Accepts Go duration strings (e.g. "30s", "2m"). Defaults to 10s.
func parseTimeout(config map[string]any) time.Duration {
	if s, ok := config["timeout"].(string); ok && s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	return 10 * time.Second
}
