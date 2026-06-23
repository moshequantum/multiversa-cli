// Multiversa workspace — MultiversaGroup private workspace setup.
//
// v0.4.0 unifies the UX behind the shared internal/tui primitives.
// The TUI model (WorkspaceModel) lives in internal/tui/workspace_model.go.
//
// Exit codes:
//
//	0  success (or --show, or non-TTY abort)
//	1  prerequisites missing OR script failure
//	2  user cancel (Esc or n at the confirm prompt)
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

const workspaceScript = "setup_multiversa.sh"

var userCancelErr = errors.New("user cancelled workspace setup")

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Configura el workspace privado MultiversaGroup (SSH, GPG, repos, bóveda).",
		Long: "Configura el workspace privado MultiversaGroup: llave SSH para\n" +
			"GitHub, llave GPG de firma, identidad git, clon del monorepo\n" +
			"privado, scaffolding de ~/.multiversa/, bóveda de secretos\n" +
			"cifrada.\n\n" +
			"El script de instalación viene embebido dentro del binario, así\n" +
			"que funciona en una máquina recién instalada sin necesidad de un\n" +
			"checkout de skills de Claude Code. Usa --show para imprimir el\n" +
			"cuerpo del script y salir.",
		RunE: func(cmd *cobra.Command, args []string) error {
			showOnly, _ := cmd.Flags().GetBool("show")
			err := runWorkspace(workspaceOpts{showOnly: showOnly, out: os.Stdout})
			if errors.Is(err, userCancelErr) {
				os.Exit(2)
			}
			return err
		},
	}
	cmd.Flags().Bool("show", false, "Imprime el cuerpo del script embebido y sale sin ejecutar.")
	return cmd
}

type workspaceOpts struct {
	showOnly bool
	out      io.Writer
}

func runWorkspace(opts workspaceOpts) error {
	if opts.out == nil {
		opts.out = os.Stdout
	}

	if opts.showOnly {
		data, err := readEmbeddedScript(workspaceScript)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(opts.out, string(data))
		return nil
	}

	report := detect.Run()
	missing := detect.RequiredMissing(report, []string{"git", "ssh"})

	if shouldRunWorkspaceTUI(opts) {
		return runWorkspaceTUI(report, missing)
	}
	return runWorkspaceNonInteractive(opts.out, report, missing)
}

func shouldRunWorkspaceTUI(opts workspaceOpts) bool {
	f, ok := opts.out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func runWorkspaceNonInteractive(out io.Writer, report detect.Report, missing []string) error {
	fmt.Fprintln(out, theme.Accent.Render("multiversa workspace"))
	fmt.Fprintln(out, theme.Dim.Render("MultiversaGroup — configuración del workspace privado"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, theme.Label.Render("script")+" "+workspaceScript+theme.Dim.Render(" (embebido)"))
	fmt.Fprintln(out, theme.Label.Render("hace")+"   "+theme.Body.Render("ssh-keygen · gpg --gen-key · git config · clone monorepo · ~/.multiversa init · bóveda"))
	fmt.Fprintln(out, theme.Label.Render("seguro")+" "+theme.Dim.Render("idempotente — al re-ejecutar se omiten pasos ya completados"))
	fmt.Fprintln(out)

	if len(missing) > 0 {
		fmt.Fprintln(out, theme.Warn.Render("Prerrequisitos faltantes: "+strings.Join(missing, ", ")))
		fmt.Fprintln(out, theme.Dim.Render("Ejecuta `multiversa stack --only=git` o tu gestor de paquetes primero."))
		return fmt.Errorf("prerrequisitos faltantes")
	}

	fmt.Fprint(out, theme.Label.Render("¿continuar? [y/N] "))
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil {
		fmt.Fprintln(out, theme.Dim.Render("cancelado"))
		return nil
	}
	if !tui.ConfirmDecision(ans) {
		fmt.Fprintln(out, theme.Dim.Render("cancelado"))
		return nil
	}

	return runEmbeddedScript(workspaceScript)
}

func runWorkspaceTUI(report detect.Report, missing []string) error {
	m := tui.NewWorkspaceModel(report, missing)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	fm, ok := final.(tui.WorkspaceModel)
	if !ok {
		return fmt.Errorf("workspace: unexpected model type %T", final)
	}
	if len(fm.MissingTools()) > 0 {
		fmt.Fprintln(os.Stderr, theme.Warn.Render("Prerrequisitos faltantes: "+strings.Join(fm.MissingTools(), ", ")))
		return fmt.Errorf("prerrequisitos faltantes")
	}
	if fm.Cancelled() {
		return userCancelErr
	}
	if !fm.Confirmed() {
		return userCancelErr
	}
	return runEmbeddedScript(workspaceScript)
}
