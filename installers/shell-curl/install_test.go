package shellcurl_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type installerHarness struct {
	t       *testing.T
	root    string
	home    string
	fakeBin string
	env     []string
	script  string
}

func newInstallerHarness(t *testing.T) *installerHarness {
	t.Helper()

	root := t.TempDir()
	h := &installerHarness{
		t:       t,
		root:    root,
		home:    filepath.Join(root, "home"),
		fakeBin: filepath.Join(root, "bin"),
		script:  "install.sh",
	}
	for _, dir := range []string{h.home, h.fakeBin} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	h.writeCommand("curl", `
for last; do :; done
printf '%s\n' "$*" >> "$INSTALLER_TEST_CURL_LOG"
printf 'fixture\n' > "$last"
`)
	h.writeCommand("sha256sum", "exit 0\n")
	h.writeCommand("tar", `
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-C" ]; then
    shift
    target=$1
    break
  fi
  shift
done
printf '#!/usr/bin/env sh\nexit 0\n' > "$target/multiversa"
chmod 0755 "$target/multiversa"
`)
	h.writeCommand("install", `
previous=
for last; do
  source=$previous
  previous=$last
done
cp "$source" "$last"
chmod 0755 "$last"
`)
	h.writeCommand("sudo", `
printf '%s\n' "$*" >> "$INSTALLER_TEST_SUDO_LOG"
exit 0
`)
	for _, command := range []string{"brew", "pipx", "pnpm"} {
		h.writeCommand(command, "exit 0\n")
	}

	h.env = append(os.Environ(),
		"HOME="+h.home,
		"PATH="+h.fakeBin+":/usr/bin:/bin",
		"MULTIVERSA_VERSION=v0.9.3",
		"INSTALLER_TEST_CURL_LOG="+filepath.Join(root, "curl.log"),
		"INSTALLER_TEST_SUDO_LOG="+filepath.Join(root, "sudo.log"),
	)
	return h
}

func (h *installerHarness) writeCommand(name, body string) {
	h.t.Helper()
	path := filepath.Join(h.fakeBin, name)
	if err := os.WriteFile(path, []byte("#!/usr/bin/env sh\nset -eu\n"+body), 0o755); err != nil {
		h.t.Fatal(err)
	}
}

func (h *installerHarness) run(extraEnv ...string) (string, error) {
	h.t.Helper()
	cmd := exec.Command("sh", h.script)
	cmd.Env = append(h.env, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInstallerDefaultsToUserTargetWithoutTTY(t *testing.T) {
	h := newInstallerHarness(t)

	output, err := h.run()
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	installed := filepath.Join(h.home, ".local", "bin", "multiversa")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("user-target binary missing at %s: %v\n%s", installed, err, output)
	}
	if data, _ := os.ReadFile(filepath.Join(h.root, "sudo.log")); len(data) != 0 {
		t.Fatalf("user install invoked sudo: %s", data)
	}
	if !strings.Contains(output, `export PATH="$HOME/.local/bin:$PATH"`) {
		t.Fatalf("PATH guidance missing:\n%s", output)
	}
}

func TestInstallerAssumeYesDefaultsToUserTarget(t *testing.T) {
	h := newInstallerHarness(t)

	output, err := h.run("MULTIVERSA_YES=1")
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	installed := filepath.Join(h.home, ".local", "bin", "multiversa")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("assume-yes target mismatch: %v\n%s", err, output)
	}
	if data, _ := os.ReadFile(filepath.Join(h.root, "sudo.log")); len(data) != 0 {
		t.Fatalf("assume-yes user install invoked sudo: %s", data)
	}
}

func TestInstallerHonorsInstallDirOverride(t *testing.T) {
	h := newInstallerHarness(t)
	customDir := filepath.Join(h.root, "custom-bin")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := h.run("MULTIVERSA_INSTALL_DIR=" + customDir)
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(customDir, "multiversa")); err != nil {
		t.Fatalf("override target not used: %v\n%s", err, output)
	}
	if strings.Contains(output, "¿Dónde quieres instalar") {
		t.Fatalf("override must skip target prompt:\n%s", output)
	}
}

func TestInstallerStopsBeforeDownloadWhenTargetWouldDuplicateBinary(t *testing.T) {
	h := newInstallerHarness(t)
	h.writeCommand("multiversa", "exit 0\n")
	existing := filepath.Join(h.fakeBin, "multiversa")

	output, err := h.run()
	if err == nil {
		t.Fatalf("installer should stop without confirmation:\n%s", output)
	}
	if !strings.Contains(output, "ya hay un multiversa en "+existing) {
		t.Fatalf("duplicate warning missing:\n%s", output)
	}
	if data, _ := os.ReadFile(filepath.Join(h.root, "curl.log")); len(data) != 0 {
		t.Fatalf("download started before duplicate confirmation: %s", data)
	}
}

func TestInstallerTTYOffersSystemTargetAndUsesSudo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the util-linux script flags used by this test are Linux-specific")
	}
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script command is unavailable")
	}
	h := newInstallerHarness(t)

	cmd := exec.Command("script", "-qfec", "sh install.sh", "/dev/null")
	cmd.Dir = "."
	cmd.Env = h.env
	cmd.Stdin = strings.NewReader("2\nn\nn\n")
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		t.Fatalf("interactive installer failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "¿Dónde quieres instalar") {
		t.Fatalf("target prompt missing:\n%s", output)
	}
	data, err := os.ReadFile(filepath.Join(h.root, "sudo.log"))
	if err != nil {
		t.Fatalf("system target did not invoke sudo: %v\n%s", err, output)
	}
	if !strings.Contains(string(data), "/usr/local/bin/multiversa") {
		t.Fatalf("sudo target mismatch: %s\n%s", data, output)
	}
}
