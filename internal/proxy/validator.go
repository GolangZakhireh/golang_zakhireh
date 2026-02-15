package proxy

import (
	"path"
	"strings"
)

// Validator checks if a module matches certain patterns
type Validator struct {
	NoSumDB   []string
	AllowList []string
	DenyList  []string
}

// NewValidator creates a validator from comma-separated strings
func NewValidator(nosumdb, allow, deny string) *Validator {
	return &Validator{
		NoSumDB:   splitClean(nosumdb),
		AllowList: splitClean(allow),
		DenyList:  splitClean(deny),
	}
}

func splitClean(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}

// IsPrivate returns true if the module matches GONOSUMDB patterns
func (v *Validator) IsPrivate(module string) bool {
	return matchAny(module, v.NoSumDB)
}

// IsAllowed returns true if the module is allowed (checks allow/deny lists)
func (v *Validator) IsAllowed(module string) bool {
	// If deny list is present and matches, deny
	if len(v.DenyList) > 0 {
		if matchAny(module, v.DenyList) {
			return false
		}
	}

	// If allow list is present, must match
	if len(v.AllowList) > 0 {
		return matchAny(module, v.AllowList)
	}

	// Default allow
	return true
}

// matchAny checks if the name matches any of the patterns.
// Patterns can be glob patterns (e.g. "github.com/foo/*") or
// prefix matches for directory-like behavior.
func matchAny(name string, patterns []string) bool {
	for _, p := range patterns {
		matched, _ := path.Match(p, name)
		if matched {
			return true
		}
		// Handle prefix matching for plain strings (e.g. "github.com/foo" matches "github.com/foo/bar")
		if strings.HasPrefix(name, p+"/") || name == p {
			return true
		}
	}
	return false
}
