// Package upstream checks the latest published releases of the curated
// engines (and the CLI itself) against what is installed locally.
//
// Read-only by design: it reports what is outdated, it never updates.
// The update itself is always a human decision — la IA propone, tú decides.
package upstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Component is one watched project: a curated engine or the CLI itself.
type Component struct {
	ID        string // registry id, e.g. "engram"
	Name      string // display name
	Repo      string // https://github.com/{owner}/{name}
	Installed string // locally detected version, "" if unknown/not installed
}

// ReleaseInfo is the per-component result of a release check. It is the
// unit of the multiversa.updates/v1 payload.
type ReleaseInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Repo            string `json:"repo"`
	Installed       string `json:"installed,omitempty"`
	Latest          string `json:"latest,omitempty"`
	ReleaseURL      string `json:"release_url,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	Error           string `json:"error,omitempty"` // per-component failure, never fatal
}

// githubRelease is the subset of the GitHub releases API we consume.
type githubRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

// Check queries the latest release of every component in parallel.
// A failing component reports its own Error; the check never aborts —
// partial data beats none, same posture as detect.
func Check(components []Component, timeout time.Duration) []ReleaseInfo {
	client := &http.Client{Timeout: timeout}
	results := make([]ReleaseInfo, len(components))

	var wg sync.WaitGroup
	for i, c := range components {
		wg.Add(1)
		go func(i int, c Component) {
			defer wg.Done()
			results[i] = checkOne(client, c)
		}(i, c)
	}
	wg.Wait()
	return results
}

func checkOne(client *http.Client, c Component) ReleaseInfo {
	info := ReleaseInfo{ID: c.ID, Name: c.Name, Repo: c.Repo, Installed: c.Installed}

	path, ok := strings.CutPrefix(c.Repo, "https://github.com/")
	if !ok {
		info.Error = "repo no soportado (solo GitHub): " + c.Repo
		return info
	}
	url := "https://api.github.com/repos/" + strings.TrimSuffix(path, "/") + "/releases/latest"

	resp, err := client.Get(url)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to decode
	case http.StatusNotFound:
		info.Error = "sin releases publicados"
		return info
	case http.StatusForbidden, http.StatusTooManyRequests:
		info.Error = "rate limit de la API de GitHub — reintenta más tarde"
		return info
	default:
		info.Error = fmt.Sprintf("GitHub API HTTP %d", resp.StatusCode)
		return info
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		info.Error = "respuesta inválida de GitHub: " + err.Error()
		return info
	}

	info.Latest = rel.TagName
	info.ReleaseURL = rel.HTMLURL
	info.PublishedAt = rel.PublishedAt
	info.UpdateAvailable = UpdateAvailable(c.Installed, rel.TagName)
	return info
}

// UpdateAvailable reports whether latest is a different version than
// installed. Unknown or non-numeric installed versions (e.g. the bare
// "instalado" marker) never claim an update — the report shows the
// latest release and lets the human judge.
func UpdateAvailable(installed, latest string) bool {
	ni, nl := Normalize(installed), Normalize(latest)
	if ni == "" || nl == "" || ni[0] < '0' || ni[0] > '9' {
		return false
	}
	return ni != nl
}

// Normalize strips the leading "v" and surrounding noise so "v0.4.0",
// "0.4.0" and "multiversa v0.4.0 (abc, date)" compare as equals when
// they carry the same semver core.
func Normalize(v string) string {
	v = strings.TrimSpace(v)
	// Keep only the first token that looks like a version, dropping any
	// leading alpha prefix ("v0.4.0", "go1.26.5", "release-2.1").
	for _, tok := range strings.Fields(v) {
		t := strings.TrimLeftFunc(strings.ToLower(tok), func(r rune) bool {
			return (r >= 'a' && r <= 'z') || r == '-' || r == '_'
		})
		if len(t) > 0 && t[0] >= '0' && t[0] <= '9' {
			// Trim trailing punctuation like "(", ",".
			t = strings.TrimFunc(t, func(r rune) bool {
				return !(r >= '0' && r <= '9') && r != '.' && !(r >= 'a' && r <= 'z') && r != '-'
			})
			return t
		}
	}
	return strings.TrimPrefix(strings.ToLower(v), "v")
}
