package adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeHermes puts a scripted `hermes` on PATH for the duration of the test.
// script receives the subcommand args and decides what to print / exit with.
func fakeHermes(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("el doble de hermes es un script de shell")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hermes")
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\n"+script), 0o755); err != nil {
		t.Fatalf("escribir hermes falso: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// El fallo que motiva este test: `hermes mcp add` pregunta "Enable all N
// tools?". Sin terminal el prompt recibe EOF, hermes imprime "Cancelled." y
// aun así sale con código 0. Confiar en ese 0 hacía que Connect reportara
// éxito sin haber registrado nada.
func TestConnectFallaCuandoHermesCancelaElPromptYSaleCero(t *testing.T) {
	fakeHermes(t, `
case "$1 $2" in
  "mcp add") echo "  Connecting to 'multiversa'..."; echo "  Enable all 7 tools? [Y/n/select]:"; echo "  Cancelled."; exit 0 ;;
  "mcp list") echo "  No MCP servers configured."; exit 0 ;;
esac
exit 0
`)

	err := (Hermes{}).Connect(ConnectOptions{})
	if err == nil {
		t.Fatal("Connect devolvió nil pese a que hermes canceló y no registró nada")
	}
	if !strings.Contains(err.Error(), "multiversa") {
		t.Fatalf("el error debe nombrar el servidor que no quedó registrado, got: %v", err)
	}
}

func TestConnectExitosoCuandoHermesRegistraElServidor(t *testing.T) {
	fakeHermes(t, `
case "$1 $2" in
  "mcp add") echo "  Connected! Found 10 tool(s)"; exit 0 ;;
  "mcp list") echo "  multiversa  (stdio)  10 tools"; exit 0 ;;
esac
exit 0
`)

	if err := (Hermes{}).Connect(ConnectOptions{}); err != nil {
		t.Fatalf("Connect debía tener éxito cuando hermes sí registra: %v", err)
	}
}

func TestConnectPropagaElFalloDuroDeHermes(t *testing.T) {
	fakeHermes(t, `
echo "boom: hermes no pudo arrancar" >&2
exit 1
`)

	if err := (Hermes{}).Connect(ConnectOptions{}); err == nil {
		t.Fatal("Connect debía propagar un exit distinto de cero")
	}
}
