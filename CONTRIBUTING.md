# Contribuir · Contributing

Gracias por querer mejorar Multiversa CLI.  
Antes de escribir una sola línea, este documento.

---

## Antes de empezar · Before you start

1. **Lee [CREDITS.md](CREDITS.md).** El proyecto orquesta motores ajenos. Entender qué es de quién es la primera responsabilidad de cualquier contribuyente.
2. **Lee [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).** Breve. Vale la pena.
3. **Abre un issue antes de un PR grande.** Un párrafo que explique el problema y la solución propuesta ahorra semanas de trabajo en la dirección equivocada.

---

## Reglas técnicas no negociables · Non-negotiable technical rules

| Regla | Razón |
|---|---|
| `pnpm` únicamente para JS/TS — `npm` está vetado | Higiene de supply-chain en el ecosistema Lab |
| Sin dependencias AGPL/GPL en el core MIT | El proyecto distribuye bajo MIT; licencias virales lo rompen |
| MiroFish = binario/servicio externo, nunca embebido | AGPL-3.0 — nunca en el árbol de fuentes |
| Sin secretos, `.env`, credenciales ni semillas SQL en el repo | La frontera de privacidad Group/Lab es constitucional |
| `go vet ./...` y `go test ./...` limpios antes de cada PR | CI es la evidencia, no la descripción |

---

## Flujo de trabajo · Workflow

```bash
# 1. Fork + clon
git clone https://github.com/TU_USUARIO/multiversa-cli
cd multiversa-cli

# 2. Rama descriptiva
git checkout -b feat/nombre-del-cambio

# 3. Construir y testear
go build ./...
go test ./...

# 4. PR hacia main
```

---

## Tests · Testing

El proyecto sigue TDD estricto para todos los wizards TUI.  
Cada feature nueva lleva su test. Sin tests, sin merge.

```bash
go test ./...                    # suite completa
go test ./cmd/multiversa/... -v  # solo cmd, verbose
go test -run TestLab ./...       # filtro por nombre
```

Los tests de Bubble Tea ejercen `Init()` / `Update()` / `View()` directamente — sin `tea.Program` real. Seguir ese patrón en tests nuevos.

---

## Nombrar commits · Commit naming

```
feat: descripción breve del feature
fix: qué se rompía y por qué
chore: mantenimiento sin cambio funcional
docs: solo documentación
test: solo tests
refactor: sin cambio de comportamiento observable
```

---

## Idioma en el código · Language in code

- **Comentarios de código:** inglés (Go estándar).
- **Strings visibles para el usuario (TUI, CLI output):** español latinoamericano neutro, salvo que el usuario haya configurado `--locale=en`.
- **Issues y PRs:** español o inglés, tu elección.

---

## Lo que no necesita PR · What does not need a PR

- Correcciones de typos en docs — un PR pequeño y directo está bien.
- Actualizaciones de `CREDITS.md` si un repo upstream cambió de URL o licencia.
- Mejoras de mensajes TUI que no alteren la lógica.

---

## Lo que siempre necesita discusión previa · What always needs prior discussion

- Nuevos motores al stack curado (implica auditoría de licencia + atribución).
- Nuevos backends.
- Cambios al sistema de diseño (tokens, tipografía, paleta).
- Cambios al manifiesto `multiversa.toml`.

---

*La IA propone. Tú decides. El PR también.*
