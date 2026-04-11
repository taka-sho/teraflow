package cmd

type exitCodeCarrier interface {
	error
	ExitCode() int
}

type commandExitCode struct {
	code int
}

func (e *commandExitCode) Error() string {
	return ""
}

func (e *commandExitCode) ExitCode() int {
	if e == nil || e.code <= 0 {
		return 1
	}
	return e.code
}

func newExitCode(code int) error {
	return &commandExitCode{code: code}
}
