package validate

// ValidationError represents a failed validation check.
type ValidationError struct {
	Level     int    `json:"level"`
	NodeID    string `json:"node_id,omitempty"`
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
	Source    string `json:"source,omitempty"`
}

// ValidationWarning represents a non-fatal validation check outcome.
type ValidationWarning struct {
	Level       int    `json:"level"`
	NodeID      string `json:"node_id,omitempty"`
	WarningType string `json:"warning_type"`
	Message     string `json:"message"`
	Source      string `json:"source,omitempty"`
}

// ValidationResult aggregates validation outcomes across all enabled levels.
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

func (r *ValidationResult) addError(level int, nodeID, errType, message, source string) {
	r.Errors = append(r.Errors, ValidationError{
		Level:     level,
		NodeID:    nodeID,
		ErrorType: errType,
		Message:   message,
		Source:    source,
	})
	r.Valid = false
}

func (r *ValidationResult) addWarning(level int, nodeID, warnType, message, source string) {
	r.Warnings = append(r.Warnings, ValidationWarning{
		Level:       level,
		NodeID:      nodeID,
		WarningType: warnType,
		Message:     message,
		Source:      source,
	})
}
