package version

var (
	Version = "v0.5.0"
	Commit  = "none"
	Date    = "unknown"
)

func Full() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
