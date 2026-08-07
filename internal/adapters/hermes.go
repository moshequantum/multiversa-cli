package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Hermes connects Nous Research's optional Hermes Agent runtime to the
// read-only Multiversa MCP server. The user must explicitly run
// `multiversa connect hermes`; detection alone never changes Hermes config.
type Hermes struct{}

func (Hermes) ID() string          { return "hermes" }
func (Hermes) DisplayName() string { return "Hermes Agent" }

func (Hermes) Detect() bool {
	if _, err := osexec.LookPath("hermes"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	fi, err := os.Stat(filepath.Join(home, ".hermes", "hermes-agent"))
	return err == nil && fi.IsDir()
}

func (Hermes) Connect(opts ConnectOptions) error {
	if _, err := osexec.LookPath("hermes"); err != nil {
		return fmt.Errorf("Hermes Agent no está disponible en PATH")
	}
	exe, err := os.Executable()
	if err != nil || exe == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("resolver binario Multiversa: %w", err)
		}
		exe = filepath.Join(home, ".local", "bin", "multiversa")
	}
	// `hermes mcp add` pregunta "Enable all N tools?" antes de escribir. Sin
	// terminal el prompt lee EOF, hermes imprime "Cancelled." y SALE CON 0,
	// así que responder por stdin es necesario pero no suficiente.
	if _, err := runHermes("y\n", "mcp", "add", "multiversa", "--command", exe, "--args", "mcp", "serve"); err != nil {
		return fmt.Errorf("hermes mcp add: %w", err)
	}

	// Un exit 0 no prueba registro. Se verifica contra el inventario del
	// propio Hermes antes de declarar éxito: reportar una conexión que no
	// existe es peor que fallar.
	registered, err := hermesHasServer("multiversa")
	if err != nil {
		return err
	}
	if !registered {
		return fmt.Errorf(
			"hermes aceptó el comando pero no registró 'multiversa'; ejecútalo en una terminal interactiva:\n"+
				"  hermes mcp add multiversa --command %s --args mcp serve", exe)
	}
	return nil
}

// hermesHasServer consulta el inventario de MCP de Hermes. Es la única fuente
// de verdad: el código de salida de `mcp add` no distingue registro de
// cancelación.
func hermesHasServer(name string) (bool, error) {
	out, err := runHermes("", "mcp", "list")
	if err != nil {
		return false, fmt.Errorf("hermes mcp list: %w", err)
	}
	return strings.Contains(out, name), nil
}

func (Hermes) Disconnect() error {
	if _, err := osexec.LookPath("hermes"); err != nil {
		return nil
	}
	if _, err := runHermes("y\n", "mcp", "remove", "multiversa"); err != nil {
		return fmt.Errorf("hermes mcp remove: %w", err)
	}
	return nil
}

// runHermes ejecuta hermes con stdin fijado y devuelve su salida combinada.
// El stdin importa: varios subcomandos piden confirmación y, sin ella, se
// cancelan silenciosamente devolviendo 0.
func runHermes(stdin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := osexec.CommandContext(ctx, "hermes", args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		last := ""
		if len(lines) > 0 {
			last = lines[len(lines)-1]
		}
		return out.String(), fmt.Errorf("%w: %s", err, last)
	}
	return out.String(), nil
}
