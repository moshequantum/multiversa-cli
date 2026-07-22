// Package arch holds no code. It holds the one rule that keeps the
// hexagon a hexagon: dependencies point inward, never outward.
//
// See docs adr-002 (multiversagroup/docs/specs/adr-002-arquitectura-hexagonal.md).
// A ring may import a ring closer to the core. It may never import one
// further out. Break that and this test fails with the exact edge.
//
// Adding a package? Give it a ring in the map below. An unclassified
// package fails the test on purpose — silent membership is how the
// architecture rots.
package arch

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/moshequantum/multiversa-cli"

// Rings, from the core outward.
const (
	ringShared   = 0 // pure leaf data. Zero deps. Importable from anywhere.
	ringCore     = 1 // domain + the ports it declares. No I/O, ever.
	ringDriven   = 2 // adapters the core drives: disk, network, processes, OS.
	ringDriving  = 3 // adapters that drive the core: TUI, wizard, JSON envelopes.
	ringAssembly = 4 // the only place that knows all rings and wires them.
)

var ringName = map[int]string{
	ringShared: "shared", ringCore: "core", ringDriven: "driven",
	ringDriving: "driving", ringAssembly: "assembly",
}

// ring assigns every package in the module. Keys are paths relative to
// the module root. Anything missing is a test failure, not a default.
var ring = map[string]int{
	// shared — pure data, no dependencies, safe from anywhere
	"internal/theme":   ringShared,
	"internal/version": ringShared,

	// core — the domain and its ports. This is what must stay testable
	// without a network, a disk, or a model.
	"internal/credits":  ringCore,
	"internal/manifest": ringCore,
	"internal/tenant":   ringCore,
	"internal/stack":    ringCore,
	"internal/lang":     ringCore,

	// driven — the outside world, one implementation per file
	"internal/exec":     ringDriven,
	"internal/detect":   ringDriven,
	"internal/profile":  ringDriven,
	"internal/upstream": ringDriven,
	"internal/embedded": ringDriven,
	"internal/adapters": ringDriven,
	"internal/backends": ringDriven,

	// driving — surfaces that call in
	"internal/tui":          ringDriving,
	"internal/wizard":       ringDriving,
	"internal/wizard/steps": ringDriving,
	"internal/agentout":     ringDriving,

	// assembly
	"cmd/multiversa": ringAssembly,
}

// knownViolations is the ratchet. Every entry is debt this test refuses
// to let grow. Fix one, delete its line. Never add a line to make a
// build pass — that is the whole point of the file.
//
// All three are the same leak: the core reaching for os/exec instead of
// declaring a ProcessRunner port. ADR-002 phase 3 closes it.
var knownViolations = map[string]bool{
	"internal/stack -> internal/exec":  true,
	"internal/lang -> internal/exec":   true,
	"internal/detect -> internal/exec": true,
}

// TestDependenciesPointInward is the rule.
func TestDependenciesPointInward(t *testing.T) {
	graph := importGraph(t)

	var unexpected, fixed []string
	for _, pkg := range sortedKeys(graph) {
		from, ok := ring[pkg]
		if !ok {
			t.Errorf("package %q has no ring in arch_test.go — classify it", pkg)
			continue
		}
		for _, imp := range graph[pkg] {
			to, ok := ring[imp]
			if !ok {
				t.Errorf("package %q has no ring in arch_test.go — classify it", imp)
				continue
			}
			if allowed(from, to) {
				continue
			}
			edge := pkg + " -> " + imp
			if knownViolations[edge] {
				continue
			}
			unexpected = append(unexpected, edge+
				"  ("+ringName[from]+" -> "+ringName[to]+")")
		}
	}

	// A known violation that no longer exists is good news the map should
	// reflect, so the ratchet actually ratchets.
	for edge := range knownViolations {
		parts := strings.Split(edge, " -> ")
		if !contains(graph[parts[0]], parts[1]) {
			fixed = append(fixed, edge)
		}
	}

	sort.Strings(unexpected)
	sort.Strings(fixed)

	for _, e := range unexpected {
		t.Errorf("dependency points outward: %s", e)
	}
	for _, e := range fixed {
		t.Errorf("violation %q is fixed — delete it from knownViolations", e)
	}
}

// allowed encodes the direction rule.
func allowed(from, to int) bool {
	switch {
	case to == ringShared:
		return true // pure leaves are free
	case to < from:
		return true // inward
	case to == from && (from == ringCore || from == ringDriving):
		return true // cohesion within the domain, and within the UI
	default:
		return false
	}
}

// importGraph parses every non-test .go file for its module-internal imports.
// Uses go/parser rather than `go list` so the test needs no toolchain
// subprocess and no third-party dependency.
func importGraph(t *testing.T) map[string][]string {
	t.Helper()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving module root: %v", err)
	}

	graph := map[string][]string{}
	fset := token.NewFileSet()

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "node_modules" ||
				(strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		pkg, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkg = filepath.ToSlash(pkg)
		if pkg == "internal/arch" {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range f.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if !strings.HasPrefix(imp, modulePath+"/") {
				continue // stdlib and third-party are not our concern
			}
			imp = strings.TrimPrefix(imp, modulePath+"/")
			if !contains(graph[pkg], imp) {
				graph[pkg] = append(graph[pkg], imp)
			}
		}
		if _, seen := graph[pkg]; !seen {
			graph[pkg] = nil // a package with no internal imports still needs a ring
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking module: %v", err)
	}
	return graph
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
