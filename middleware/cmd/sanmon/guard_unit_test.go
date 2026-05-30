package main

import "testing"

func TestGuardExitCode(t *testing.T) {
	cases := []struct {
		agent string
		pass  bool
		want  int
	}{
		{"generic", false, 2},
		{"generic", true, 0},
		{"claude", false, 0},
		{"claude", true, 0},
		{"codex", false, 0},
		{"codex", true, 0},
	}
	for _, c := range cases {
		if got := guardExitCode(c.agent, c.pass); got != c.want {
			t.Errorf("guardExitCode(%q, %v) = %d, want %d", c.agent, c.pass, got, c.want)
		}
	}
}
