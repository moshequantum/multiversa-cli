const NATIVE_HOST_NAME = "com.multiversa.browser_bridge";
const CONTROL_TTL_MS = 30 * 60 * 1000;
const CONTROL_STORAGE_KEY = "multiversaControlSession";
const CONTROL_ALARM = "multiversa-control-keepalive";
const ALLOWED_HOSTS = ["mail.google.com", "app.hotmart.com", "hotmart.com", "manychat.com"];
const CONTROL_ACTIONS = new Set(["snapshot", "reload", "wait", "click", "download", "fill"]);
const DOWNLOAD_HOSTS = [
  "mail.google.com",
  "drive.google.com",
  "googleusercontent.com",
  "usercontent.google.com"
];

let controlPort = null;
let controlSession = null;
const pendingNative = new Map();

function isAllowedURL(rawURL) {
  try {
    const { hostname, protocol } = new URL(rawURL);
    return protocol === "https:" && ALLOWED_HOSTS.some(
      (host) => hostname === host || hostname.endsWith(`.${host}`)
    );
  } catch {
    return false;
  }
}

function friendlyDomain(rawURL) {
  try { return new URL(rawURL).hostname; } catch { return "esta pestaña"; }
}

function isAllowedDownloadURL(rawURL) {
  try {
    const { hostname, protocol } = new URL(rawURL);
    return protocol === "https:" && DOWNLOAD_HOSTS.some(
      (host) => hostname === host || hostname.endsWith(`.${host}`)
    );
  } catch {
    return false;
  }
}

function safeDownloadName(value) {
  const cleaned = String(value || "archivo")
    .normalize("NFC")
    .replace(/[\\/:*?"<>|]/g, "-")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 100)
    .replace(/[. ]+$/g, "");
  return cleaned || "archivo";
}

async function activeTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id || !tab.url) throw new Error("No hay una pestaña activa disponible.");
  if (!isAllowedURL(tab.url)) {
    throw new Error(`${friendlyDomain(tab.url)} no está habilitado. Abre Gmail, Hotmart o ManyChat.`);
  }
  return tab;
}

async function inject(tabId) {
  await chrome.scripting.executeScript({
    target: { tabId },
    files: ["content/snapshot.js", "content/controller.js"]
  });
}

async function captureSnapshot(tabId = null) {
  const tab = tabId ? await chrome.tabs.get(tabId) : await activeTab();
  if (!tab?.id || !tab.url || !isAllowedURL(tab.url)) throw new Error("La pestaña ya no está autorizada.");
  await inject(tab.id);
  const snapshot = await chrome.tabs.sendMessage(tab.id, { type: "MULTIVERSA_CAPTURE" });
  if (!snapshot?.ok) throw new Error(snapshot?.error || "No se pudo leer la pestaña activa.");
  return snapshot.data;
}

function oneShotNative(snapshot) {
  return new Promise((resolve, reject) => {
    let port;
    try { port = chrome.runtime.connectNative(NATIVE_HOST_NAME); }
    catch (error) { reject(new Error(`No se pudo abrir el puente local: ${error.message}`)); return; }
    const timeout = setTimeout(() => {
      port.disconnect();
      reject(new Error("El puente local no respondió a tiempo."));
    }, 5000);
    port.onMessage.addListener((response) => {
      clearTimeout(timeout);
      port.disconnect();
      if (response?.ok !== true) {
        reject(new Error(response?.error?.message || "El puente local rechazó la vista segura."));
        return;
      }
      resolve(response);
    });
    port.onDisconnect.addListener(() => {
      if (chrome.runtime.lastError) {
        clearTimeout(timeout);
        reject(new Error(chrome.runtime.lastError.message));
      }
    });
    port.postMessage({
      version: 1,
      requestId: `snapshot-${crypto.randomUUID()}`,
      action: "snapshot",
      target: `${snapshot.page.origin}${snapshot.page.path}`,
      payload: { snapshot }
    });
  });
}

function postPersistent(message, timeoutMs = 5000) {
  return new Promise((resolve, reject) => {
    if (!controlPort) {
      reject(new Error("El canal de control no está conectado."));
      return;
    }
    const timeout = setTimeout(() => {
      pendingNative.delete(message.requestId);
      reject(new Error("El puente local no respondió a tiempo."));
    }, timeoutMs);
    pendingNative.set(message.requestId, { resolve, reject, timeout });
    controlPort.postMessage(message);
  });
}

function rejectPending(message) {
  for (const { reject, timeout } of pendingNative.values()) {
    clearTimeout(timeout);
    reject(new Error(message));
  }
  pendingNative.clear();
}

async function waitForTab(tabId, timeoutMs = 20000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const tab = await chrome.tabs.get(tabId);
    if (tab.status === "complete") return tab;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error("La pestaña no terminó de cargar a tiempo.");
}

async function executeCommand(command) {
  if (!controlSession || Date.now() >= controlSession.expiresAt) {
    throw new Error("La autorización temporal expiró.");
  }
  if (!CONTROL_ACTIONS.has(command.action)) throw new Error("Acción no permitida.");
  const tab = await chrome.tabs.get(controlSession.tabId);
  if (!tab?.url || new URL(tab.url).origin !== controlSession.origin) {
    throw new Error("La pestaña cambió de origen y perdió la autorización.");
  }
  if (command.targetOrigin && command.targetOrigin !== controlSession.origin) {
    throw new Error("La orden no corresponde al origen autorizado.");
  }
  const payload = command.payload || {};
  if (command.action === "snapshot") {
    return { action: "snapshot", snapshot: await captureSnapshot(tab.id) };
  }
  if (command.action === "reload") {
    await chrome.tabs.reload(tab.id, { bypassCache: Boolean(payload.bypassCache) });
    await waitForTab(tab.id, Math.min(Number(payload.timeoutMs || 20000), 30000));
    await inject(tab.id);
    return { action: "reload", loaded: true, url: (await chrome.tabs.get(tab.id)).url };
  }
  if (command.action === "wait") {
    const milliseconds = Math.min(Math.max(Number(payload.milliseconds || 1000), 0), 30000);
    await new Promise((resolve) => setTimeout(resolve, milliseconds));
    return { action: "wait", milliseconds };
  }
  await inject(tab.id);
  const response = await chrome.tabs.sendMessage(tab.id, {
    type: "MULTIVERSA_CONTROL_ACTION",
    action: command.action,
    payload,
    sessionId: controlSession.sessionId
  });
  if (!response?.ok) throw new Error(response?.error || "La pestaña rechazó la acción.");
  return response.result;
}

async function receiveCommand(message) {
  const command = message.command;
  let result;
  try {
    result = { ok: true, result: await executeCommand(command) };
  } catch (error) {
    result = { ok: false, error: { code: "CONTROL_ACTION_FAILED", message: error.message } };
  }
  try {
    await postPersistent({
      version: 1,
      requestId: `result-${command.commandId}`,
      action: "control.result",
      target: controlSession?.origin || command.targetOrigin,
      payload: { commandId: command.commandId, ...result }
    });
  } catch {
    // The CLI will time out and the host keeps the inflight command for diagnosis.
  }
}

function connectControlPort() {
  if (controlPort) return;
  controlPort = chrome.runtime.connectNative(NATIVE_HOST_NAME);
  controlPort.onMessage.addListener((message) => {
    if (message?.event === "control.command" && message.command) {
      receiveCommand(message);
      return;
    }
    const pending = pendingNative.get(message?.requestId);
    if (!pending) return;
    clearTimeout(pending.timeout);
    pendingNative.delete(message.requestId);
    if (message.ok === true) pending.resolve(message);
    else pending.reject(new Error(message?.error?.message || "El host rechazó la solicitud."));
  });
  controlPort.onDisconnect.addListener(() => {
    const detail = chrome.runtime.lastError?.message || "El host nativo se desconectó.";
    controlPort = null;
    rejectPending(detail);
    if (controlSession && Date.now() < controlSession.expiresAt) {
      setTimeout(() => restoreControl().catch(() => {}), 1000);
    }
  });
}

async function registerControlSession(session) {
  controlSession = session;
  connectControlPort();
  await postPersistent({
    version: 1,
    requestId: `register-${crypto.randomUUID()}`,
    action: "control.register",
    target: session.origin,
    payload: session
  });
}

async function clearControl() {
  controlSession = null;
  if (controlPort) {
    controlPort.disconnect();
    controlPort = null;
  }
  await chrome.storage.session.remove(CONTROL_STORAGE_KEY);
  await chrome.alarms.clear(CONTROL_ALARM);
}

async function restoreControl() {
  if (controlPort && controlSession && Date.now() < controlSession.expiresAt) return controlSession;
  const stored = await chrome.storage.session.get(CONTROL_STORAGE_KEY);
  const session = stored?.[CONTROL_STORAGE_KEY];
  if (!session || Date.now() >= session.expiresAt) {
    await clearControl();
    return null;
  }
  const tab = await chrome.tabs.get(session.tabId);
  if (!tab?.url || new URL(tab.url).origin !== session.origin || !isAllowedURL(tab.url)) {
    await clearControl();
    return null;
  }
  await registerControlSession(session);
  return session;
}

async function enableControl() {
  const tab = await activeTab();
  await inject(tab.id);
  const origin = new URL(tab.url).origin;
  const session = {
    sessionId: crypto.randomUUID(),
    tabId: tab.id,
    origin,
    expiresAt: Date.now() + CONTROL_TTL_MS
  };
  await chrome.storage.session.set({ [CONTROL_STORAGE_KEY]: session });
  await chrome.alarms.create(CONTROL_ALARM, {
    delayInMinutes: 0.25,
    periodInMinutes: 0.5
  });
  await registerControlSession(session);
  return session;
}

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === CONTROL_ALARM) restoreControl().catch(() => {});
});

chrome.runtime.onStartup.addListener(() => {
  restoreControl().catch(() => {});
});

chrome.runtime.onMessage.addListener((message, _sender, respond) => {
  if (message?.type === "MULTIVERSA_CONTROL_KEEPALIVE") {
    const active = Boolean(
      controlSession &&
      controlPort &&
      Date.now() < controlSession.expiresAt &&
      _sender.tab?.id === controlSession.tabId
    );
    respond({ ok: true, active });
    return;
  }
  if (message?.type === "MULTIVERSA_INTERNAL_DOWNLOAD") {
    if (
      !controlSession ||
      Date.now() >= controlSession.expiresAt ||
      message.sessionId !== controlSession.sessionId ||
      !isAllowedDownloadURL(message.href)
    ) {
      respond({ ok: false, error: "La descarga no pertenece a la pestaña autorizada." });
      return;
    }
    chrome.downloads.download({
      url: message.href,
      filename: `Multiversa/Kit-SOS/${safeDownloadName(message.filename)}`,
      conflictAction: "uniquify",
      saveAs: false
    }).then(
      (downloadId) => respond({
        ok: true,
        downloadId,
        filename: safeDownloadName(message.filename)
      }),
      (error) => respond({ ok: false, error: error.message })
    );
    return true;
  }
  if (message?.type === "MULTIVERSA_CAPTURE") {
    captureSnapshot().then(
      (snapshot) => respond({ ok: true, snapshot }),
      (error) => respond({ ok: false, error: error.message })
    );
    return true;
  }
  if (message?.type === "MULTIVERSA_SEND_TO_CLI") {
    captureSnapshot().then(oneShotNative).then(
      (result) => respond({ ok: true, result }),
      (error) => respond({ ok: false, error: error.message })
    );
    return true;
  }
  if (message?.type === "MULTIVERSA_CONTROL_ENABLE") {
    enableControl().then(
      (session) => respond({ ok: true, expiresAt: session.expiresAt, origin: session.origin }),
      (error) => respond({ ok: false, error: error.message })
    );
    return true;
  }
  if (message?.type === "MULTIVERSA_CONTROL_STATUS") {
    restoreControl().then(
      (session) => respond({
        ok: true,
        active: Boolean(session && controlPort),
        expiresAt: session?.expiresAt || null
      }),
      () => respond({ ok: true, active: false, expiresAt: null })
    );
    return true;
  }
});

restoreControl().catch(() => {});
