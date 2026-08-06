# Multiversa Browser Bridge

A local-first control bridge from an already authenticated Chromium-family browser tab to Multiversa. One click authorizes a specific Gmail, Hotmart or ManyChat tab for 30 minutes; after that the CLI can inspect, reload, click, download and fill labeled controls without making the person an intermediary.

## Security boundary

The extension requests exactly:

- `activeTab` — temporary access only after the person invokes the extension on the active tab.
- `scripting` — injects the snapshot and semantic-action scripts only into that authorized tab.
- `downloads` — supports browser-owned downloads; the native host never fetches authenticated URLs.
- `nativeMessaging` — keeps a local, time-boxed command channel open.

It does **not** request `cookies`, `debugger`, `tabs`, broad host access, or `<all_urls>`.

The snapshot deliberately omits cookies, query strings, input values and content-editable values; it redacts email addresses, phone numbers and card-like strings from visible text.

## Load unpacked

### Chrome or Brave (including the current Linux Snap installation)

1. Open `brave://extensions` (or `chrome://extensions`).
2. Turn on **Developer mode**.
3. Select **Load unpacked** and choose `browser-extension/`.
4. Pin **Multiversa Browser Bridge**.
5. Open an authenticated Gmail, Hotmart, or ManyChat page, press the extension icon and select **Activar control por 30 min**.

Brave installed as a Snap still loads unpacked extensions through `brave://extensions`; its profile path (for example `~/snap/brave/661/.config/BraveSoftware/Brave-Browser`) must never be copied into this project.

## Native Messaging host

The control button expects a local host named `com.multiversa.browser_bridge`. The host is intentionally not auto-installed by the extension and must be registered only after its executable and extension ID are known.

Native host manifests belong in:

- Chrome Linux: `~/.config/google-chrome/NativeMessagingHosts/`
- Chromium Linux: `~/.config/chromium/NativeMessagingHosts/`
- Brave Snap: `~/snap/brave/current/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts/`

The companion host uses Chrome Native Messaging framing (a 32-bit message length followed by UTF-8 JSON), not JSON-lines. It relays allowlisted semantic commands from a filesystem queue and must never request browser credentials.

## CLI control

After the 30-minute authorization:

```bash
BRIDGE=/home/moshe/Documentos/Multiversa/multiversa-browser-bridge/native-host/bin/multiversa-browser
"$BRIDGE" status
"$BRIDGE" exec snapshot
"$BRIDGE" exec reload '{"timeoutMs":20000}'
"$BRIDGE" exec click '{"label":"Descargar todos los archivos adjuntos","exact":true,"waitForEnabledMs":30000}'
```

When a framework renders a same-origin link without an accessible label, use
the fresh `id` and same-origin `href` returned by `snapshot` together. The pair
is validated against the current DOM, so a stale snapshot fails closed:

```bash
"$BRIDGE" exec click '{"id":27,"href":"/account/cms/files/example"}'
```

Commands are serialized and scoped to the authorized origin. The extension rejects cross-origin navigation and password fields.

## Validate

```bash
python3 /home/moshe/.codex/skills/.system/plugin-creator/scripts/validate_plugin.py .
python3 scripts/validate_extension.py plugins/multiversa-browser-bridge/browser-extension
```
