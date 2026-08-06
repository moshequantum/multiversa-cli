// Package alerts persists doctor findings as a small local lifecycle ledger.
// It stores no secret values and never leaves ~/.multiversa by itself.
package alerts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/moshequantum/multiversa-cli/internal/doctor"
)

const Schema = "multiversa.alert-ledger/v1"

type State string

const (
	Open     State = "open"
	Resolved State = "resolved"
)

type Alert struct {
	Finding    doctor.Finding `json:"finding"`
	State      State          `json:"state"`
	FirstSeen  string         `json:"first_seen"`
	LastSeen   string         `json:"last_seen"`
	ResolvedAt string         `json:"resolved_at,omitempty"`
}

type Ledger struct {
	Schema    string  `json:"schema"`
	UpdatedAt string  `json:"updated_at"`
	Alerts    []Alert `json:"alerts"`
}

type Summary struct {
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
	P0       int `json:"p0"`
	P1       int `json:"p1"`
	P2       int `json:"p2"`
	P3       int `json:"p3"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".multiversa", "alerts.json"), nil
}

func Load(path string) (Ledger, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Ledger{Schema: Schema}, nil
		}
		return Ledger{}, err
	}
	var ledger Ledger
	if err := json.Unmarshal(b, &ledger); err != nil {
		return Ledger{}, fmt.Errorf("parse alert ledger %s: %w", path, err)
	}
	if ledger.Schema != "" && ledger.Schema != Schema {
		return Ledger{}, fmt.Errorf("unsupported alert ledger schema %q", ledger.Schema)
	}
	ledger.Schema = Schema
	return ledger, nil
}

// Reconcile refreshes the ledger from current findings. It is idempotent for
// a fixed timestamp and writes atomically with mode 0600.
func Reconcile(path string, findings []doctor.Finding, now time.Time) (Ledger, error) {
	ledger, err := Load(path)
	if err != nil {
		return Ledger{}, err
	}
	stamp := now.UTC().Format(time.RFC3339)
	existing := make(map[string]Alert, len(ledger.Alerts))
	for _, a := range ledger.Alerts {
		existing[a.Finding.ID] = a
	}

	seen := map[string]bool{}
	for _, f := range findings {
		seen[f.ID] = true
		a, ok := existing[f.ID]
		if !ok {
			a = Alert{FirstSeen: stamp}
		}
		a.Finding = f
		a.State = Open
		a.LastSeen = stamp
		a.ResolvedAt = ""
		existing[f.ID] = a
	}
	for id, a := range existing {
		if seen[id] || a.State == Resolved {
			continue
		}
		a.State = Resolved
		a.ResolvedAt = stamp
		existing[id] = a
	}

	ledger = Ledger{Schema: Schema, UpdatedAt: stamp, Alerts: make([]Alert, 0, len(existing))}
	for _, a := range existing {
		ledger.Alerts = append(ledger.Alerts, a)
	}
	sortAlerts(ledger.Alerts)
	if err := save(path, ledger); err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func (l Ledger) Summary() Summary {
	var s Summary
	for _, a := range l.Alerts {
		if a.State == Resolved {
			s.Resolved++
			continue
		}
		s.Open++
		switch a.Finding.Severity {
		case doctor.P0:
			s.P0++
		case doctor.P1:
			s.P1++
		case doctor.P2:
			s.P2++
		case doctor.P3:
			s.P3++
		}
	}
	return s
}

func (l Ledger) OpenAlerts() []Alert {
	var out []Alert
	for _, a := range l.Alerts {
		if a.State == Open {
			out = append(out, a)
		}
	}
	return out
}

func save(path string, ledger Ledger) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".alerts-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func sortAlerts(alerts []Alert) {
	severityRank := map[doctor.Severity]int{doctor.P0: 0, doctor.P1: 1, doctor.P2: 2, doctor.P3: 3}
	sort.SliceStable(alerts, func(i, j int) bool {
		if alerts[i].State != alerts[j].State {
			return alerts[i].State == Open
		}
		ri, rj := severityRank[alerts[i].Finding.Severity], severityRank[alerts[j].Finding.Severity]
		if ri != rj {
			return ri < rj
		}
		return alerts[i].Finding.ID < alerts[j].Finding.ID
	})
}
