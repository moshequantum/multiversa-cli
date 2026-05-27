package detect

import (
	"bufio"
	"os"
	"runtime"
	"strings"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// OSInfo describes the host operating system and chosen package manager.
type OSInfo struct {
	Kind    string // "darwin" | "linux" | "windows"
	Arch    string // "amd64" | "arm64" | "386"
	Distro  string // "macos" | "ubuntu" | "debian" | "fedora" | "arch" | "windows" | "unknown"
	Version string // human-readable OS version
	PkgMgr  string // "brew" | "apt" | "dnf" | "pacman" | "winget" | "scoop" | "" (none detected)
}

func detectOS() OSInfo {
	info := OSInfo{
		Kind: runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	switch info.Kind {
	case "darwin":
		info.Distro = "macos"
		if r := xexec.Run("sw_vers", "-productVersion"); r.Err == nil {
			info.Version = strings.TrimSpace(r.LastLine())
		}
		if xexec.Check("brew") {
			info.PkgMgr = "brew"
		}
	case "linux":
		distro, version := parseOSRelease()
		info.Distro = distro
		info.Version = version
		info.PkgMgr = detectLinuxPkgMgr()
	case "windows":
		info.Distro = "windows"
		if r := xexec.Run("cmd", "/C", "ver"); r.Err == nil {
			info.Version = strings.TrimSpace(r.LastLine())
		}
		switch {
		case xexec.Check("winget"):
			info.PkgMgr = "winget"
		case xexec.Check("scoop"):
			info.PkgMgr = "scoop"
		case xexec.Check("choco"):
			info.PkgMgr = "choco"
		}
	default:
		info.Distro = "unknown"
	}
	return info
}

// parseOSRelease reads /etc/os-release on Linux and returns (ID, PRETTY_NAME).
// Returns ("unknown", "") if the file is missing.
func parseOSRelease() (distro, version string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown", ""
	}
	defer f.Close()
	fields := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := line[:eq]
		v := strings.Trim(line[eq+1:], `"`)
		fields[k] = v
	}
	distro = fields["ID"]
	if distro == "" {
		distro = "unknown"
	}
	version = fields["PRETTY_NAME"]
	return
}

// detectLinuxPkgMgr returns the first package manager binary found in PATH.
// Order reflects desktop-distro popularity, not policy.
func detectLinuxPkgMgr() string {
	for _, mgr := range []string{"apt", "dnf", "pacman", "zypper", "apk", "xbps-install"} {
		if xexec.Check(mgr) {
			return mgr
		}
	}
	return ""
}
