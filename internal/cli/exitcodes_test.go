package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExitCodeFromErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		msg  string
	}{
		{"nil is ok", nil, ExitOK, ""},
		{"drift ExitError no msg", &ExitError{Code: ExitDrift}, ExitDrift, ""},
		{"internal ExitError with msg", &ExitError{Code: ExitInternal, Msg: "boom"}, ExitInternal, "boom"},
		{"wrapped ExitError still resolves", fmt.Errorf("wrap: %w", &ExitError{Code: ExitDrift}), ExitDrift, ""},
		{"plain error maps to internal", errors.New("nope"), ExitInternal, "nope"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, msg := ExitCodeFromErr(tc.err)
			require.Equal(t, tc.code, code)
			require.Equal(t, tc.msg, msg)
		})
	}
}
