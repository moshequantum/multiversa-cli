# Empieza aquí · Multiversa en 5 minutos

> **La IA propone, tú decides.** Nada se instala, se cambia ni se envía sin tu confirmación.

Multiversa convierte tu computadora en un **Sistema Operativo de Negocios Inteligente**: tu memoria, tu marca, tus pilares y tus herramientas de IA trabajando juntas — y tú solo tomas decisiones.

---

## 1 · Instala (un solo comando)

Abre una terminal y pega:

```bash
curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/installers/shell-curl/install.sh | sh
```

El instalador se encarga de todo. Tú solo respondes **y** (sí) o **n** (no) a cada pregunta:

| Te preguntará | Qué significa | Si dudas |
|---|---|---|
| ¿Instalo brew / pipx / pnpm? | Herramientas base que los motores necesitan | Responde **y** |
| ¿Corro el asistente de configuración? | Abre el wizard visual paso a paso | Responde **y** |
| ¿Activo el chequeo diario? | Cada mañana revisa si hay actualizaciones y **te avisa** (jamás actualiza solo) | Responde **y** |

Verifica que quedó instalado:

```bash
multiversa version
```

## 2 · Crea tu OS (tu perfil)

Tu OS es un **perfil aislado**: tu identidad de marca, tus pilares, tu bóveda de secretos y tu memoria. Puedes tener varios (uno tuyo, uno por cliente) y cambiar entre ellos sin que se mezclen.

```bash
# Elige la plantilla que se parece a ti:
multiversa tenant new mi-os --kind personal-os        # productividad personal
multiversa tenant new mi-marca --kind personal-brand  # marca personal (contenido, ofertas)
multiversa tenant new mi-agencia --kind agency        # agencia (clientes, entrega, marca)

multiversa tenant use mi-os      # actívalo
multiversa tenant show mi-os     # mira tu ADN
```

Tu perfil vive en `~/.multiversa/tenants/mi-os/`. Dentro hay una carpeta **vault/**: ahí van tus claves y secretos. Multiversa **nunca** lee, copia ni sincroniza lo que pongas ahí.

Para personalizarlo, abre `~/.multiversa/tenants/mi-os/multiversa.toml` y completa la sección `[identity]` con tu marca, tu voz y tus límites (taboos). Todo lo que el sistema haga después heredará esa identidad.

## 3 · Conecta tu IA

Si usas Claude Code (u otro agente compatible con MCP):

```bash
claude mcp add multiversa -- multiversa mcp serve
```

Desde ese momento tu IA puede leer tu entorno, tus perfiles y el estado de tus herramientas — **pero no puede cambiar nada**: las acciones siempre pasan por ti.

---

## Comandos del día a día

```bash
multiversa detect      # ¿qué hay en mi máquina? (solo lectura)
multiversa updates     # ¿hay algo que actualizar? (solo avisa)
multiversa tenant list # mis perfiles y cuál está activo
multiversa credits     # los autores de los motores que orquestamos
```

## Preguntas frecuentes

**¿Esto me cambia algo sin avisar?** No. Es el principio fundacional: detectar es de solo lectura, instalar requiere tu confirmación, actualizar solo te avisa, y sincronizar siempre se propone primero.

**¿Dónde están mis datos?** En tu máquina (`~/.multiversa/`). El respaldo remoto (InsForge, Google Drive) es opcional, se configura por perfil y jamás incluye tu vault.

**¿Puedo tener el OS de un cliente y el mío en la misma máquina?** Sí — para eso existen los tenants. Cada uno está aislado: `multiversa tenant use <slug>` cambia el contexto completo.

**Algo falló.** Corre `multiversa doctor` y comparte la salida. Y si quieres reportarlo: [github.com/moshequantum/multiversa-cli/issues](https://github.com/moshequantum/multiversa-cli/issues).

---

*Humano + IA, navegando el infinito de a uno.*
