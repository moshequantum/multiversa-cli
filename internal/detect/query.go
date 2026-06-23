package detect

// RequiredMissing returns the subset of required tool names not
// installed according to the report.
func RequiredMissing(r Report, required []string) []string {
	have := make(map[string]bool, len(r.Tools))
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
