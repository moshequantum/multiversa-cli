# Multiversa CLI

> **ES** — Pre-configurador agnóstico del ecosistema agentic curado. Un comando, todos los motores, cualquier agente.
> **EN** — Agnostic pre-configurator for the curated agentic stack. One command, every engine, any agent.

**La IA propone, tú decides.** — *AI proposes, you decide.*

[![License: MIT](https://img.shields.io/badge/License-MIT-BDEB34.svg)](LICENSE)
[![Stack](https://img.shields.io/badge/stack-orchestrated-FAFCE8.svg)](CREDITS.md)
[![Go](https://img.shields.io/badge/go-1.22+-0A0A0F.svg)](https://go.dev)

---

## ¿Qué es esto? · What is this?

**ES.** Multiversa CLI **no inventa motores** — los **orquesta**. Es un wizard interactivo en terminal (TUI) + instalador multiplataforma que descarga y cablea un stack curado de proyectos open-source agenticos para que funcionen juntos, en cualquier agente que uses (Claude Code, Cursor, Codex, Gemini CLI, OpenCode, Aider…), con o sin backend (Supabase / Firebase / InsForge).

**EN.** Multiversa CLI **does not invent engines** — it **orchestrates** them. An interactive terminal wizard (TUI) + multi-platform installer that downloads and wires a curated stack of open-source agentic projects so they work together, with whichever agent you use (Claude Code, Cursor, Codex, Gemini CLI, OpenCode, Aider…), with or without a backend.

## Filosofía · Philosophy

- **Agnóstico** de agente, modelo, y suscripción. No requerimos cuentas. Tu setup, tu llamada.
- **Atribución built-in.** Cada `install` imprime créditos a los autores upstream. Ver [CREDITS.md](CREDITS.md).
- **Local-first.** Backend remoto es opcional. SQLite por default.
- **MIT.** Sin embeber código AGPL. Los motores AGPL (p.ej. MiroFish) se invocan como servicios externos.

## Stack orquestado · Stack we orchestrate

Multiversa Lab es **arquitectura + curaduría**. Los motores que activamos son obra de otros creadores. Crédito y enlace directo a cada uno:

| Motor · Engine | Autor · Author | Licencia · License | Repo |
|---|---|---|---|
| **Engram** — persistent agent memory | Gentleman-Programming | MIT | [github.com/Gentleman-Programming/engram](https://github.com/Gentleman-Programming/engram) |
| **Graphify** — content → knowledge graph | Safi Shamsi | MIT | [github.com/safishamsi/graphify](https://github.com/safishamsi/graphify) |
| **gentle-ai** — agentic ecosystem framework | Gentleman-Programming | MIT | [github.com/Gentleman-Programming/gentle-ai](https://github.com/Gentleman-Programming/gentle-ai) |
| **gentle-pi** — SDD harness for Pi agent | Gentleman-Programming | MIT | [github.com/Gentleman-Programming/gentle-pi](https://github.com/Gentleman-Programming/gentle-pi) |
| **codegraph** — semantic code knowledge graph | Colby McHenry | MIT | [github.com/colbymchenry/codegraph](https://github.com/colbymchenry/codegraph) |
| **MiroFish** ⚠️ | 666ghj | **AGPL-3.0** (opt-in, external only) | [github.com/666ghj/MiroFish](https://github.com/666ghj/MiroFish) |

> Si Multiversa te resulta útil, por favor **dale star a los repos upstream primero**. Sin ellos, Multiversa no existe.
> If you find Multiversa useful, please **star the upstream repos first**. Without them, Multiversa does not exist.

## Instalación · Install

```bash
# macOS (Homebrew)
brew tap moshequantum/multiversa
brew install multiversa

# Linux / macOS / WSL
curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/installers/shell-curl/install.sh | sh

# Windows (Scoop)
scoop bucket add multiversa https://github.com/moshequantum/scoop-multiversa
scoop install multiversa

# Go users (anywhere)
go install github.com/moshequantum/multiversa-cli/cmd/multiversa@latest
```

## Uso · Usage

```bash
multiversa init               # wizard interactivo · interactive wizard
multiversa add engram         # añadir motor · add an engine
multiversa connect cursor     # cablear agente · wire an agent
multiversa backend insforge   # configurar backend opcional · configure backend
multiversa doctor             # verificar salud · health check
multiversa credits            # ver atribución · view attribution
multiversa --version
```

## Agentes soportados · Supported agents

`claude-code` · `cursor` · `codex` · `gemini-cli` · `opencode` · `aider` · `cline` · `continue` · `roo-code` · `generic-mcp` (cualquier agente compatible con MCP)

## Backends opcionales · Optional backends

`local` (default, SQLite) · `supabase` · `firebase` · `insforge`

## Manifest declarativo · Declarative manifest

Ver [`multiversa.toml.example`](multiversa.toml.example). Reproducible, versionable, idempotente.

## Estructura · Layout

```
cmd/multiversa/          # Cobra commands
internal/
  wizard/                # Bubble Tea TUI + steps
  stack/                 # per-engine managers
  adapters/              # per-agent connectors
  backends/              # per-backend sync layer
  manifest/              # multiversa.toml parser
  credits/               # canonical attribution (single source of truth)
  theme/                 # Lipgloss tokens (Carbon · Chartreuse · Ivory)
installers/              # GoReleaser configs for every distribution channel
```

## Diseño · Design

Carbon `#0A0A0F` · Chartreuse `#BDEB34` (single accent) · Ivory `#FAFCE8`
Playfair Display Italic (display) · Sora 300 (body) · JetBrains Mono UPPERCASE (labels)

Ver [`docs/design-system.md`](docs/design-system.md).

## Contribuir · Contributing

Issues + PRs bienvenidos. **Antes**, lee [CREDITS.md](CREDITS.md) — el respeto a los autores upstream es la primera regla.

## Licencia · License

[MIT](LICENSE). Curaduría, arquitectura, sistema de diseño y ética: Moshe — Multiversa Lab / Group.

---

*Humano + IA, escalando el infinito.*
