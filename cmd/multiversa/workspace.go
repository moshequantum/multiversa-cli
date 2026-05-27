package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

const workspaceScript = "setup_multiversa.sh"

// newWorkspaceCmd configures the MultiversaGroup private workspace:
// SSH key for GitHub, GPG signing key, git identity, private repo
// clone, ~/.multiversa/ scaffolding, encrypted secrets vault.
//
// The destructive parts (key generation, repo clone, vault create) all
// live in the bash skill script today. This command is the consultive
// entry point: it explains what will happen, asks for confirmation,
// and shells out. Future versions will port the steps natively.
func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Configure the MultiversaGroup private workspace (SSH, GPG, repos, vault).",
		Long: "Set up the private MultiversaGroup workspace: SSH key for\n" +
			"GitHub, GPG signing key, git identity, private monorepo clone,\n" +
			"~/.multiversa/ scaffolding, encrypted secrets vault.\n\n" +
			"The setup script is embedded inside the binary, so this works\n" +
			"on a freshly installed machine without any Claude Code skill\n" +
			"checkout. Use --show to print the script body and exit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			showOnly, _ := cmd.Flags().GetBool("show")
			return runWorkspace(showOnly)
		},
	}
	cmd.Flags().Bool("show", false, "Print the embedded script body without running it.")
	return cmd
}

func runWorkspace(showOnly bool) error {
	fmt.Println(theme.Accent.Render("multiversa workspace"))
	fmt.Println(theme.Dim.Render("MultiversaGroup — private workspace setup"))
	fmt.Println()

	if showOnly {
		data, err := readEmbeddedScript(workspaceScript)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(theme.Label.Render("script") + " " + workspaceScript + theme.Dim.Render(" (embedded)"))
	fmt.Println(theme.Label.Render("does") + "  " + theme.Body.Render("ssh-keygen · gpg --gen-key · git config · clone monorepo · ~/.multiversa init · vault"))
	fmt.Println(theme.Label.Render("safe") + "  " + theme.Dim.Render("idempotent — re-running skips steps already done"))
	fmt.Println()

	// Sanity-check the host: workspace setup requires git and ssh-keygen.
	report := detect.Run()
	missing := requiredMissing(report, []string{"git", "ssh"})
	if len(missing) > 0 {
		fmt.Println(theme.Warn.Render("Missing prerequisites: " + strings.Join(missing, ", ")))
		fmt.Println(theme.Dim.Render("Run `multiversa stack --only=git` or your package manager first."))
		return fmt.Errorf("prerequisites missing")
	}

	fmt.Print(theme.Label.Render("proceed? [y/N] "))
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil {
		fmt.Println(theme.Dim.Render("aborted"))
		return nil
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	if ans != "y" && ans != "yes" {
		fmt.Println(theme.Dim.Render("aborted"))
		return nil
	}

	// Execute the embedded script — streams stdin/stdout/stderr through.
	return runEmbeddedScript(workspaceScript)
}

func requiredMissing(r detect.Report, required []string) []string {
	have := map[string]bool{}
	for _, t := range r.Tools {
		if t.Installed {
			have[t.Name] = true
		}
	}
	var missing []string
	for _, req := range required {
		if !have[req] {
			missing = append(missing, req)
		}
	}
	return missing
}
