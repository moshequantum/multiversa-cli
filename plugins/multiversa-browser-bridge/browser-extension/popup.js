const domain = document.querySelector("#domain");
const control = document.querySelector("#control");
const capture = document.querySelector("#capture");
const send = document.querySelector("#send");
const status = document.querySelector("#status");
const previewWrap = document.querySelector("#preview-wrap");
const preview = document.querySelector("#preview");

function setStatus(message, error = false) {
  status.textContent = message;
  status.classList.toggle("error", error);
}

async function currentTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  domain.textContent = tab?.url ? new URL(tab.url).hostname : "Sin pestaña activa";
}

async function request(type, button) {
  button.disabled = true;
  setStatus(type === "MULTIVERSA_CAPTURE" ? "Creando vista segura…" : "Enviando al puente local…");
  try {
    const response = await chrome.runtime.sendMessage({ type });
    if (!response?.ok) throw new Error(response?.error || "Acción no disponible.");
    if (response.snapshot) {
      preview.textContent = JSON.stringify(response.snapshot, null, 2);
      previewWrap.hidden = false;
      setStatus(`Vista lista: ${response.snapshot.interactive.length} elementos navegables.`);
    } else {
      setStatus(response.result?.message || "La vista segura se envió al puente local.");
    }
  } catch (error) {
    setStatus(error.message, true);
  } finally {
    button.disabled = false;
  }
}

async function refreshControlStatus() {
  try {
    const response = await chrome.runtime.sendMessage({ type: "MULTIVERSA_CONTROL_STATUS" });
    if (response?.ok && response.active) {
      const minutes = Math.max(1, Math.ceil((response.expiresAt - Date.now()) / 60000));
      control.textContent = `Control activo · ${minutes} min`;
      setStatus("La CLI ya puede operar esta pestaña sin pasos manuales.");
    } else {
      control.textContent = "Activar control por 30 min";
    }
  } catch {
    control.textContent = "Activar control por 30 min";
  }
}

control.addEventListener("click", async () => {
  control.disabled = true;
  setStatus("Abriendo canal de control local…");
  try {
    const response = await chrome.runtime.sendMessage({ type: "MULTIVERSA_CONTROL_ENABLE" });
    if (!response?.ok) throw new Error(response?.error || "No se pudo activar el control.");
    const minutes = Math.max(1, Math.ceil((response.expiresAt - Date.now()) / 60000));
    control.textContent = `Control activo · ${minutes} min`;
    setStatus("Canal activo. Multiversa puede continuar sin que seas intermediario.");
  } catch (error) {
    setStatus(error.message, true);
  } finally {
    control.disabled = false;
  }
});

capture.addEventListener("click", () => request("MULTIVERSA_CAPTURE", capture));
send.addEventListener("click", () => request("MULTIVERSA_SEND_TO_CLI", send));
currentTab().catch(() => { domain.textContent = "No disponible"; });
refreshControlStatus();
