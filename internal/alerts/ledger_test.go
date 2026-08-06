package alerts

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moshequantum/multiversa-cli/internal/doctor"
)

func TestReconcilePersistsOpenResolveAndReopenLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "alerts.json")
	finding := doctor.Finding{ID: "cli.duplicate-binaries", Severity: doctor.P2, Title: "Duplicado"}
	t1 := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	first, err := Reconcile(path, []doctor.Finding{finding}, t1)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Alerts) != 1 || first.Alerts[0].State != Open {
		t.Fatalf("first ledger = %+v", first)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("ledger permissions: info=%v err=%v", fi, err)
	}

	t2 := t1.Add(time.Hour)
	resolved, err := Reconcile(path, nil, t2)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Alerts[0].State != Resolved || resolved.Alerts[0].ResolvedAt == "" {
		t.Fatalf("resolved ledger = %+v", resolved)
	}

	t3 := t2.Add(time.Hour)
	reopened, err := Reconcile(path, []doctor.Finding{finding}, t3)
	if err != nil {
		t.Fatal(err)
	}
	a := reopened.Alerts[0]
	if a.State != Open || a.ResolvedAt != "" || a.FirstSeen != t1.Format(time.RFC3339) {
		t.Fatalf("reopened alert = %+v", a)
	}
	if got := reopened.Summary(); got.Open != 1 || got.P2 != 1 {
		t.Fatalf("summary = %+v", got)
	}
}
