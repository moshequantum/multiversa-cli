# ADR-003 · Group y Lab usan data planes separados

- **Estado:** aceptado
- **Fecha:** 2026-08-05

## Decisión

Multiversa Group y Multiversa Lab usarán **dos proyectos InsForge** dentro de la
misma organización. Compartirán contratos versionados y librerías, no base de
datos, buckets, service roles ni secretos. Group es privado; Lab expone sólo el
conjunto mínimo de capacidades públicas.

Un Worker de Cloudflare puede ser el gateway común de modelos (Groq, Gemini y
Mistral), pero deberá autenticar al llamador, fijar tenant/ruta del lado servidor,
aplicar allowlists de modelos, cuotas, rate limiting, timeouts, redacción de logs
y circuit breakers. El cliente nunca elige libremente el data plane.

## Consecuencias

- Menor radio de explosión y RLS más demostrable.
- Dos migraciones y dos juegos de secretos, coordinados por contratos comunes.
- Ningún despliegue se habilita hasta tener credenciales Cloudflare válidas,
  proyectos InsForge enlazados y pruebas de aislamiento negativas.

