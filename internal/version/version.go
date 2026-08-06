package version

var (
	// Untagged builds must never impersonate an official release. GoReleaser
	// replaces all three values for signed/tagged artifacts.
	Version = "v0.9.0-dev"
	Commit  = "none"
	Date    = "unknown"
)

func Full() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
