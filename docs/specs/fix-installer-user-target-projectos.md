# Spec: instalador pregunta usuario/target + nombre del ProjectOS

**Rama:** `fix/installer-user-target-projectos`
**Ejecutor:** Codex · **Spec + verificación:** Claude · **Decisión:** Moshe
**Regla:** `main` no se toca. Todo aquí, PR al final.

---

## Contexto / causa raíz (con evidencia)

El `curl … | sh` genera un **binario duplicado**:
- `install.sh:12` → `INSTALL_DIR="${MULTIVERSA_INSTALL_DIR:-/usr/local/bin}"` — default system.
- `install.sh:101-107` → si no es escribible, hace `sudo install` → binario **root:root** en `/usr/local/bin`.

Resultado real en la máquina de Moshe: dos binarios `multiversa` v0.9.3 idénticos
(`~/.local/bin` dueño moshe + `/usr/local/bin` dueño root), y el cron apuntando al de root.
El auditor lo marca P2 ("Binarios duplicados" + "El cron ejecuta otro binario").

**El instalador nunca pregunta DÓNDE ni con qué usuario instala. Ese es el bug.**

---

## Objetivo

1. **El instalador pregunta el target de instalación** (usuario vs sistema) antes de descargar.
2. **El wizard `init` pregunta el nombre del ProjectOS** y lo persiste antes de configurar pilares.
3. Detectar instalaciones previas y evitar crear un segundo binario.

---

## Cambio 1 — Target de instalación en `install.sh` (BLOQUEA LA DEMO, va primero)

**Archivo:** `installers/shell-curl/install.sh`

**Comportamiento nuevo (antes de la sección de descarga, ~línea 68):**

- Añadir un prompt de selección de target leído desde `/dev/tty` (reusar el patrón de `ask()`):
  - **Opción A (recomendada, default): usuario** → `INSTALL_DIR="$HOME/.local/bin"`, **sin sudo**.
  - **Opción B: sistema** → `INSTALL_DIR="/usr/local/bin"`, con sudo.
- Si `$HOME/.local/bin` no existe en la opción A: crearlo (`mkdir -p`) y, si no está en `PATH`,
  imprimir el `export PATH` a añadir al shell rc (no editar el rc sin confirmación).
- **Respetar overrides:** si `MULTIVERSA_INSTALL_DIR` está seteado, se usa tal cual y se OMITE el prompt.
- **Sin TTY / `MULTIVERSA_YES=1`:** default a **usuario** (`$HOME/.local/bin`), nunca sudo silencioso.

**Detección de duplicado (antes de instalar):**
- Si ya existe un `multiversa` en PATH en una ubicación distinta a la elegida, **avisar**
  ("ya hay un multiversa en <ruta>; instalar aquí creará un duplicado") y preguntar si continuar.
  No borrar nada automáticamente — la IA propone, Moshe decide.

**No romper:** checksums (85-97), `MULTIVERSA_VERSION`, la rama sudo para opción B,
y la ejecución bajo `curl | sh` (lectura desde `/dev/tty`).

**Aceptación C1:**
- `curl … | sh` en TTY ofrece elegir usuario/sistema; usuario NO pide sudo y deja el binario en `~/.local/bin`.
- Con `MULTIVERSA_INSTALL_DIR=/x` no pregunta y usa `/x`.
- Sin TTY instala en `~/.local/bin` sin sudo.
- Segunda corrida detecta el binario existente y avisa antes de duplicar.

---

## Cambio 2 — Nombre del ProjectOS en el wizard `init`

**Archivos (de graphify):** `internal/wizard/model.go`, `internal/wizard/steps/*.go`
(nuevo step de naming, antes del step `install.go`), persistencia en el perfil
(`~/.multiversa/profile.toml` / contrato ProjectOS — ver `capabilities.go` / `TestCapabilitiesDeclareProjectOSWriteContract`).

**Comportamiento:**
- Nuevo step temprano en el wizard: **"¿Cómo se llamará tu ProjectOS?"** (input de texto, ej. `MiniUniversoOS`).
  - Validar: no vacío, slug-safe; default sugerido derivado del hostname o del usuario.
- Persistir el nombre en el perfil ANTES de instalar/configurar pilares, para que el resto del flujo lo use.

**Aceptación C2:**
- `multiversa init` pide el nombre del ProjectOS y lo guarda en el perfil.
- `multiversa init --dry-run` muestra el nombre elegido sin escribir.
- Tras `init`, `multiversa status`/`manifest` reflejan el nombre del ProjectOS.

---

## Fuera de alcance
- Borrar el binario duplicado actual (lo hace Moshe con `rm`).
- Homebrew tap / Scoop (marcados "planned").
- Reescritura del sistema de tenants.

## Verificación (la corre Claude antes del merge)
- Los criterios de aceptación C1 y C2.
- `go build ./...` y `go test ./...` verdes.
- `shellcheck installers/shell-curl/install.sh` sin errores nuevos.
- Ninguna corrida crea un binario root a menos que se elija sistema explícitamente.
