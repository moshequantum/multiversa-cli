// Package agentout renders machine-readable envelopes for AI agents.
//
// Every command that supports --json emits exactly one Envelope on stdout.
// The envelope is the stable contract between the CLI and any agentic
// consumer (Claude Code, Codex, gemini-cli, …): the schema field is
// versioned independently of the CLI version, and fields are only ever
// added, never renamed or removed, within the same schema version.
package agentout

import (
	"encoding/json"
	"io"

	"github.com/moshequantum/multiversa-cli/internal/version"
)

// Envelope is the single top-level JSON shape emitted by --json commands.
type Envelope struct {
	Schema     string `json:"schema"`      // "multiversa.<command>/v1"
	Command    string `json:"command"`     // command that produced this
	CLIVersion string `json:"cli_version"` // CLI semver at emit time
	OK         bool   `json:"ok"`
	Data       any    `json:"data,omitempty"`
	Error      *Error `json:"error,omitempty"`
}

// Error carries a machine-matchable code plus human context. Hint tells
// the agent what to try next — it is advisory, never an instruction to
// execute without user approval ("La IA propone, tú decides").
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Emit writes a success envelope for command with the given data payload.
func Emit(w io.Writer, command string, data any) error {
	return write(w, Envelope{
		Schema:     "multiversa." + command + "/v1",
		Command:    command,
		CLIVersion: version.Version,
		OK:         true,
		Data:       data,
	})
}

// EmitError writes a failure envelope. Callers still control the process
// exit code; this only shapes stdout.
func EmitError(w io.Writer, command, code, message, hint string) error {
	return write(w, Envelope{
		Schema:     "multiversa." + command + "/v1",
		Command:    command,
		CLIVersion: version.Version,
		OK:         false,
		Error:      &Error{Code: code, Message: message, Hint: hint},
	})
}

func write(w io.Writer, e Envelope) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(e)
}
