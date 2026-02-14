package app

import (
	"regexp"
	"strings"
)

var (
	// DNS-1123 label naming convention for Kubernetes resources
	resourceNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	// Safer naming for favourites/saved outputs (alphanumeric, spaces, dashes, dots, underscores)
	safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\s\-\.\_]*[a-zA-Z0-9]$`)
)

// ValidateResourceName checks if a resource name is a valid Kubernetes DNS-1123 label.
func ValidateResourceName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	return resourceNameRegex.MatchString(name)
}

// ValidateSafeName checks if a name is safe for file paths and display.
func ValidateSafeName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	return safeNameRegex.MatchString(name)
}

// SanitizeInput removes any suspicious characters from user input strings.
func SanitizeInput(input string) string {
	// Remove common shell injection characters
	badChars := []string{";", "|", "&", "`", "$", "(", ")", "<", ">", "\\"}
	result := input
	for _, char := range badChars {
		result = strings.ReplaceAll(result, char, "")
	}
	return strings.TrimSpace(result)
}
