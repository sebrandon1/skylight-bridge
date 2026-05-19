package action

import (
	"fmt"
	"net/url"
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
