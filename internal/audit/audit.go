package audit

// AuditEntry is a single audit log record.
type AuditEntry struct {
	User   string
	Action string
	Target string
	Result string
}

// Append is a temporary stub until Wave2-f audit persistence is implemented.
func Append(_ string, _ AuditEntry) error {
	return nil
}
