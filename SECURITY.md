# Seguridad · Security Policy

## Versiones soportadas · Supported versions

| Versión | Soportada |
|---|---|
| v0.4.x (actual) | ✓ |
| v0.3.x | Solo vulnerabilidades críticas |
| < v0.3 | No soportada |

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

---

## Tiempo de respuesta · Response timeline

- Acuse de recibo: dentro de 72 horas.
- Evaluación inicial: dentro de 7 días.
- Parche o mitigación: depende de la severidad; críticas tienen prioridad máxima.

---

*Multiversa Lab · MIT · 2026*
