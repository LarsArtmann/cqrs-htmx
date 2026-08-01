package cqrshtmx

import (
	"testing"
)

func TestAuthModeString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mode authMode
		want string
	}{
		{authNone, "none"},
		{authRequired, "required"},
		{authAuthorized, "authorized"},
		{authMode(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("authMode(%d).String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}
