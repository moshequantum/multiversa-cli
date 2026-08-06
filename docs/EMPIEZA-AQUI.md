# Empieza aquí · Multiversa en 5 minutos

> **La IA propone, tú decides.** Nada se instala, se cambia ni se envía sin tu confirmación.

Multiversa convierte tu computadora en un **Sistema Operativo de Negocios Inteligente**: tu memoria, tu marca, tus pilares y tus herramientas de IA trabajando juntas — y tú solo tomas decisiones.

---

## 1 · Instala (Elige tu método)

### Opción A: Instalador Visual Asistido por Voz (Tauri + ElevenLabs)
Si prefieres una interfaz gráfica interactiva, acrílica y con asistencia de voz inteligente (que se sincroniza con tu configuración):

```bash
# Ingresa al directorio del instalador visual, instala dependencias y lánzalo
cd multiversa-installer
pnpm install
pnpm tauri dev
```

*El asistente de voz usará ElevenLabs si ingresas tu API Key, o el sintetizador nativo de tu sistema si lo dejas en blanco.*

### Opción B: Instalador Rápido por Terminal (CLI / TUI)
Si prefieres usar la terminal tradicional, abre tu consola y pega:

```bash
curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/installers/shell-curl/install.sh | sh
```

El instalador se encarga de todo. Tú solo respondes **y** (sí) o **n** (no) a cada pregunta:

| Te preguntará | Qué significa | Si dudas |
|---|---|---|
| ¿Instalo brew / pipx / pnpm? | Herramientas base que los motores necesitan | Responde **y** |
| ¿Corro el asistente de configuración? | Abre el wizard visual (TUI) paso a paso | Responde **y** |
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

Si ya tienes fuentes públicas aprobadas, puedes crear el OS y construir su
grafo en un solo flujo reanudable:

```bash
multiversa tenant bootstrap cliente-os \
  --name "Cliente" --owner "Cliente" \
  --source https://cliente.example/acerca-de \
  --author "Cliente" --contributor "Tu nombre" --activate
```

Cada `--source` queda registrada con procedencia antes de que Graphify extraiga
el grafo. Usa `--dry-run` para revisar el plan sin crear archivos ni acceder a
la red. Graphify ingiere las URLs que le entregas: no descubre perfiles ni
decide por sí mismo si dos identidades públicas pertenecen a la misma persona.

Puedes conectar varios proveedores semánticos por tenant. La clave se introduce
de forma oculta en una terminal y permanece en el vault local:

```bash
multiversa tenant connect cliente-os gemini  --model gemini-3.6-flash --priority 10
multiversa tenant connect cliente-os mistral --model mistral-large-latest --priority 20
multiversa tenant connect cliente-os groq    --model qwen/qwen3.6-27b --priority 30
```

El bootstrap intentará los proveedores configurados en orden de prioridad y
solo activará el tenant después de obtener y validar un grafo completo.

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
multiversa doctor      # ¿qué está incoherente, bloqueado o sin indexar?
multiversa status      # tenant, salud, alertas y próxima acción aprobable
multiversa alerts      # historial local de hallazgos abiertos/resueltos
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
