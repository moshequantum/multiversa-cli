## Qué cambia · What changes

<!-- Una oración. Si requiere más, el PR es demasiado grande. -->
<!-- One sentence. If it needs more, the PR is too large. -->

## Tipo · Type

- [ ] Bug fix
- [ ] Feature
- [ ] Refactor
- [ ] Docs / chore

## Checklist

- [ ] `go build ./...` limpio
- [ ] `go test ./...` limpio — sin tests nuevos que fallen
- [ ] `./scripts/e2e-release.sh` valida el binario y el handshake MCP
- [ ] `govulncheck` y CodeQL sin hallazgos explotables
- [ ] Tests añadidos o actualizados para el cambio
- [ ] Sin dependencias AGPL/GPL introducidas al core MIT
- [ ] Sin secretos, `.env`, ni credenciales en el diff
- [ ] `CREDITS.md` actualizado si se añadió o cambió un motor upstream
- [ ] Sin referencias a términos legacy (CortexOS, Cockpit, Triada, Hydra Pool)
- [ ] Revisé `docs/DECISION-GATES.md`; límites/proveedores/storage tienen ADR
- [ ] El cambio tiene rollback y señales observables sin PII/secretos

## Issue relacionado · Related issue

Closes #

---

> La IA propone. Tú decides. El PR también.
