

<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

# Balaur

> **Un agente personal soberano con prioridad local, ejecutado a través de `balaur`.**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](./LICENSE)
[![Node](https://img.shields.io/badge/node-%E2%89%A523.4-339933.svg)](./package.json)

Balaur es un agente personal basado en Bun/Ink con una bóveda (vault) Markdown Johnny Decimal, habilidades (skills) y una conversación continua sobre tu vida. Utiliza `@earendil-works/pi-ai` para el acceso a modelos/proveedores y `@earendil-works/pi-agent-core` para el ciclo del agente. **No** utiliza extensiones de agentes de código, paquetes TUI externos ni MCP.

El nombre proviene del balaur de los cuentos populares rumanos: un dragón con múltiples cabezas. Balaur mantiene una cabeza principal: la conversación maestra sobre tu vida. El trabajo enfocado ocurre como subcabezas temporales, creadas con `/branch`, que luego se compactan y fusionan de nuevo en la cabeza maestra con `/merge`. Esta restricción es intencional: una conversación, un alcance, muchas ramas enfocadas cuando sea necesario.

Tu vida no es un producto. El registro de tu vida debe vivir en archivos que tú poseas.

## Estado actual

- **CLI:** `balaur` / `bun run balaur`
- **UI:** Interfaz de terminal basada en Ink/React con la marca Balaur
- **Motor del agente:** `pi-ai` + `pi-agent-core`
- **Bóveda:** Markdown Johnny Decimal en el directorio de datos de Balaur
- **Conversación:** una cabeza de conversación maestra, más subcabezas de rama compactables
- **Habilidades:** Habilidades en Markdown, incluyendo aquellas almacenadas como entradas en la bóveda con `kind: skill`
- **Sin MCP**
- **Puerta de enlace de API REST:** endpoint de Bun en loopback para probar el contrato de la puerta de enlace en tiempo de ejecución

## Inicio rápido

```bash
bun install
bun run balaur
```

Puerta de enlace de API REST local opcional:

```bash
bun run api
# POST http://127.0.0.1:8787/api/messages
# GET  http://127.0.0.1:8787/api/events?clientId=local
```

Comandos útiles en el chat:

```txt
/help             mostrar comandos y atajos de la TUI
/clear            limpiar el chat visible
/model            mostrar el modelo/proveedor actual
/branch <title>   iniciar una subconversación enfocada
/merge            compactar y fusionar la subconversación activa en la maestra
/branches         mostrar el estado actual de las ramas
/skill:name       aplicar una habilidad en Markdown
/exit             salir
```

Atajos útiles en la TUI:

```txt
Ctrl+C / Ctrl+D   salir
Ctrl+L            limpiar el chat visible
Ctrl+U            limpiar la entrada
Ctrl+A / Ctrl+E   mover al inicio/fin de la entrada
← / →             mover el cursor de entrada
Backspace/Delete  editar en el cursor
```

La selección de modelo utiliza cualquier par proveedor/modelo integrado de `@earendil-works/pi-ai`, por ejemplo:

```bash
BALAUR_MODEL=synthetic/syn:large:text bun run balaur
BALAUR_MODEL=openai/gpt-4o-mini bun run balaur
```

Las claves API se leen desde variables de entorno estándar del proveedor soportadas por `pi-ai`, o `BALAUR_<PROVIDER>_API_KEY`.
Synthetic también acepta `SYNTHETIC_API_KEY` (o `BALAUR_SYNTHETIC_API_KEY`).

También puedes colocarlos en un archivo `.env` en la raíz del proyecto y ejecutar Balaur normalmente:

```bash
bun run balaur
```

`runtimeEnv()` prefiere las variables de entorno reales sobre los valores de `.env`, y `BALAUR_ENV_FILE`
puede apuntar a una ruta de archivo explícita cuando sea necesario. Un template completo está disponible en
`.env.example`.

```bash
cp .env.example .env
# editar .env
# luego ejecutar
bun run balaur
```

## Estructura del proyecto

```txt
src/          Entradas de CLI (`balaur`, reindex) y componentes TUI de Ink
lib/runtime/  Tiempo de ejecución de Balaur: bus de eventos, puente de gateway, envoltorio de agente, conversación maestra/de rama, habilidades
lib/tui/      Ayudantes puros de estado TUI
lib/vault.ts  Bóveda Markdown Johnny Decimal + índice FTS SQLite desechable
skills/       Habilidades semilla Markdown integradas
assets/        Hojas de sprites PNG de avatar/fuente
lib/avatar/    Renderizador de avatar ANSI: solo sextante + octante
lib/design/    Tokens de diseño React/Ink compartidos
docs/          Documentación del proyecto
```

## Desarrollo

```bash
bun run typecheck
bun run test
bun run reindex
```

Target de archivo único de Bun:

```bash
bun run build:balaur:bun
```

## Licencia

[AGPL-3.0-or-later](./LICENSE). Tu bóveda privada y conversaciones no forman parte de este repositorio.

Desarrollado de forma abierta, en Brașov.
