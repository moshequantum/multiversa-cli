package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// newUSBCmd is the entry point for encrypted bootable USB lab
// creation. The operation is destructive (it wipes the target device
// and writes a LUKS container) so the contract is: read the host,
// explain what will happen, require typed-device confirmation, then
// delegate to the platform-specific bash script.
//
// A full native Bubble Tea wizard is planned, but the gate ergonomics
// matter more than the UI — typed device confirmation prevents the
// single most expensive mistake (wrong /dev/sdX) and that part is
// already correct in the bash scripts.
func newUSBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usb",
		Short: "Build an encrypted bootable USB lab (LUKS on Linux, guided VeraCrypt/balenaEtcher on macOS).",
		Long: "Create an encrypted, bootable USB lab — full disk LUKS on\n" +
			"Linux, guided VeraCrypt + balenaEtcher flow on macOS.\n\n" +
			"This is destructive: the target device will be wiped. The flow\n" +
			"will ask you to type the device path twice before any wipe or\n" +
			"`cryptsetup luksFormat` call.\n\n" +
			"The platform-specific script is embedded inside the binary, so\n" +
			"this works on a fresh machine without any Claude Code skill\n" +
			"checkout. Use --show to print the script body and exit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			showOnly, _ := cmd.Flags().GetBool("show")
			return runUSB(showOnly)
		},
	}
	cmd.Flags().Bool("show", false, "Print the embedded script body without running it.")
	return cmd
}

func runUSB(showOnly bool) error {
	fmt.Println(theme.Accent.Render("multiversa usb"))
	fmt.Println()

	report := detect.Run()

	var scriptName string
	switch report.OS.Kind {
	case "linux":
		scriptName = "encrypted_usb_linux.sh"
	case "darwin":
		scriptName = "encrypted_usb_macos.sh"
	case "windows":
		fmt.Println(theme.Warn.Render("USB encryption from Windows is not supported yet."))
		fmt.Println(theme.Dim.Render("Boot from a Linux live ISO and re-run `multiversa usb`."))
		return nil
	default:
		return fmt.Errorf("unsupported OS for usb command: %s", report.OS.Kind)
	}

	if showOnly {
		data, err := readEmbeddedScript(scriptName)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(theme.Label.Render("host") + "  " + theme.Body.Render(fmt.Sprintf("%s/%s · %s", report.OS.Kind, report.OS.Arch, report.OS.Distro)))
	fmt.Println(theme.Label.Render("script") + " " + scriptName + theme.Dim.Render(" (embedded)"))
	fmt.Println()
	fmt.Println(theme.Warn.Render("⚠ destructive — wipes the target device. Have the device path ready (e.g. /dev/sdb on Linux, disk4 on macOS)."))
	fmt.Println(theme.Dim.Render("The script will ask you to confirm the device path twice before any write."))
	fmt.Println()

	// Prerequisites by platform.
	required := requiredForUSB(report.OS.Kind)
	missing := requiredMissing(report, required)
	if len(missing) > 0 {
		fmt.Println(theme.Warn.Render("Missing prerequisites: " + strings.Join(missing, ", ")))
		switch report.OS.Kind {
		case "linux":
			fmt.Println(theme.Dim.Render("Install with: sudo " + report.OS.PkgMgr + " install cryptsetup"))
		case "darwin":
			fmt.Println(theme.Dim.Render("Install with: brew install --cask veracrypt balenaetcher"))
		}
		return fmt.Errorf("prerequisites missing")
	}

	fmt.Print(theme.Label.Render("continue with the encrypted USB flow? [y/N] "))
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

	return runEmbeddedScript(scriptName)
}

func requiredForUSB(osKind string) []string {
	switch osKind {
	case "linux":
		return []string{"cryptsetup"}
	case "darwin":
		// We don't probe veracrypt/balenaEtcher because the user may
		// run them via Applications; the script handles those checks.
		return nil
	}
	return nil
}
