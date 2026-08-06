# Multiversa Lab adaptativo

## Estado

Decisión de arquitectura para la evolución posterior a `v0.8.0`.

Primera vertical implementada localmente el 5 de agosto de 2026:

- lifecycle común con evidencia para herramientas, motores y agentes;
- Hermes detectado como runtime opt-in separado de motores y skills;
- `doctor` separado de `detect`;
- `status` diario y ledger local `~/.multiversa/alerts.json`;
- tools MCP read-only para `doctor`, `status` y `alerts`;
- adapter explícito `multiversa connect hermes` hacia `multiversa mcp serve`;
- detección honesta de gentle-pi bloqueado cuando existe el paquete pero falta Pi.

No se cambiaron todavía el binario canónico ni el crontab: ambas acciones
requieren aprobación humana. Catálogo, `advise`, configuradores y loops siguen
en roadmap.

## Tesis

Multiversa CLI no compite con los motores que integra ni se atribuye su trabajo.
Es la capa local que convierte una necesidad humana en un sistema técnico
explicable, reproducible y gobernado:

```text
necesidad → inventario → diagnóstico → alternativas → propuesta → aprobación
          → configuración → verificación → operación → aprendizaje
```

Las herramientas pueden instalarse por separado. El valor de Multiversa es
determinar cuáles hacen falta, cuáles ya existen, cómo se conectan, cuánto
cuestan y qué riesgos o dependencias introducen.

## Principios

1. **Agnóstico por contrato.** Ningún agente, modelo, proveedor, framework,
   backend, cloud o runtime es obligatorio.
2. **Stack recomendado, no stack impuesto.** Lab publica una ruta curada y un
   catálogo mayor de integraciones opcionales.
3. **Atribución verificable.** Cada integración declara autor, repositorio,
   licencia, rol y condiciones de uso.
4. **Local primero.** El inventario de la máquina, los tenants y sus alertas se
   calculan localmente. Ningún Worker es necesario para auditar el host.
5. **La IA propone, la persona decide.** Una recomendación nunca equivale a una
   instalación o mutación.
6. **Degradación honesta.** `instalado` no significa `configurado`, y
   `configurado` no significa `saludable`.
7. **Fases pequeñas.** Un Project OS crece por capacidades y loops, no mediante
   una instalación total de una sola vez.

## Estado canónico de una capacidad

Cada runtime, engine, agente, skill, MCP, canal, backend o servicio debe
reportar el mismo ciclo:

```text
absent
detected
installed
configured
connected
indexed
healthy
drifted
update_available
blocked
```

Los estados no son mutuamente equivalentes. Por ejemplo, un paquete de
subagentes puede estar `installed` pero `blocked` si falta el runtime que lo
ejecuta. Un grafo está `configured` cuando el manifiesto declara Graphify, pero
solo está `indexed` cuando existen artefactos válidos y frescos.

Cada estado incluye evidencia, no inferencias silenciosas:

```json
{
  "id": "engram",
  "state": "configured",
  "evidence": ["binary:/path/to/engram", "mcp:codex"],
  "missing": ["shell_path"],
  "next_actions": ["add_go_bin_to_path"],
  "requires_approval": true
}
```

Los paths sensibles se redactan en superficies compartibles.

## Superficies del CLI

### `multiversa detect`

Inventario factual y de solo lectura:

- sistema operativo y arquitectura;
- runtimes y package managers;
- agentes y CLIs;
- engines y sus rutas reales;
- contenedores y targets de deploy;
- presencia de credenciales o sesiones, solo como booleanos y sin leer valores;
- perfiles, proyectos y repositorios conocidos.

### `multiversa doctor`

Deja de ser alias de `detect`. Evalúa coherencia:

- binarios duplicados o versiones divergentes;
- paquete instalado sin runtime;
- manifiesto que declara un motor sin índice materializado;
- PATH incompleto;
- cron configurado contra otro binario;
- integración presente pero no conectada;
- permisos inseguros;
- release o configuración con deriva.

### `multiversa status`

Es la pantalla diaria del operador. Resume:

- tenant activo;
- salud del host;
- stack recomendado y opcional;
- conexiones e índices;
- alertas abiertas;
- próxima acción aprobable;
- loops activos y última ejecución.

### `multiversa advise`

Produce un plan, nunca lo aplica. La primera versión usa reglas deterministas;
un modelo puede explicar o ampliar el plan, pero no reemplazar la evidencia.

Entradas mínimas:

- objetivo;
- stack existente;
- presupuesto inicial y mensual;
- privacidad y criticidad;
- volumen;
- capacidad técnica y de mantenimiento;
- preferencias y restricciones.

Salida:

- capacidades necesarias;
- alternativas comparadas;
- recomendación principal;
- razones para descartar otras opciones;
- costo y complejidad estimados;
- riesgos, bloqueantes y aprobaciones;
- plan por fases.

### `multiversa alerts`

Mantiene un ledger local de hallazgos con severidad:

- `P0`: riesgo inmediato de datos, dinero o aislamiento;
- `P1`: operación rota o bloqueada;
- `P2`: deriva, actualización o conexión incompleta;
- `P3`: mejora opcional.

El CLI siempre puede mostrar alertas. Cron, systemd, Telegram, email o un Worker
son destinos opcionales, no requisitos del motor de alertas.

### `multiversa catalog`

Separa dos vistas:

1. **Stack recomendado por Lab:** integración probada y mantenida.
2. **Catálogo de laboratorio:** opciones que una persona puede evaluar e
   integrar bajo su responsabilidad.

Cada entrada vive como integration pack declarativo y versionado. Debe incluir:

- procedencia y licencia;
- capacidades y requisitos;
- estrategias de instalación por plataforma;
- detectores de instalación/configuración/salud;
- configuración guiada;
- desinstalación o rollback;
- costos conocidos y lock-in;
- alternativas;
- skills y MCP relacionados;
- pruebas de aceptación.

## Skills, agentes y departamentos

Una skill es un procedimiento reusable. Un agente es un runtime con permisos y
herramientas. Un departamento es una composición de responsabilidades, skills,
agentes, límites y métricas. No deben tratarse como sinónimos.

El configurador debe comenzar por la capacidad que falta y luego sugerir la
composición mínima. Ejemplo:

```text
capacidad: seguimiento de prospectos
skills: calificar, resumir, redactar seguimiento
agente: cualquiera compatible con MCP y el schema requerido
integraciones: Gmail, CRM o ManyChat existentes
gate humano: aprobar el envío
```

Los directorios de skills externos se descubren y auditan. Multiversa no copia
automáticamente todo un catálogo ni ejecuta instrucciones de terceros sin
mostrar procedencia y permisos.

## Loops de un Project OS

Lab recomienda siete loops, activables por fases:

1. **Dirección y contexto** — objetivos, prioridades, decisiones, memoria y
   revisión semanal.
2. **Prospección y ventas** — señal, calificación, contacto, seguimiento,
   propuesta y decisión humana.
3. **Acuerdos y cobro** — alcance, SOAPS/contrato, aceptación, anticipo,
   factura y estado de cobro.
4. **Entrega** — intake, plan, ejecución, QA, aceptación, documentación y cierre.
5. **Comunicación** — inbox y canales, triage, respuesta, recordatorios y
   escalamiento humano.
6. **Finanzas y sostenibilidad** — caja, costos, suscripciones, compromisos,
   runway y alertas.
7. **Aprendizaje y gobierno** — memoria, skills, retrospectivas, seguridad,
   licencias, backups y actualización del catálogo.

### Orden para un solofounder

```text
Fase 0 · sobrevivir: dirección + ventas + cobro + entrega
Fase 1 · estabilizar: comunicación + finanzas
Fase 2 · escalar: aprendizaje, departamentos, subagentes y automatización
```

No se automatiza un loop que todavía no tiene un responsable, una entrada, una
salida, un criterio de éxito y un punto de aprobación definidos.

## Worker y servicios remotos

La auditoría del host permanece local. Un Worker puede añadirse después para:

- servir el catálogo firmado y sus releases;
- recibir señales mínimas autorizadas;
- enviar notificaciones;
- coordinar dispositivos;
- exponer un endpoint de estado compartible.

Por defecto no recibe inventario, paths, nombres de tenants ni credenciales. El
usuario ve exactamente qué se enviará y puede operar sin Worker.

## Frontera Lab / Group

- **Lab público:** CLI, schemas, integration packs, detectores, atribución,
  manifests, MCP y loops genéricos.
- **Group privado:** contratos, facturación, engagements, datos de clientes,
  delivery gestionado, Studio y políticas comerciales.
- **Project OS privado:** identidad, memoria, grafo, vault, skills, loops,
  métricas y decisiones del propietario.

## Secuencia de implementación

1. **En curso:** verdad local; detección y estados listos, unificación física de
   binario/PATH pendiente de aprobación.
2. **Implementado:** separar `doctor` de `detect` y añadir diagnósticos con evidencia.
3. **Implementado:** `status` y ledger local de alertas.
4. Formalizar integration packs y el catálogo recomendado/opcional.
5. Implementar `advise` determinista y su salida JSON versionada.
6. Añadir configuradores por pilar y por tenant.
7. Materializar loops y entregas programadas.
8. Conectar destinos remotos opcionales.

## Criterio de éxito de la primera fase

Al ejecutar `multiversa status`, una persona debe poder responder sin revisar
carpetas ni recordar comandos:

- qué está instalado;
- qué está realmente configurado;
- qué está conectado e indexado;
- qué está roto o duplicado;
- qué falta;
- qué propone Multiversa;
- qué acción requiere su aprobación.
