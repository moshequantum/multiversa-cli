# Seguridad · Security Policy

## Versiones soportadas · Supported versions

| Versión | Soportada |
|---|---|
| v0.9.x (desarrollo) | ✓ |
| v0.8.x (estable) | ✓ |
| v0.7.x | Solo vulnerabilidades críticas |
| < v0.7 | No soportada |

---

## Reportar una vulnerabilidad · Reporting a vulnerability

**No abras un issue público para vulnerabilidades de seguridad.**

Envía un reporte privado al maintainer:

1. **GitHub Security Advisory** — usa la pestaña "Security" > "Report a vulnerability" en este repositorio.
2. Si no tienes acceso a esa función, abre un issue con el título `[SECURITY] — descripción genérica` y el maintainer te contactará para continuar fuera de público.

Incluye en tu reporte:
- Descripción concisa del vector de ataque.
- Pasos mínimos para reproducirlo.
- Impacto esperado (confidencialidad, integridad, disponibilidad).
- Versión afectada.

---

## Alcance · Scope

Este proyecto es un wizard CLI local. El modelo de amenaza relevante:

- **En scope:** Inyección de comandos a través de inputs del usuario, escapado insuficiente en llamadas al sistema, permisos de archivo inseguros en scripts generados, credenciales expuestas en logs o en el perfil de usuario.
- **Fuera de scope:** Vulnerabilidades en los motores upstream (Engram, Graphify, etc.) — repórtalas directamente a sus autores. Ver [CREDITS.md](CREDITS.md) para los repos correspondientes.

## Fronteras obligatorias

- Secretos únicamente por STDIN/vault con permisos `0700/0600`; nunca en Git,
  argumentos, envelopes JSON, telemetría ni logs.
- Group y Lab usan data planes separados. No existe una credencial compartida
  con acceso de escritura a ambos.
- El servidor MCP expone observación read-only; efectos externos exigen comando
  local explícito y autorización humana.
- Un release requiere race tests, E2E del binario, `govulncheck`, CodeQL,
  checksums y metadata de commit/fecha.

---

## Tiempo de respuesta · Response timeline

- Acuse de recibo: dentro de 72 horas.
- Evaluación inicial: dentro de 7 días.
- Parche o mitigación: depende de la severidad; críticas tienen prioridad máxima.

---

*Multiversa Group LLC — Multiversa Lab · MIT · 2026*
