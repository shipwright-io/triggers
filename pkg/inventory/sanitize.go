package inventory

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrInvalidGItURL unable to parse git scheme URL
var ErrInvalidGItURL = errors.New("invalid git scheme URL")

// parseGitSchemeURL parses ssh style URLs, for instance "git@hostname.tld/username/project.git".
func parseGitSchemeURL(rawURL string) (string, error) {
	if !strings.Contains(rawURL, ":") {
		return "", fmt.Errorf("%w: %q", ErrInvalidGItURL, rawURL)
	}

	rawURL = strings.TrimPrefix(rawURL, "git@")
	gitURLParts := strings.SplitN(rawURL, ":", 2)
	if len(gitURLParts) != 2 {
		return "", fmt.Errorf("%w: %q", ErrInvalidGItURL, rawURL)
	}

	urlParts := strings.SplitN(gitURLParts[1], "/", 2)
	if len(urlParts) != 2 {
		return "", fmt.Errorf("%w: %q", ErrInvalidGItURL, rawURL)
	}
	suffix := strings.TrimSuffix(urlParts[1], ".git")

	return fmt.Sprintf("%s/%s/%s", gitURLParts[0], urlParts[0], suffix), nil
}

// SanitizeURL takes a raw repository URL and returns only the hostname and path, removing possible
// prefix protocol, and extension suffix.
func SanitizeURL(rawURL string) (string, error) {
	if strings.HasPrefix(rawURL, "git@") {
		return parseGitSchemeURL(rawURL)
	}

	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", err
	}

	urlPath := strings.TrimSuffix(u.EscapedPath(), ".git")
	return fmt.Sprintf("%s%s", u.Hostname(), urlPath), nil
}

// CompareURLs compare the informed URLs.
func CompareURLs(a, b string) bool {
	if a == b {
		return true
	}
	aSanitized, err := SanitizeURL(a)
	if err != nil {
		return false
	}
	bSanitized, err := SanitizeURL(b)
	if err != nil {
		return false
	}
	return aSanitized == bSanitized
}
