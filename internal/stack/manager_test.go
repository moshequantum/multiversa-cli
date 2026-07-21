package stack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// onlyOnPath rewrites PATH to a temp dir containing just the named fake
// executables, so strategy resolution can be tested against a machine that
// deliberately lacks Homebrew.
func onlyOnPath(t *testing.T, tools ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range tools {
		p := filepath.Join(dir, tool)
		if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("writing fake %q: %v", tool, err)
		}
	}
	t.Setenv("PATH", dir)
}

// fakeBinOnPath puts one executable with scripted output on an otherwise
// empty PATH, so Status() probes can be tested without the real engine.
func fakeBinOnPath(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatalf("writing fake %q: %v", name, err)
	}
	t.Setenv("PATH", dir)
}

// gentle-ai ships under two different binary names depending on how it was
// installed: Homebrew calls it `gentle`, `go install` calls it `gentle-ai`
// after its cmd/ directory. Probing only one reported a correctly installed
// engine as missing, which is what made the wizard's review step lie.
func TestGentleAIStatusFindsBothBinaryNames(t *testing.T) {
	for _, bin := range []string{"gentle-ai", "gentle"} {
		t.Run(bin, func(t *testing.T) {
			fakeBinOnPath(t, bin, `echo "gentle-ai 1.49.0"`)

			st, err := (GentleAI{}).Status()
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if !st.Installed {
				t.Fatalf("installed as %q but reported missing", bin)
			}
			if st.Version != "gentle-ai 1.49.0" {
				t.Errorf("version = %q", st.Version)
			}
		})
	}
}

// The gentle-pi npm package declares no `bin` at all — it is a development
// harness, not a CLI — so a PATH probe can never find it. Detection has to
// ask pnpm, which owns the install.
func TestGentlePiStatusAsksPnpmNotThePath(t *testing.T) {
	fakeBinOnPath(t, "pnpm", `echo "dependencies:"; echo "└── gentle-pi@1.1.0"`)

	st, err := (GentlePi{}).Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.Installed {
		t.Fatal("gentle-pi installed via pnpm but reported missing")
	}
	if st.Version != "1.1.0" {
		t.Errorf("version = %q, want 1.1.0", st.Version)
	}
}

func TestGentlePiStatusFalseWhenAbsent(t *testing.T) {
	fakeBinOnPath(t, "pnpm", `echo "dependencies:"; echo "└── node@24.18.0"`)

	st, err := (GentlePi{}).Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Installed {
		t.Error("reported installed when pnpm does not list it")
	}
}

func TestGoModuleVersion(t *testing.T) {
	cases := map[string]string{
		"":        "@latest",
		"latest":  "@latest",
		"1.20.0":  "@v1.20.0",
		"v1.20.0": "@v1.20.0",
	}
	for in, want := range cases {
		if got := goModuleVersion(in); got != want {
			t.Errorf("goModuleVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// Every engine must offer at least one runnable command. This is the guard
// that catches a new engine added with an empty or half-written strategy.
func TestEveryEngineDeclaresARunnableStrategy(t *testing.T) {
	for id, eng := range Registry() {
		strategies := eng.Strategies("latest")
		if len(strategies) == 0 {
			t.Errorf("%s: no install strategies", id)
			continue
		}
		for i, s := range strategies {
			if len(s.Cmd) == 0 {
				t.Errorf("%s: strategy %d has an empty command", id, i)
				continue
			}
			if strings.TrimSpace(s.Cmd[0]) == "" {
				t.Errorf("%s: strategy %d has a blank program", id, i)
			}
			if s.Note == "" {
				t.Errorf("%s: strategy %d has no Note to show the user", id, i)
			}
		}
	}
}

// The T9 regression: Engram and gentle-ai are Go binaries, so a Linux box
// with the Go toolchain but no Homebrew must still resolve an install route.
// Before multi-strategy support this was impossible by construction.
func TestGoEnginesInstallWithoutHomebrew(t *testing.T) {
	onlyOnPath(t, "go")

	for _, eng := range []Engine{Engram{}, GentleAI{}} {
		s, ok := PickStrategy(eng, "latest")
		if !ok {
			t.Fatalf("%s: no viable strategy with only Go on PATH", eng.ID())
		}
		if s.Prereq != "go" {
			t.Errorf("%s: picked %q, want the Go route", eng.ID(), s.Prereq)
		}
		if s.Cmd[0] != "go" || s.Cmd[1] != "install" {
			t.Errorf("%s: unexpected command %v", eng.ID(), s.Cmd)
		}
	}
}

// Module paths differ in case upstream (Engram is capitalised, gentle-ai is
// not) and `go install` is case-sensitive, so a "normalising" refactor would
// silently break installs. Pin both.
func TestGoModulePathsMatchUpstreamCasing(t *testing.T) {
	want := map[string]string{
		"engram":    "github.com/Gentleman-Programming/engram/cmd/engram@latest",
		"gentle-ai": "github.com/gentleman-programming/gentle-ai/cmd/gentle-ai@latest",
	}
	for _, eng := range []Engine{Engram{}, GentleAI{}} {
		var got string
		for _, s := range eng.Strategies("latest") {
			if s.Prereq == "go" {
				got = s.Cmd[len(s.Cmd)-1]
			}
		}
		if got != want[eng.ID()] {
			t.Errorf("%s: module path = %q, want %q", eng.ID(), got, want[eng.ID()])
		}
	}
}

// gentle-ai's Go route is knowingly limited: upstream tags v2.x while its
// go.mod still declares the v1 module path, so `@latest` can only ever
// resolve to v1.49.0 and a `/v2` path does not exist. The Note must say so,
// otherwise the wizard silently installs a major version behind Homebrew.
// If upstream ever fixes their go.mod, this test is the reminder to revisit.
func TestGentleAIGoRouteWarnsAboutMajorVersionGap(t *testing.T) {
	var note, path string
	for _, s := range (GentleAI{}).Strategies("latest") {
		if s.Prereq == "go" {
			note, path = s.Note, s.Cmd[len(s.Cmd)-1]
		}
	}
	if strings.Contains(path, "/v2") {
		t.Error("upstream go.mod does not declare a /v2 module path — this install would fail")
	}
	if !strings.Contains(note, "v1.x") {
		t.Errorf("the Note must warn the user they get v1.x, got %q", note)
	}
}

// Homebrew stays the preferred route where upstream blesses it.
func TestHomebrewWinsWhenAvailable(t *testing.T) {
	onlyOnPath(t, "brew", "go")

	s, ok := PickStrategy(Engram{}, "latest")
	if !ok {
		t.Fatal("no viable strategy with brew and go on PATH")
	}
	if s.Prereq != "brew" {
		t.Errorf("picked %q, want brew to take precedence", s.Prereq)
	}
}

// AGPL-3.0 invariant: MiroFish is pulled as a container and never built from
// source. A future contributor adding a `go install` or `git clone` route
// here would create a licensing problem, so fail loudly.
func TestMiroFishNeverBuildsFromSource(t *testing.T) {
	banned := []string{"go", "git", "cargo", "make", "pip", "pipx"}
	for _, s := range (MiroFish{}).Strategies("latest") {
		if s.Cmd[0] != "docker" {
			t.Errorf("MiroFish must be pulled with docker, got %q", s.Cmd[0])
		}
		for _, b := range banned {
			if s.Prereq == b || s.Cmd[0] == b {
				t.Errorf("MiroFish must not build from source (found %q) — AGPL-3.0", b)
			}
		}
	}
}

// When nothing is installable the user gets every option, not just the first.
func TestPickStrategyFailsClosedAndNamesEveryOption(t *testing.T) {
	onlyOnPath(t) // empty PATH

	s, ok := PickStrategy(Engram{}, "latest")
	if ok {
		t.Fatal("expected no viable strategy on an empty PATH")
	}
	if s.Prereq != "brew" {
		t.Errorf("fallback should still report the preferred route, got %q", s.Prereq)
	}

	opts := Prereqs(Engram{}, "latest")
	if len(opts) != 2 || opts[0] != "brew" || opts[1] != "go" {
		t.Fatalf("Prereqs = %v, want [brew go]", opts)
	}

	err := &PrereqError{Engine: "engram", Options: opts}
	for _, want := range []string{"engram", "brew", "go"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

// Engines with a single route keep their original behaviour verbatim.
func TestSingleStrategyEnginesAreUnchanged(t *testing.T) {
	cases := map[string][]string{
		"graphify":  {"pipx", "install", "graphifyy"},
		"gentle-pi": {"pnpm", "add", "-g", "gentle-pi"},
		"codegraph": {"pnpm", "add", "-g", "@colbymchenry/codegraph"},
	}
	for id, want := range cases {
		eng, err := Resolve(id)
		if err != nil {
			t.Fatalf("resolving %s: %v", id, err)
		}
		strategies := eng.Strategies("latest")
		if len(strategies) != 1 {
			t.Errorf("%s: %d strategies, want 1", id, len(strategies))
			continue
		}
		if strings.Join(strategies[0].Cmd, " ") != strings.Join(want, " ") {
			t.Errorf("%s: command = %v, want %v", id, strategies[0].Cmd, want)
		}
	}
}

// Pinned versions must reach the package manager in its own syntax.
func TestVersionPinningPerPackageManager(t *testing.T) {
	cases := map[string]struct{ engine, want string }{
		"pipx":      {"graphify", "graphifyy==1.2.3"},
		"pnpm":      {"gentle-pi", "gentle-pi@1.2.3"},
		"goinstall": {"engram", "github.com/Gentleman-Programming/engram/cmd/engram@v1.2.3"},
	}
	for name, c := range cases {
		eng, err := Resolve(c.engine)
		if err != nil {
			t.Fatalf("resolving %s: %v", c.engine, err)
		}
		var found bool
		for _, s := range eng.Strategies("1.2.3") {
			if s.Cmd[len(s.Cmd)-1] == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no strategy produced %q", name, c.want)
		}
	}
}
