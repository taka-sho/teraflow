package errors

import (
	stdErrors "errors"
	"fmt"
	"sort"
	"strings"
)

// Category classifies application errors by domain.
type Category string

const (
	CatConfig  Category = "config"
	CatRBAC    Category = "rbac"
	CatGate    Category = "gate"
	CatAudit   Category = "audit"
	CatDoc     Category = "doc"
	CatTrace   Category = "trace"
	CatIndex   Category = "index"
	CatSummary Category = "summary"
	CatSkill   Category = "skill"
	CatHook    Category = "hook"
	CatAgent   Category = "agent"
	CatAI      Category = "ai"
	CatGitHub  Category = "github"
	CatState   Category = "state"
	CatCLI     Category = "cli"
	CatIO      Category = "io"
)

// AppError is a typed error for CLI/application failures.
type AppError struct {
	Code     string
	Category Category
	Message  string
	Cause    error
	ExitCode int
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if e.Cause != nil {
		return msg + ": " + e.Cause.Error()
	}
	return msg
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// CatalogEntry is generated in catalog_gen.go.
type CatalogEntry struct {
	Code     string
	Category string
	ExitCode int
	Template string
}

func New(code string, args ...any) *AppError {
	entry, ok := catalog[code]
	if !ok {
		return &AppError{
			Code:     code,
			Category: CatCLI,
			Message:  fmt.Sprintf("unknown error code: %s", code),
			ExitCode: 1,
		}
	}

	return &AppError{
		Code:     entry.Code,
		Category: Category(entry.Category),
		Message:  fmt.Sprintf(entry.Template, args...),
		ExitCode: entry.ExitCode,
	}
}

func Wrap(code string, cause error, args ...any) *AppError {
	err := New(code, args...)
	err.Cause = cause
	return err
}

func Is(err error, code string) bool {
	var appErr *AppError
	if !stdErrors.As(err, &appErr) {
		return false
	}
	return appErr.Code == code
}

func GetExitCode(err error) int {
	if err == nil {
		return 0
	}

	var appErr *AppError
	if stdErrors.As(err, &appErr) && appErr.ExitCode > 0 {
		return appErr.ExitCode
	}
	return 1
}

// ErrorJSON is the public JSON shape for CLI errors.
type ErrorJSON struct {
	ErrorCode string `json:"error_code"`
	Category  string `json:"category"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	ExitCode  int    `json:"exit_code"`
}

func (e *AppError) ToJSON() ErrorJSON {
	if e == nil {
		return ErrorJSON{}
	}

	out := ErrorJSON{
		ErrorCode: e.Code,
		Category:  string(e.Category),
		Message:   e.Message,
		ExitCode:  e.ExitCode,
	}
	if e.Cause != nil {
		out.Details = e.Cause.Error()
	}
	return out
}

// LookupCatalogEntry returns one catalog entry by error code.
func LookupCatalogEntry(code string) (CatalogEntry, bool) {
	entry, ok := catalog[strings.ToUpper(strings.TrimSpace(code))]
	return entry, ok
}

// ListCatalogEntries returns all catalog entries sorted by code.
func ListCatalogEntries() []CatalogEntry {
	entries := make([]CatalogEntry, 0, len(catalog))
	for _, entry := range catalog {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Code < entries[j].Code
	})
	return entries
}
