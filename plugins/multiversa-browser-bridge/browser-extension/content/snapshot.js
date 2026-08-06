(() => {
  const SNAPSHOT_VERSION = "0.1.1";
  if (window.__multiversaBrowserBridgeSnapshotVersion === SNAPSHOT_VERSION) return;
  window.__multiversaBrowserBridgeSnapshotVersion = SNAPSHOT_VERSION;

  const MAX_TEXT = 5000;
  const MAX_ITEMS = 80;
  const REDACTED = "[redactado]";
  const SENSITIVE_SELECTOR = [
    "input",
    "textarea",
    "[contenteditable='true']",
    "[role='textbox']",
    "[role='combobox']",
    "[data-lexical-editor='true']"
  ].join(",");

  const normalize = (value) => String(value || "").replace(/\s+/g, " ").trim();
  const truncate = (value, length) => {
    const normalized = normalize(value);
    return normalized.length > length ? `${normalized.slice(0, length - 1)}…` : normalized;
  };

  function redact(value) {
    return String(value || "")
      // Email and phone values are useful identifiers but not required for navigation.
      .replace(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi, REDACTED)
      .replace(/(?:\+?\d[\d().\s-]{7,}\d)/g, REDACTED)
      // Card-like strings are never included even if rendered on a checkout.
      .replace(/\b(?:\d[ -]*?){13,19}\b/g, REDACTED)
      // API keys, request signatures and similar opaque hexadecimal secrets.
      .replace(/\b[a-f0-9]{32,}\b/gi, REDACTED);
  }

  function visible(element) {
    const style = window.getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
  }

  function elementLabel(element) {
    const aria = element.getAttribute("aria-label");
    const title = element.getAttribute("title");
    const editable = element.matches(SENSITIVE_SELECTOR);
    const text = editable ? "" : (element.innerText || element.textContent);
    const placeholder = editable ? element.getAttribute("placeholder") : "";
    return truncate(redact(aria || title || placeholder || text || element.name || element.id), 180);
  }

  function safeBodyText() {
    const clone = document.body.cloneNode(true);
    clone.querySelectorAll(SENSITIVE_SELECTOR).forEach((element) => element.remove());
    clone.querySelectorAll("script, style, noscript, svg, [aria-hidden='true']").forEach((element) => element.remove());
    return truncate(redact(clone.innerText || clone.textContent), MAX_TEXT);
  }

  function interactiveElements() {
    const selector = "a[href], button, input, select, textarea, [role='button'], [role='link'], [role='checkbox'], [role='tab']";
    return [...document.querySelectorAll(selector)]
      .filter(visible)
      .slice(0, MAX_ITEMS)
      .map((element, index) => ({
        id: index + 1,
        tag: element.tagName.toLowerCase(),
        role: element.getAttribute("role") || undefined,
        type: element.getAttribute("type") || undefined,
        label: elementLabel(element) || "Sin etiqueta",
        disabled: Boolean(element.disabled || element.getAttribute("aria-disabled") === "true"),
        // Hrefs can contain opaque tokens; capture only same-origin pathname.
        href: element.href ? (() => {
          try {
            const href = new URL(element.href);
            return href.origin === location.origin ? href.pathname : href.origin;
          } catch { return undefined; }
        })() : undefined
      }));
  }

  function snapshot() {
    const url = new URL(location.href);
    return {
      schema: "multiversa.browser.snapshot/v1",
      capturedAt: new Date().toISOString(),
      page: {
        origin: url.origin,
        path: url.pathname,
        title: truncate(redact(document.title), 300)
      },
      text: safeBodyText(),
      interactive: interactiveElements(),
      privacy: {
        browserSessionData: "not-collected",
        inputValues: "not-collected",
        contentEditable: "not-collected",
        queryString: "not-collected"
      }
    };
  }

  chrome.runtime.onMessage.addListener((message, _sender, respond) => {
    if (message?.type !== "MULTIVERSA_CAPTURE") return;
    try {
      respond({ ok: true, data: snapshot() });
    } catch (error) {
      respond({ ok: false, error: error instanceof Error ? error.message : String(error) });
    }
  });
})();
