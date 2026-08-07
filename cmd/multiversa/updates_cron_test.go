package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// El watcher diario corría con el PATH mínimo de cron, así que los motores
// instalados bajo el home aparecían como ausentes en el reporte. `doctor`
// pedía "add-explicit-user-path-to-cron" y `updates cron --apply` no lo
// escribía: la herramienta diagnosticaba algo que ella misma no arreglaba.
func TestCronEntryDeclaraPATHExplicito(t *testing.T) {
	entry := cronEntry()

	fields := strings.Fields(entry)
	if len(fields) <= 5 {
		t.Fatalf("la línea de cron no tiene campos tras el horario: %q", entry)
	}
	if !strings.HasPrefix(fields[5], "PATH=") {
		t.Fatalf("el primer campo tras el horario debe ser PATH=; got %q en %q", fields[5], entry)
	}
	if !strings.HasSuffix(entry, cronMarker) {
		t.Fatalf("la línea debe conservar el marcador %q: %q", cronMarker, entry)
	}
}

func TestCronPATHSoloListaDirectoriosQueExisten(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "multiversa")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("crear binario de prueba: %v", err)
	}

	for _, entry := range strings.Split(cronPATH(bin), ":") {
		if entry == "" {
			t.Fatal("cronPATH produjo un segmento vacío")
		}
		fi, err := os.Stat(entry)
		if err != nil || !fi.IsDir() {
			t.Fatalf("cronPATH listó %q, que no es un directorio existente", entry)
		}
	}
}

func TestCronPATHEmpiezaPorElDirectorioDelBinario(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "multiversa")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("crear binario de prueba: %v", err)
	}

	// El build que instala el cron es el que el operador quiso programar.
	if got := strings.Split(cronPATH(bin), ":")[0]; got != dir {
		t.Fatalf("el primer segmento debe ser el directorio del binario %q; got %q", dir, got)
	}
}

func TestCronPATHNoRepiteDirectorios(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range strings.Split(cronPATH("/usr/bin/multiversa"), ":") {
		if seen[entry] {
			t.Fatalf("cronPATH repitió %q", entry)
		}
		seen[entry] = true
	}
}
