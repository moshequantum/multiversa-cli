---
name: browser-bridge
description: Inspect and operate a time-boxed, user-authorized Gmail, Hotmart, or ManyChat tab via Multiversa Browser Bridge.
---

# Multiversa Browser Bridge

Use this plugin only with the companion Chrome/Brave extension and a locally registered Native Messaging host.

## Safety contract

- The person manually opens the target page and authorizes that tab once for 30 minutes.
- Operate only supported HTTPS domains: Gmail, Hotmart, and ManyChat.
- Never request, read, copy, export, persist, or transmit cookies, browser profiles, session tokens, passwords, OTPs, or MFA codes.
- Treat the snapshot as sensitive operational context. It deliberately excludes input values, editable content, URLs query strings, and card-like values.
- The temporary session permits semantic inspection, reload, click, download and non-secret form fill on that specific origin.
- Do not use the bridge for checkout confirmation, deletion, permission changes, passwords, OTPs or MFA.

## Workflow

1. Ask the person once to open the authenticated tab and select **Activar control por 30 min**.
2. Check the session with `native-host/bin/multiversa-browser status`.
3. Use `exec snapshot`, `reload`, `wait`, `click`, `download`, or `fill` with visible semantic labels. For a same-origin link without an accessible label, use the fresh snapshot `id` together with its internal `href`; stale pairs fail closed.
4. Inspect each command result and retry only bounded, idempotent operations.
5. Keep account sessions local to the browser. Do not create scripts that copy profiles or use cookies/debugger permissions.

## Scope

The bridge executes allowlisted semantic actions against one authorized tab. It does not provide arbitrary JavaScript execution, cookie access, cross-origin navigation or secret-field entry.
