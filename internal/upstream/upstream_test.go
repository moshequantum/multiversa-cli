package upstream

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"v0.4.0":                          "0.4.0",
		"0.4.0":                           "0.4.0",
		"V1.2.3":                          "1.2.3",
		"multiversa v0.4.0 (none, x)":     "0.4.0",
		"go1.26.5 linux/amd64":            "1.26.5",
		"engram version 2.1.0, MIT build": "2.1.0",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUpdateAvailable(t *testing.T) {
	cases := []struct {
		installed, latest string
		want              bool
	}{
		{"v0.4.0", "v0.5.0", true},
		{"v0.4.0", "0.4.0", false},   // same version, different prefix
		{"", "v0.5.0", false},        // unknown installed never claims update
		{"v0.4.0", "", false},        // no release info
		{"instalado", "v1.0", false}, // non-semver installed marker: report only
	}
	for _, c := range cases {
		if got := UpdateAvailable(c.installed, c.latest); got != c.want {
			t.Errorf("UpdateAvailable(%q, %q) = %v, want %v", c.installed, c.latest, got, c.want)
		}
	}
}
