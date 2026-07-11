package cli

import (
	"errors"
	"fmt"
)

// Exit codes used across all psy subcommands.
const (
	ExitOK       = 0
	ExitDrift    = 1 // psy check: drift detected (gate failure)
	ExitInternal = 70
)

// ExitError carries a process exit code distinct from ExitInternal. An empty
// Msg suppresses the "psy: <msg>" line that main otherwise prints to stderr.
// It lets `psy check` report drift (exit 1) without looking like an error
// (exit 70).
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

// ExitCodeFromErr maps a command error to a process exit code plus an
// optional stderr message.
//
//	nil             -> (ExitOK, "")
//	*ExitError      -> (Code, Msg)
//	any other error -> (ExitInternal, err.Error())
func ExitCodeFromErr(err error) (int, string) {
	if err == nil {
		return ExitOK, ""
	}
	var ec *ExitError
	if errors.As(err, &ec) {
		return ec.Code, ec.Msg
	}
	return ExitInternal, err.Error()
}
