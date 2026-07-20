package version

var (
	Version = "v0.7.0"
	Commit  = "none"
	Date    = "unknown"
)

func Full() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
