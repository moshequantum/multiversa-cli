# Multiversa CLI

> **La IA propone, tú decides.**  
> *AI proposes, you decide.*

Un conductor de campo agentic. No inventa los motores — los despierta, los conecta y los pone a orbitar juntos.  
An agentic field conductor. It does not invent the engines — it wakes them, connects them, and sets them in orbit together.

[![License: MIT](https://img.shields.io/badge/License-MIT-BDEB34.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.22+-0A0A0F.svg)](https://go.dev)
[![v0.4.0](https://img.shields.io/badge/version-v0.4.0-BDEB34.svg)](https://github.com/moshequantum/multiversa-cli/releases)
[![Stack](https://img.shields.io/badge/stack-orchestrated-5C5C64.svg)](CREDITS.md)

---

```
                         · · ·  M U L T I V E R S A  · · ·
                         quantum intelligence orchestrator

          ┌─ Técnica ─────────────────────────────────────────┐
          │  detect · stack · init                             │
          │  read your universe before touching anything       │
          └────────────────────────────────────────────────────┘
          ┌─ Identitaria ──────────────────────────────────────┐
          │  profile · workspace · brain                       │
          │  SSH · GPG · memory · knowledge graph              │
          └────────────────────────────────────────────────────┘
          ┌─ Operacional ──────────────────────────────────────┐
          │  usb · credits                                     │
          │  encrypted persistence · upstream attribution      │
          └────────────────────────────────────────────────────┘

          engines: engram · graphify · gentle-ai · codegraph
          agents:  claude-code · cursor · codex · gemini-cli
                   opencode · aider · cline · continue · roo-code
          backends: local · supabase · firebase · insforge
```

---

## Qué es esto · What is this

**ES.** Multiversa CLI es un wizard interactivo en terminal (TUI) + instalador multiplataforma que descarga, configura y conecta un stack curado de proyectos open-source agenticos para que funcionen juntos — con cualquier agente que uses, en cualquier backend que ya tengas.

No genera código. No vende acceso. No asume suscripciones.  
Detecta lo que hay, propone lo que falta, ejecuta solo con tu confirmación.

**EN.** Multiversa CLI is an interactive terminal wizard (TUI) + multi-platform installer that downloads, configures, and connects a curated open-source agentic stack so it works together — with whichever agent you use, on whichever backend you already have.

It does not generate code. It does not sell access. It does not assume subscriptions.  
It reads what is there, proposes what is missing, and executes only on your confirmation.

---

## Filosofía · Philosophy

Los motores de inteligencia agentic ya existen. Están dispersos, aislados, sin memoria compartida entre ellos.  
Multiversa es la **fuerza de unión** — el campo que los hace orbitar en el mismo sistema.

The agentic intelligence engines already exist. They are scattered, isolated, with no shared memory between them.  
Multiversa is the **binding force** — the field that makes them orbit the same system.

Cuatro principios que no cambian:

| Principio | Descripción |
|---|---|
| **Agnóstico** | Sin preferencia de agente, modelo, ni suscripción. Lo que ya tienes funciona. |
| **Atribución built-in** | Cada instalación imprime créditos a los autores upstream. Ver [CREDITS.md](CREDITS.md). |
| **Local-first** | El backend remoto es opcional. SQLite por defecto. Tus datos en tu máquina. |
| **No destructivo por diseño** | Detect es de solo lectura. Nada se ejecuta sin confirmación explícita. |

---

## Los nodos que orquestamos · Engines we orchestrate

Multiversa Lab es **arquitectura y curaduría** — no autoría. Cada motor es obra de su creador.  
Si te resulta útil, **dales star a sus repos primero**. Sin ellos, esto no existe.

| Motor · Engine | Qué hace · What it does | Autor · Author | Licencia |
|---|---|---|---|
| **Engram** | Memoria persistente entre sesiones de agente | Gentleman-Programming | MIT |
| **Graphify** | Transforma contenido en grafos de conocimiento | Safi Shamsi | MIT |
| **gentle-ai** | Configurador de ecosistema agentic (memoria + SDD + skills + MCP) | Gentleman-Programming | MIT |
| **gentle-pi** | Harness SDD para el agente Pi | Gentleman-Programming | MIT |
| **codegraph** | Grafo semántico del código fuente (tree-sitter) | Colby McHenry | MIT |
| **MiroFish** ⚠ | Motor de simulación multi-agente (opt-in, solo externo) | 666ghj | **AGPL-3.0** |

> MiroFish se invoca como binario/servicio externo únicamente. Nunca se embebe en el core MIT.

---

## Instalación · Install

```bash
# macOS (Homebrew)
brew tap moshequantum/multiversa
brew install multiversa

# Linux / macOS / WSL — un comando, sin dependencias previas
curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/installers/shell-curl/install.sh | sh

# Windows (Scoop)
scoop bucket add multiversa https://github.com/moshequantum/scoop-multiversa
scoop install multiversa

# Go (cualquier plataforma)
go install github.com/moshequantum/multiversa-cli/cmd/multiversa@latest
```

---

## Uso · Usage

```bash
# Entrada principal — elige tu camino
multiversa lab                # meta-wizard por capas (v0.4.0+)
multiversa init               # wizard de setup completo

# Módulos individuales
multiversa detect             # leer entorno, sin modificar nada
multiversa stack              # instalar toolchain base (Go · Rust · Node · pnpm · Docker)
multiversa add engram         # añadir un motor específico
multiversa connect cursor     # cablear un agente de IA
multiversa backend insforge   # configurar backend (opcional)

# Información
multiversa credits            # atribución completa upstream
multiversa doctor             # verificar salud del stack
multiversa version            # versión actual
```

### `multiversa lab` — el meta-wizard (v0.4.0)

```
↑/↓  navegar pasos dentro de la capa
tab  avanzar a la siguiente capa
⏎    lanzar el paso seleccionado
q    salir (nada se modifica al navegar)
```

Los pasos completados en sesiones anteriores se marcan ✓ y son saltables.  
`--reinstall` fuerza la re-ejecución de pasos ya marcados.

---

## Arquitectura · Layout

```
cmd/multiversa/          Cobra commands + Bubble Tea TUI models
internal/
  tui/                   Primitivas TUI compartidas (Step · Header · Selector · ProgressList · Sidebar)
  profile/               Perfil de usuario (Layer enum · InstalledEngines · Engram bridge)
  adapters/              Conectores por agente (claude-code · cursor · codex · gemini-cli · …)
  backends/              Capa de sync por backend (local · supabase · firebase · insforge)
  stack/                 Managers por motor (engram · graphify · gentle-ai · …)
  wizard/                Pasos del wizard init (Bubble Tea)
  theme/                 Tokens Lipgloss (Carbon · Chartreuse · Ivory)
  detect/                Escáner de entorno (OS · toolchain · motores presentes)
  credits/               Atribución upstream (fuente única de verdad)
installers/              GoReleaser — Brew · curl · Scoop · go install
```

---

## Diseño · Design system

```
Carbon   #0A0A0F  — fondo
Ivory    #FAFCE8  — texto principal
Chartreuse #BDEB34 — acento único, señal de vida

Playfair Display Italic  — titulares
Sora 300                 — cuerpo
JetBrains Mono UPPERCASE — labels y código
```

Los tokens del CLI viven en [`internal/theme`](internal/theme).

---

## Contribuir · Contributing

Lee [CONTRIBUTING.md](CONTRIBUTING.md) antes de abrir un PR.  
La primera regla: **respeto a los autores upstream**. Leer [CREDITS.md](CREDITS.md) es obligatorio.

---

## Seguridad · Security

Ver [SECURITY.md](SECURITY.md) para reportar vulnerabilidades de forma responsable.

---

## Licencia · License

[MIT](LICENSE). Curaduría, arquitectura, sistema de diseño y la ética *"La IA propone, tú decides"*: Moshe — Multiversa Lab / Multiversa Group.

---

*Humano + IA, navegando el infinito de a uno.*
