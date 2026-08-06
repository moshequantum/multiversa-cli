(() => {
  const CONTROLLER_VERSION = "0.2.5";
  if (window.__multiversaBrowserBridgeControllerVersion === CONTROLLER_VERSION) return;
  window.__multiversaBrowserBridgeControllerVersion = CONTROLLER_VERSION;

  const SELECTOR = [
    "a[href]",
    "button",
    "input",
    "select",
    "textarea",
    "[role='button']",
    "[role='link']",
    "[role='checkbox']",
    "[role='tab']",
    "[role='textbox']",
    "[contenteditable='true']"
  ].join(",");
  // Keep this selector and limit aligned with snapshot.js so an ephemeral
  // snapshot reference resolves to the same visible element.
  const SNAPSHOT_SELECTOR = [
    "a[href]",
    "button",
    "input",
    "select",
    "textarea",
    "[role='button']",
    "[role='link']",
    "[role='checkbox']",
    "[role='tab']"
  ].join(",");
  const MAX_SNAPSHOT_ITEMS = 80;

  const normalize = (value) => String(value || "").replace(/\s+/g, " ").trim();
  const folded = (value) => normalize(value).toLocaleLowerCase("es");

  function visible(element) {
    const style = window.getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" &&
      Number(style.opacity || "1") !== 0 && rect.width > 0 && rect.height > 0;
  }

  function snapshotVisible(element) {
    const style = window.getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" &&
      rect.width > 0 && rect.height > 0;
  }

  function label(element) {
    const editable = element.matches(
      "input, textarea, select, [contenteditable='true'], [role='textbox'], [role='combobox']"
    );
    return normalize(
      element.getAttribute("aria-label") ||
      element.getAttribute("title") ||
      (editable ? element.getAttribute("placeholder") : "") ||
      (editable ? "" : element.innerText) ||
      (editable ? "" : element.textContent) ||
      element.name ||
      element.id
    ).slice(0, 300);
  }

  function disabled(element) {
    return Boolean(
      element.disabled ||
      element.getAttribute("aria-disabled") === "true" ||
      element.getAttribute("data-disabled") === "true"
    );
  }

  function referencedCandidate(payload) {
    const id = Number(payload.id);
    if (!Number.isInteger(id) || id < 1 || id > MAX_SNAPSHOT_ITEMS) {
      throw new Error("La referencia del snapshot no es válida.");
    }
    if (typeof payload.href !== "string" || !payload.href.startsWith("/")) {
      throw new Error("La referencia sin etiqueta necesita la ruta interna observada.");
    }
    const element = [...document.querySelectorAll(SNAPSHOT_SELECTOR)]
      .filter(snapshotVisible)
      .slice(0, MAX_SNAPSHOT_ITEMS)[id - 1];
    if (!element) throw new Error(`La referencia ${id} ya no está disponible.`);
    if (!(element instanceof HTMLAnchorElement) || !element.href) {
      throw new Error("La referencia sin etiqueta debe apuntar a un enlace interno.");
    }
    const actual = new URL(element.href, location.href);
    if (actual.origin !== location.origin || actual.pathname !== payload.href) {
      throw new Error("La página cambió desde el snapshot; toma una vista nueva antes de continuar.");
    }
    return element;
  }

  function candidates(payload) {
    const wanted = folded(payload.label);
    if (!wanted) return [referencedCandidate(payload)];
    const exact = payload.exact !== false;
    return [...document.querySelectorAll(SELECTOR)]
      .filter(visible)
      .filter((element) => {
        const current = folded(label(element));
        return exact ? current === wanted : current.includes(wanted);
      });
  }

  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

  // Brave may suspend an MV3 service worker even while a native port is open.
  // A message from the already-authorized content script is a normal extension
  // event and keeps the in-memory, time-boxed control session alive.
  const keepAlive = () => {
    chrome.runtime.sendMessage({ type: "MULTIVERSA_CONTROL_KEEPALIVE" }).catch(() => {});
  };
  keepAlive();
  window.setInterval(keepAlive, 15000);

  async function findEnabled(payload) {
    const timeout = Math.min(Math.max(Number(payload.waitForEnabledMs || 0), 0), 30000);
    const deadline = Date.now() + timeout;
    const description = folded(payload.label)
      ? `“${payload.label}”`
      : `la referencia ${payload.id}`;
    do {
      const matches = candidates(payload);
      const enabled = matches.find((element) => !disabled(element));
      if (enabled) return enabled;
      if (matches.length && Date.now() >= deadline) {
        throw new Error(`${description} está visible, pero continúa deshabilitada.`);
      }
      if (!matches.length && Date.now() >= deadline) {
        throw new Error(`No encontré ${description} en la pestaña autorizada.`);
      }
      await sleep(250);
    } while (true);
  }

  async function click(payload) {
    const element = await findEnabled(payload);
    element.scrollIntoView({ block: "center", inline: "center", behavior: "instant" });
    element.focus({ preventScroll: true });
    element.click();
    return {
      action: "click",
      label: label(element) || "Sin etiqueta",
      tag: element.tagName.toLowerCase()
    };
  }

  async function download(payload, sessionId) {
    const element = await findEnabled(payload);
    if (!(element instanceof HTMLAnchorElement) || !element.href) {
      // Native Gmail attachment controls are buttons rather than links.
      element.scrollIntoView({ block: "center", inline: "center", behavior: "instant" });
      element.click();
      return { action: "download", label: label(element), browserOwned: true };
    }
    let href = element.href;
    try {
      const url = new URL(href);
      if (url.hostname === "drive.google.com") {
        const pathId = url.pathname.match(/\/file\/(?:u\/\d+\/)?d\/([^/]+)/)?.[1];
        const fileId = pathId || url.searchParams.get("id");
        if (fileId) {
          // Resolve the Drive viewer entirely inside the extension. The CLI
          // receives neither this URL nor browser credentials.
          href = `https://drive.usercontent.google.com/u/0/uc?id=${encodeURIComponent(fileId)}&export=download`;
        }
      }
    } catch {
      throw new Error("El enlace de descarga no es válido.");
    }
    const response = await chrome.runtime.sendMessage({
      type: "MULTIVERSA_INTERNAL_DOWNLOAD",
      sessionId,
      href,
      filename: payload.filename || label(element)
    });
    if (!response?.ok) throw new Error(response?.error || "El navegador rechazó la descarga.");
    return {
      action: "download",
      label: label(element),
      browserOwned: true,
      downloadId: response.downloadId
    };
  }

  async function fill(payload) {
    const element = await findEnabled(payload);
    if (element.matches("input[type='password']")) {
      throw new Error("El puente no escribe contraseñas.");
    }
    if (typeof payload.value !== "string" || payload.value.length > 10000) {
      throw new Error("El valor de formulario no es válido.");
    }
    element.scrollIntoView({ block: "center", inline: "center", behavior: "instant" });
    element.focus({ preventScroll: true });
    if (element.isContentEditable) {
      element.textContent = payload.value;
    } else {
      const proto = element instanceof HTMLTextAreaElement
        ? HTMLTextAreaElement.prototype
        : HTMLInputElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, "value")?.set;
      if (!setter) throw new Error("Ese campo no admite escritura segura.");
      setter.call(element, payload.value);
    }
    element.dispatchEvent(new InputEvent("input", { bubbles: true, inputType: "insertText", data: null }));
    element.dispatchEvent(new Event("change", { bubbles: true }));
    return { action: "fill", label: label(element), written: true };
  }

  chrome.runtime.onMessage.addListener((message, _sender, respond) => {
    if (message?.type !== "MULTIVERSA_CONTROL_ACTION") return;
    const action = message.action;
    const payload = message.payload || {};
    const task = action === "click"
      ? click(payload)
      : action === "download"
        ? download(payload, message.sessionId)
      : action === "fill"
        ? fill(payload)
        : Promise.reject(new Error(`Acción DOM no admitida: ${action}`));
    task.then(
      (result) => respond({ ok: true, result }),
      (error) => respond({ ok: false, error: error instanceof Error ? error.message : String(error) })
    );
    return true;
  });
})();
