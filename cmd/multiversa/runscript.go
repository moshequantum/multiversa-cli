package main

import (
	"github.com/moshequantum/multiversa-cli/internal/embedded"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// readEmbeddedScript is a thin pass-through used by --show flags.
func readEmbeddedScript(name string) ([]byte, error) {
	return embedded.Script(name)
}

// runEmbeddedScript reads the named embedded script and delegates
// execution to xexec.RunEmbeddedScript, which owns the temp-file
// materialization and the bash invocation.
func runEmbeddedScript(name string) error {
	data, err := embedded.Script(name)
	if err != nil {
		return err
	}
	return xexec.RunEmbeddedScript(data)
}
