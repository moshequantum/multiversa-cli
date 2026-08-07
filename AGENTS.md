# Reglas de revisión · Multiversa CLI

Contrato de revisión para agentes de IA (gga, Claude Code, Codex, opencode).
Deriva de [CONTRIBUTING.md](CONTRIBUTING.md) y [docs/DECISION-GATES.md](docs/DECISION-GATES.md);
si hay discrepancia, esos dos mandan.

Multiversa CLI orquesta motores ajenos. No los reimplementa, no los embebe y no
se atribuye su trabajo. Casi toda regla de aquí protege esa frontera.

---

## Bloqueantes · nunca aprobar un cambio que los viole

- **Sin dependencias AGPL/GPL en el core.** El proyecto distribuye bajo MIT y
  una licencia viral lo rompe. MiroFish se invoca como binario o servicio
  externo, jamás embebido en el árbol de fuentes.
- **Sin secretos, `.env`, credenciales ni semillas SQL.** La frontera de
  privacidad Group/Lab es constitucional, no una preferencia.
- **`npm` está vetado** para cualquier flujo JS/TS: `pnpm` únicamente.
- **Nada de exit codes sin verificar.** Un comando externo que devuelve 0 no
  prueba que hizo lo que dijo. Si una operación muta estado ajeno (config de
  otro agente, cron, vault, registro MCP), verificar el resultado contra la
  fuente de verdad antes de reportar éxito. Reportar una acción inexistente es
  peor que fallar: el operador se entera tarde.
- **Sin nuevas excepciones a los gates de decisión.**

## Correctitud

- Todo comportamiento nuevo lleva test. TDD estricto en los wizards TUI.
- Los tests de Bubble Tea ejercen `Init()` / `Update()` / `View()` directamente,
  sin `tea.Program` real. Seguir ese patrón.
- `go build ./...`, `go vet ./...`, `gofmt -l .` y `go test -count=1 ./...`
  limpios. El CI es la evidencia, no la descripción del PR.
- Un cambio que toque el flujo de release debe pasar `./scripts/e2e-release.sh`.
- Detección y diagnóstico son de **solo lectura**. `detect`, `doctor`, `status`
  y `alerts` no modifican nada de la máquina. Cualquier mutación exige que la
  persona ejecute un comando explícito.

## Duplicación

- Una lista escrita a mano que replique un registro existente es un defecto,
  no un estilo. Derivarla del registro. Este repo ya sufrió el caso:
  `agentIDs()` omitía `hermes` porque duplicaba `adapters.List()`.

## Comentarios

- Explican **por qué**, no qué. El qué ya está en el código.
- Un comentario que documenta una trampa real de un sistema externo vale;
  uno que parafrasea la línea siguiente, no.
- Sin comentarios de andamiaje (`// TODO: quitar esto`) en `main`.

## Idioma

- Comentarios de código: **inglés** (Go estándar).
- Strings visibles para la persona (TUI, salida de CLI, errores): **español
  latinoamericano neutro**, salvo `--locale=en`.
- Un mensaje de error debe decir qué hacer a continuación, no solo qué falló.

## Commits

`feat:` · `fix:` · `chore:` · `docs:` · `test:` · `refactor:`

Sin atribución de IA ni `Co-Authored-By`.

Un commit es una unidad revisable: el cambio, su test y su documentación
juntos. Si el diff pasa de ~400 líneas, partirlo en PRs encadenados.

---

*La IA propone. Tú decides.*
