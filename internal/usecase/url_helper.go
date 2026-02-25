package usecase

import "strings"

// GetURLScheme returns the appropriate URL scheme (http or https) based on environment and domain
// Production always uses https, development/staging detects based on domain
func GetURLScheme(environment, domain string) string {
	// Production always uses HTTPS
	if environment == "production" {
		return "https"
	}

	// For non-production, use http for localhost/127.0.0.1, https for everything else
	if strings.Contains(domain, "localhost") || strings.Contains(domain, "127.0.0.1") {
		return "http"
	}

	// Default to https for staging environments with real domains
	return "https"
}

// BuildURL constructs a full URL with the appropriate scheme
func BuildURL(environment, domain, path string) string {
	scheme := GetURLScheme(environment, domain)
	return scheme + "://" + domain + path
}
