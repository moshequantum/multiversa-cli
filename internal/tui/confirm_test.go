package tui

import "testing"

func TestConfirmDecision(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Explicit yes — the only inputs that should pass.
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{"  yes  ", true},

		// Everything else, especially the ambiguous cases that
		// scripts sometimes pass, MUST return false. This protects
		// `multiversa usb` and similar destructive flows.
		{"", false},
		{" ", false},
		{"n", false},
		{"no", false},
		{"NO", false},
		{"maybe", false},
		{"ye", false},
		{"yess", false},
		{"sí", false}, // Spanish yes is intentionally NOT honored — keep ASCII for unambiguous CI.
		{"1", false},
		{"true", false},
	}
	for _, c := range cases {
		if got := ConfirmDecision(c.in); got != c.want {
			t.Errorf("ConfirmDecision(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
