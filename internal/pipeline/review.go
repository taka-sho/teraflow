package pipeline

import "strings"

var reviewRank = map[string]int{
	"auto":    0,
	"review":  1,
	"approve": 2,
}

// DetermineReviewLevel determines effective review_required for one module.
func DetermineReviewLevel(spec ModuleSpec, waveDefault string, affectedModuleCount int) string {
	level := normalizeReviewLevel(waveDefault)
	if explicit := normalizeReviewLevel(spec.ReviewRequired); explicit != "" {
		level = explicit
	}

	path := strings.ToLower(strings.TrimSpace(spec.Path))
	if touchesSecurityModule(path) {
		return "approve"
	}
	if isDBSchemaChange(path) {
		return "approve"
	}
	if hasExternalAPIChange(path) {
		level = maxReviewLevel(level, "review")
	}
	if affectedModuleCount >= 5 {
		level = maxReviewLevel(level, "review")
	}
	if level == "" {
		return "review"
	}
	return level
}

func normalizeReviewLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "auto":
		return "auto"
	case "review":
		return "review"
	case "approve":
		return "approve"
	default:
		return ""
	}
}

func maxReviewLevel(a, b string) string {
	ra := reviewRank[normalizeReviewLevel(a)]
	rb := reviewRank[normalizeReviewLevel(b)]
	if rb > ra {
		return normalizeReviewLevel(b)
	}
	if normalizeReviewLevel(a) == "" {
		return normalizeReviewLevel(b)
	}
	return normalizeReviewLevel(a)
}

func touchesSecurityModule(path string) bool {
	keywords := []string{"security", "auth", "rbac", "oauth", "jwt", "token", "crypto", "secret", "permission"}
	return containsAny(path, keywords)
}

func hasExternalAPIChange(path string) bool {
	keywords := []string{"api", "client", "http", "rest", "grpc", "webhook"}
	return containsAny(path, keywords)
}

func isDBSchemaChange(path string) bool {
	keywords := []string{"schema", "migration", "migrate", "sql", "database", "db/"}
	return containsAny(path, keywords)
}

func containsAny(path string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(path, kw) {
			return true
		}
	}
	return false
}
