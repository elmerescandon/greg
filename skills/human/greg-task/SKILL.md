---
name: greg-task
description: Guided assistant for creating multi-agent greg tasks. Helps design the goal, define agents with good IDs and roles, and run the correct command. Use when the user invokes /greg-task or asks to create a multi-agent task with greg.
---

Ayudo a Raul a diseñar y lanzar una tarea multi-agente con greg correctamente.

## Cómo funciona una tarea greg

Cuando corres `greg task run`, sucede lo siguiente:

1. Greg crea un workspace en `~/.greg/multi-tasks/mtask-<id>/` con subdirectorios:
   - `workspace/` — cada agente escribe su output (`<agent-id>.md`), tiene sus criterios de aceptación (`<agent-id>.criteria.md`) y recibe el veredicto del director (`<agent-id>.review.md`)
   - `messages/` — mensajes entre agentes (`remitente→destinatario.md`)
   - `status/` — archivo por agente con su estado (`working`, `waiting`, `needs-help`, `review`, `done`)

2. Se inyecta automáticamente un agente **director** que no debes declarar tú. Coordina al equipo y **verifica** el trabajo de cada especialista contra sus criterios de aceptación.

3. Cada agente recibe un prompt con su rol, sus criterios de aceptación y acceso al workspace compartido. Trabajan **en paralelo** en sesiones tmux independientes.

4. **El ciclo de vida es `working → review → done`.** Un especialista nunca marca `done` solo: cuando cree que cumplió, entra a `review`. El director contrasta su output contra `<agent-id>.criteria.md`, criterio por criterio, y solo entonces lo pasa a `done` (verificado) o lo regresa a `working` con los gaps. Esto es lo que evita que un agente declare victoria prematura — la razón principal de este diseño.

5. Un coordinator en background vigila los status. Cuando **todos** (incluyendo el director) están en `done`, cierra la tarea (`coordinator_status: completed`).

> **La calidad del resultado depende de la calidad de los criterios de aceptación.** Si los criterios son vagos, el director no tiene contra qué verificar y volvemos al problema original. Por eso el diseño de la tarea es una elicitación grado-issue, no solo redactar roles.

---

## Comando

```bash
greg task run \
  --goal "<objetivo completo y concreto>" \
  --agent "<id>:<descripción del rol>" \
  --criteria-file "<id>:<ruta-al-criteria.md>" \
  --agent "<id>:<descripción del rol>" \
  --criteria-file "<id>:<ruta-al-criteria.md>" \
  [--agent ...] \
  [--preset coding|research] \
  [--dir <path>] \
  [--model <alias|id>]
```

- `--goal` — requerido. Una oración que define qué debe producir la tarea.
- `--agent` — uno o más especialistas. El director se agrega solo.
- `--criteria-file` — **uno por agente**. Ruta a un `.md` con los criterios de aceptación de ese agente. El CLI lo copia a `workspace/<id>.criteria.md` antes de hacer spawn, así existe cuando el agente arranca. El `<id>` debe coincidir con un `--agent`.
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

### Criterios de aceptación (la pieza que asegura la calidad)

Cada agente lleva un archivo `<id>.criteria.md` que es **su contrato**: lo que el director verifica antes de dejarlo pasar a `done`. El rol dice *qué hace*; los criterios dicen *cómo sé que lo hizo bien*. Sin criterios fuertes, el agente satisfice y el director no tiene contra qué verificar.

Para construirlos reutilizo la metodología de `/coding:issue` (la skill `skills/coding/issue/SKILL.md`): elicito **de a una pregunta** hasta tener, por agente, un archivo con esta estructura:

```
# Criterios de aceptación — <id>

## Definición de "terminado"
Qué tiene que ser cierto para considerar este trabajo completo. Concreto y observable.

## Criterios
- [ ] Given <contexto> / When <acción> / Then <resultado esperado>
- [ ] ...   (en research: fuentes mínimas, secciones obligatorias, profundidad esperada)
- [ ] ...   (en coding: comportamiento implementado de verdad + tests que lo cubren)

## Scope — cerca
Qué SÍ se toca / qué NO se toca.

## Done checklist
1. <comando/verificación concreta>
2. ...

## Guardrails
Nunca: ... / Preguntar antes de: ...
```

La profundidad de la elicitación es deliberada — es el upfront que evita las rondas de corrección al final.

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
2. Identifico las perspectivas naturales que lo componen — cada una es un agente, con su rol delimitado.
3. **Por cada agente, elicito sus criterios de aceptación grado-issue** — de a una pregunta, sin avalancha. No asumo la vara: la pregunto. (Ej.: "¿Qué cuenta como cobertura completa acá?", "¿Qué tendría que poder hacer el usuario que hoy no puede?", "¿Qué profundidad de tests esperas?").
4. Escribo cada criterio a un archivo temporal, p. ej. `/tmp/greg-criteria-<id>.md`, con la estructura de arriba.
5. Propongo el comando completo (`--agent` + `--criteria-file` por agente) para revisión antes de ejecutar.
6. Si Raul aprueba, ejecuto.

> No me salto el paso 3 por velocidad. La elección de Raul fue elicitación pesada *a cambio de* un resultado que genuinamente no requiera corrección al final. Roles sin criterios = volver al problema original.

---

## Ejemplo completo

**Objetivo**: Reporte sobre el estado de la IA generativa en México en 2026

Primero escribo un criteria por agente (ej. `/tmp/greg-criteria-empresas.md`):

```
# Criterios de aceptación — empresas

## Definición de "terminado"
Panorama de adopción empresarial respaldado con datos y casos reales, no generalidades.

## Criterios
- [ ] Al menos 3 sectores líderes con datos de adopción citados (fuente + año)
- [ ] Mínimo 4 casos de uso reales con empresa nombrada
- [ ] Barreras de adopción analizadas, no listadas
- [ ] Cada afirmación cuantitativa tiene fuente

## Scope — cerca
SÍ: adopción empresarial. NO: regulación ni talento (otros agentes).

## Done checklist
1. Toda cifra tiene fuente verificable
2. Ningún caso de uso es hipotético
```

Luego lanzo (un `--criteria-file` por agente):

```bash
greg task run \
  --goal "Producir un reporte sobre el estado de la IA generativa en México en 2026: adopción empresarial, talento disponible, regulación y casos de uso relevantes" \
  --agent "empresas:Investiga la adopción de IA generativa en empresas mexicanas. Cubre sectores líderes, barreras de adopción y casos de uso reales. No incluyas regulación ni análisis de talento." \
  --criteria-file "empresas:/tmp/greg-criteria-empresas.md" \
  --agent "talento:Analiza el ecosistema de talento en IA en México: universidades, bootcamps, salarios, migración. No entres en adopción empresarial ni regulación." \
  --criteria-file "talento:/tmp/greg-criteria-talento.md" \
  --agent "regulacion:Mapea el marco regulatorio de IA en México en 2026: leyes vigentes, propuestas, comparación con LATAM y tendencias. No analices adopción ni talento." \
  --criteria-file "regulacion:/tmp/greg-criteria-regulacion.md"
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

## Mientras corre: la ventana de `review`

Cuando un agente entra a `review`, `greg task status` lo marca y el director está verificándolo contra sus criterios. Esa es **tu ventana** para intervenir antes de que el trabajo quede cerrado — sin ser obligatorio. Si quieres meter mano, mándale al director por el canal humano:

```bash
greg task message <task-id> "Para el agente empresas: exige también cifras de inversión, no solo adopción."
```

El veredicto del director por agente queda en `workspace/<agent-id>.review.md` (criterio por criterio).

## Leer los resultados cuando la tarea termina

Una vez que `coordinator_status: completed`, cada agente dejó su output en `workspace/<agent-id>.md` (ya verificado contra sus criterios). El director dejó sus notas de síntesis en `workspace/director-synthesis-notes.md` — ese es el documento consolidado.

```bash
greg task status <task-id>     # ver qué agente produjo qué
cat ~/.greg/multi-tasks/<task-id>/workspace/director-synthesis-notes.md
cat ~/.greg/multi-tasks/<task-id>/workspace/<agent-id>.review.md   # cómo se verificó
cat ~/.greg/multi-tasks/<task-id>/workspace/<agent-id>.md
```
