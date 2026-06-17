---
name: greg-task
description: Guided assistant for creating multi-agent greg tasks. Helps design the goal, define agents with good IDs and roles, and run the correct command. Use when the user invokes /greg-task or asks to create a multi-agent task with greg.
---

Ayudo a Raul a diseñar y lanzar una tarea multi-agente con greg correctamente.

## Cómo funciona una tarea greg

Cuando corres `greg task run`, sucede lo siguiente:

1. Greg crea un workspace en `~/.greg/multi-tasks/mtask-<id>/` con subdirectorios:
   - `workspace/` — cada agente escribe su output aquí (`<agent-id>.md`)
   - `messages/` — mensajes entre agentes (`remitente→destinatario.md`)
   - `status/` — archivo por agente con su estado (`working`, `waiting`, `needs-help`, `done`)

2. Se inyecta automáticamente un agente **director** que no debes declarar tú. Su rol es coordinar al equipo, desbloquear agentes y preparar notas de síntesis.

3. Cada agente recibe un prompt con su rol y acceso al workspace compartido. Trabajan **en paralelo** dentro de sesiones tmux independientes.

4. Un coordinator en background vigila los status. Cuando **todos** los agentes (incluyendo el director) marcan `done`, cierra la tarea automáticamente (`coordinator_status: completed`).

---

## Comando

```bash
greg task run \
  --goal "<objetivo completo y concreto>" \
  --agent "<id>:<descripción del rol>" \
  --agent "<id>:<descripción del rol>" \
  [--agent ...] \
  [--preset coding|research] \
  [--dir <path>] \
  [--model <alias|id>]
```

- `--goal` — requerido. Una oración que define qué debe producir la tarea.
- `--agent` — uno o más especialistas. El director se agrega solo.
- `--preset` — opcional. Inyecta instrucciones especializadas a todos los agentes (excepto el director). Valores válidos: `coding`, `research`.
- `--dir` — opcional. Directorio de trabajo (default: `$GREG_VAULT` o `$HOME`).
- `--model` — opcional. Alias o model ID. Aliases: `opus`, `sonnet`, `haiku`.

---

## Presets

### `--preset coding`

Úsalo cuando la tarea produce **código** que debe mergearse a un repositorio.

Hace dos cosas automáticamente:

1. **Crea un worktree git aislado** en `/tmp/greg-worktree-<task-id>` — los agentes trabajan ahí, nunca en el branch principal. El humano decide cuándo mergear.
2. **Inyecta el skill `greg-coding`** en el rol de cada agente (excepto el director). Ese skill cubre: git workflow, checklist pre-done, estándares de calidad y protocolo de colaboración entre agentes.

El skill inyectado vive en `~/Documents/greg/skills/greg-coding/SKILL.md`.

### `--preset research`

Úsalo cuando la tarea produce **análisis o síntesis** a partir de fuentes externas.

Detecta el tipo de agente por palabras clave en el rol:
- Roles con "recolect", "gather", "search", "busca" → reciben el skill de collector.
- Roles con "analiz", "review", "critic", "evalúa", "sintetiz" → reciben el skill de analyzer.

---

## Diseño de agentes

### IDs

- Cortos, en minúsculas, sin espacios: `modelos`, `industria`, `investigacion`, `frontend`, `seguridad`
- Deben describir **la perspectiva** del agente, no una tarea
- Evitar nombres genéricos como `agente1`, `experto`, `analista`

### Roles (la descripción después de `:`)

El rol es el prompt inicial del agente. Debe decir:
- **Qué perspectiva cubre** — su ángulo específico dentro del goal
- **Qué debe producir** — tipo de output esperado
- **Qué NO debe hacer** — delimitación de scope para evitar solapamiento con otros agentes

Ejemplo bien definido:
```
modelos:Analiza el estado actual de los modelos de lenguaje más relevantes en 2026 (GPT, Claude, Gemini, Llama). Cubre capacidades, benchmarks y posicionamiento relativo. No entres en aplicaciones industriales ni investigación académica, eso lo cubre otro agente.
```

Ejemplo mal definido:
```
modelos:Investiga sobre los modelos de lenguaje
```

### Cuántos agentes usar

- **2-3 agentes**: tareas con perspectivas claramente separables y comparable complejidad
- **4-5 agentes**: investigaciones amplias donde cada ángulo justifica trabajo profundo
- Más de 5: poco común, solo si los dominios son verdaderamente independientes
- El director ya cuenta — si declaras 4 agentes, corren 5 sesiones en total

### Solapamiento entre agentes

Antes de lanzar, verifica que los roles no se pisen. El director puede manejar algo de solapamiento, pero si dos agentes cubren exactamente lo mismo, uno de ellos desperdicia contexto. Delimita con frases como "No cubras X, eso lo maneja <otro-agente>".

---

## Flujo de diseño que sigo

1. Entiendo el objetivo final: ¿qué documento / análisis / output quiere Raul?
2. Identifico las perspectivas naturales que lo componen — cada una es un agente
3. Redacto roles con delimitación explícita
4. Propongo el comando completo para revisión antes de ejecutar
5. Si Raul aprueba, ejecuto

---

## Ejemplo completo

**Objetivo**: Reporte sobre el estado de la IA generativa en México en 2026

```bash
greg task run \
  --goal "Producir un reporte sobre el estado de la IA generativa en México en 2026: adopción empresarial, talento disponible, regulación y casos de uso relevantes" \
  --agent "empresas:Investiga la adopción de IA generativa en empresas mexicanas. Cubre sectores líderes, barreras de adopción y casos de uso reales. No incluyas regulación ni análisis de talento." \
  --agent "talento:Analiza el ecosistema de talento en IA en México: universidades, bootcamps, salarios, migración. No entres en adopción empresarial ni regulación." \
  --agent "regulacion:Mapea el marco regulatorio de IA en México en 2026: leyes vigentes, propuestas, comparación con LATAM y tendencias. No analices adopción ni talento."
```

---

## Comandos de seguimiento

```bash
greg task status <task-id>     # ver estado de cada agente en tiempo real
greg task list                 # listar todas las tareas multi-agente
greg list                      # ver todas las sesiones y su status [running/active/finished]
greg peek <task-id>            # ver las últimas 30 líneas de todos los agentes de una tarea
greg peek <session-id>         # ver las últimas 30 líneas de un agente específico
greg peek <id> -n 50           # cambiar la cantidad de líneas
greg attach <session-id>       # entrar a ver una sesión en vivo
```

## Leer los resultados cuando la tarea termina

Una vez que `coordinator_status: completed`, cada agente dejó su output en `workspace/<agent-id>.md`. El director dejó sus notas de síntesis en `workspace/director-synthesis-notes.md` — ese es el documento consolidado.

```bash
greg task status <task-id>     # ver qué agente produjo qué
cat ~/.greg/multi-tasks/<task-id>/workspace/director-synthesis-notes.md
cat ~/.greg/multi-tasks/<task-id>/workspace/<agent-id>.md
```
