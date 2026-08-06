#!/usr/bin/env python3
"""Cheap static guardrails for the MV3 browser bridge manifest and sources."""
from __future__ import annotations

import json
import sys
from pathlib import Path

root = Path(sys.argv[1] if len(sys.argv) > 1 else "browser-extension")
manifest_path = root / "manifest.json"
manifest = json.loads(manifest_path.read_text())
errors: list[str] = []

if manifest.get("manifest_version") != 3:
    errors.append("manifest_version must be 3")
expected = {"activeTab", "scripting", "downloads", "nativeMessaging", "storage", "alarms"}
permissions = set(manifest.get("permissions", []))
if permissions != expected:
    errors.append(f"permissions must be exactly {sorted(expected)}, got {sorted(permissions)}")
for forbidden in ("cookies", "debugger", "tabs", "webRequest"):
    if forbidden in permissions:
        errors.append(f"forbidden permission: {forbidden}")
for key in ("host_permissions", "optional_host_permissions"):
    values = manifest.get(key, [])
    if "<all_urls>" in values:
        errors.append(f"{key} must not include <all_urls>")
for source in root.rglob("*.js"):
    contents = source.read_text()
    if "chrome.cookies" in contents or "chrome.debugger" in contents:
        errors.append(f"forbidden API reference in {source}")

if errors:
    print("Extension validation failed:", *[f"- {error}" for error in errors], sep="\n")
    raise SystemExit(1)
print(f"Extension validation passed: {root}")
