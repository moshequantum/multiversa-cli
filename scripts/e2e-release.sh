#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

BIN="${MULTIVERSA_E2E_BINARY:-$WORK/multiversa}"
if [[ ! -x "$BIN" ]]; then
  go build \
    -ldflags "-X github.com/moshequantum/multiversa-cli/internal/version.Version=0.9.0-e2e -X github.com/moshequantum/multiversa-cli/internal/version.Commit=ci -X github.com/moshequantum/multiversa-cli/internal/version.Date=1970-01-01T00:00:00Z" \
    -o "$BIN" "$ROOT/cmd/multiversa"
fi

export HOME="$WORK/home"
mkdir -p "$HOME"

"$BIN" version --json > "$WORK/version.json"
"$BIN" capabilities --json > "$WORK/capabilities.json"
"$BIN" credits --json > "$WORK/credits.json"

python3 - "$BIN" "$WORK" <<'PY'
import json
import pathlib
import subprocess
import sys

binary = sys.argv[1]
work = pathlib.Path(sys.argv[2])

for name in ("version", "capabilities", "credits"):
    payload = json.loads((work / f"{name}.json").read_text())
    assert payload["ok"] is True, payload
    assert payload["schema"] == f"multiversa.{name}/v1", payload

p = subprocess.Popen(
    [binary, "mcp", "serve"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    text=True,
)
assert p.stdin is not None and p.stdout is not None

def send(message):
    p.stdin.write(json.dumps(message, separators=(",", ":")) + "\n")
    p.stdin.flush()

def response(expected_id):
    while True:
        line = p.stdout.readline()
        if not line:
            stderr = p.stderr.read() if p.stderr else ""
            raise AssertionError(f"MCP closed before response {expected_id}: {stderr}")
        message = json.loads(line)
        if message.get("id") == expected_id:
            return message

send({"jsonrpc":"2.0","id":1,"method":"initialize","params":{
    "protocolVersion":"2025-03-26","capabilities":{},
    "clientInfo":{"name":"multiversa-release-e2e","version":"1"}}})
init = response(1)
assert init["result"]["protocolVersion"] == "2025-03-26", init

send({"jsonrpc":"2.0","method":"notifications/initialized","params":{}})
send({"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}})
listed = response(2)
names = {tool["name"] for tool in listed["result"]["tools"]}
required = {"detect", "doctor", "status", "alerts", "version", "manifest", "credits", "updates"}
missing = sorted(required - names)
assert not missing, f"missing MCP tools: {missing}"

p.terminate()
p.wait(timeout=5)
print(f"release E2E passed: {binary} · {len(names)} MCP tools")
PY

