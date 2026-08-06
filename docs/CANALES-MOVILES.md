# Canales móviles — Telegram vía el gateway de Hermes

El CLI de Multiversa corre en **Windows, macOS y Linux** (`amd64` y `arm64`).
Cada release publica los seis archivos más paquetes `.deb`, `.rpm` y
`.archlinux`. Ese es el camino en escritorio y servidor.

En **móvil no hay CLI**. La superficie recomendada es un canal de mensajería, y
por ahora documentamos uno solo: **Telegram**, a través del gateway de
[Hermes Agent](https://github.com/NousResearch/hermes-agent) (Nous Research, MIT).

---

## Por qué Telegram y no WhatsApp

Hermes soporta varias plataformas (`hermes gateway setup` las lista: Telegram,
Discord, WhatsApp, Weixin y más). Elegimos Telegram por dos razones operativas:

1. **WhatsApp es restrictivo con las plantillas.** Los mensajes iniciados por el
   negocio pasan por plantillas aprobadas. Un ledger de alertas que cambia de
   forma según el hallazgo no encaja en ese modelo sin fricción constante.
2. **La API de WhatsApp pasa a ser de pago el 1 de septiembre de 2026.** No
   queremos que la ruta recomendada de notificación tenga un costo por mensaje.

Telegram usa un bot token, no cobra por mensaje y no exige plantillas. Para
alertas operativas es la opción de menor desgaste.

> Esto es una recomendación de Multiversa, no un límite del motor. El ledger de
> `multiversa alerts` es local y agnóstico: cron, systemd, email o un Worker
> siguen siendo destinos válidos. Solo documentamos Telegram por ahora.

---

## Dónde encaja Hermes

Hermes **no es un motor embebido ni un requisito**. Es un runtime de agente
opcional que se conecta como *cliente* del MCP read-only de Multiversa.

```
  multiversa mcp serve   ──(stdio, read-only)──>   Hermes Agent
                                                        │
                                                        ▼
                                                  gateway → Telegram
```

Multiversa nunca toca la configuración de Hermes por su cuenta. `detect` solo lo
inventaria y `doctor` solo diagnostica la conexión. La configuración cambia
únicamente cuando la persona ejecuta `multiversa connect hermes`.

---

## Puesta en marcha

### 1. Registrar Multiversa como MCP en Hermes

```bash
multiversa connect hermes
```

Corre `hermes mcp add multiversa --command <ruta-al-binario> --args mcp serve`.
Para revertirlo: `multiversa disconnect hermes`.

### 2. Configurar el canal de Telegram

```bash
hermes gateway setup
```

Wizard interactivo. Pide el bot token de Telegram (lo emite
[@BotFather](https://t.me/BotFather)) y el chat de destino. Las credenciales
quedan en `~/.hermes/.env` y `~/.hermes/config.yaml`.

### 3. Levantar el gateway

En escritorio o servidor, como servicio de fondo:

```bash
hermes gateway install
```

```bash
hermes gateway start
```

En WSL, Docker o Termux, en primer plano:

```bash
hermes gateway run
```

Estado en cualquier momento:

```bash
hermes gateway status
```

### 4. Verificar

```bash
hermes send --list telegram
```

```bash
hermes send -t telegram "Multiversa conectado."
```

---

## Empujar alertas al canal

`hermes send` reutiliza las credenciales del gateway y **no necesita el gateway
corriendo** para plataformas con bot token como Telegram. Sirve desde cualquier
script, cron o CI:

```bash
multiversa alerts --json | hermes send -t telegram -s "Multiversa · alertas"
```

Para mandar solo cuando haya algo que reportar:

```bash
multiversa alerts --json | jq -e '.alerts | length > 0' >/dev/null && multiversa alerts --json | hermes send -t telegram -s "Multiversa · alertas"
```

Formatos de destino que acepta `--to`:

| Forma | Ejemplo |
|---|---|
| Canal por defecto | `telegram` |
| Chat concreto | `telegram:-1001234567890` |
| Hilo dentro del chat | `telegram:-1001234567890:17585` |

---

## Límites que conviene tener presentes

- El MCP de Multiversa es **read-only**. El canal reporta y consulta; no aplica
  cambios en la máquina.
- El bot token de Telegram vive en `~/.hermes/.env` en claro. Trátalo como
  cualquier otro secreto de la máquina.
- Telegram no es un canal auditado. Para hallazgos `P0` que impliquen datos de
  cliente, el canal avisa — el detalle se revisa en la máquina, no en el chat.

---

## Referencias

- Hermes Agent — Nous Research, MIT: https://github.com/NousResearch/hermes-agent
- ADR-003 · zonas de confianza Group/Lab: [`adr-003-group-lab-trust-zones.md`](adr-003-group-lab-trust-zones.md)
- Arquitectura del Lab adaptativo: [`ARQUITECTURA-LAB-ADAPTATIVO.md`](ARQUITECTURA-LAB-ADAPTATIVO.md)
