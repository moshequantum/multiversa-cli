# Gates de decisión · Multiversa CLI

Estas reglas convierten “no romper nada” en evidencia verificable. Un cambio se
detiene si no puede responder **sí** a cada gate aplicable.

## 1. Dominio y arquitectura

- La lógica de dominio permanece en el anillo core y depende sólo de puertos.
- Adaptadores de disco, red, procesos y proveedores dependen hacia adentro.
- Todo paquete nuevo se clasifica en `internal/arch/arch_test.go`; no se amplía
  `knownViolations` para hacer pasar CI.
- Una decisión que cambie límites, almacenamiento, proveedor o contratos lleva ADR.

## 2. Privacidad y tenants

- Group y Lab son zonas de confianza diferentes: dos data planes, dos juegos de
  secretos y cero acceso cruzado implícito.
- Ningún secreto entra por argumentos, JSON, logs, Git o métricas.
- Los comandos con efectos requieren tenant explícito, mínimo privilegio y
  confirmación humana; el MCP público continúa read-only.

## 3. Pruebas y QA

- Unitarias para dominio, contract tests para adaptadores y E2E del binario real.
- `gofmt`, build, vet, race tests, arquitectura y E2E son obligatorios en PR.
- La prueba de release valida envelopes JSON y el handshake MCP, no mocks.

## 4. Ciberseguridad

- `govulncheck` y CodeQL bloquean publicación; secretos se inyectan sólo desde
  vault/secret stores y deben poder rotarse.
- Inputs externos tienen límites, allowlists, timeouts y fallan cerrados.
- Dependencias nuevas requieren autor, licencia, versión fijada y atribución.

## 5. Observabilidad y operación

- Todo fallo operativo debe producir estado, severidad, evidencia y próxima acción
  sin exponer valores sensibles.
- Despliegues usan staging/canary, health check, evidencia y rollback documentado.
- Alertas P0/P1 abiertas bloquean release; deriva P2 requiere aceptación explícita.

## 6. Release y reversibilidad

- Un solo binario canónico: artefacto de tag oficial, checksum verificado y metadata
  de commit/fecha. Builds locales dicen `-dev` y jamás se hacen pasar por release.
- Archivar antes de sustituir; no `reset --hard`, no force-push, no borrado de
  producción. La IA propone y valida; el humano conserva la última decisión.

