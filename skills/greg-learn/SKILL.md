---
name: greg-learn
description: Consolidate general communication patterns, decision mannerisms, recurring corrections/affirmations, and useful tools from the current conversation into Raul's persistent memory. Filters out particulars (state, projects, tasks, one-off facts). Use when user invokes /greg-learn or says "consolida lo aprendido".
---

Consolido aprendizajes generales sobre cómo se comunica Raul, cómo decide, qué corrige, qué afirma, y qué herramientas vale rescatar — de la conversación actual hacia memoria persistente.

## Flujo

1. Reviso la conversación actual desde el inicio.
2. Extraigo aprendizajes según las taxonomías abajo.
3. Filtro: descarto particulares, mantengo generalidades.
4. Leo `MEMORY.md` y los archivos `feedback_*.md` / `reference_*.md` relevantes para no duplicar ni contradecir sin darme cuenta.
5. Decido dónde va cada aprendizaje: actualizar archivo existente o crear uno nuevo (`feedback_*` para preferencias/estilo, `reference_*` para herramientas/recursos).
6. Preview a Raul: "encontré N aprendizajes, propongo estos cambios en estos archivos". Breve.
7. Espero OK explícito (per regla preview-before-write de `operating-style`).
8. Ejecuto los writes.
9. Resumen breve: qué archivos toqué, qué cambió.

## Sí extraer (generalidades)

- **Mannerisms y comunicación:** palabras/frases que él usa, palabras que no usa, ritmo, formato preferido, idioma, longitud, signos de afirmación o irritación.
- **Patrones de decisión:** qué prioriza al elegir entre opciones, qué le convence, qué le molesta, cómo procesa trade-offs.
- **Correcciones recurrentes:** si me corrigió, ¿qué patrón hay detrás? (no la corrección puntual — el patrón).
- **Afirmaciones recurrentes:** si dijo "exacto / perfecto / así / listo", ¿qué de mi comportamiento rescato como cosa-a-repetir?
- **Herramientas útiles:** frameworks que usó, procesos que funcionaron, comandos / skills / snippets / mental models que valga la pena reusar.

## No extraer

- Estado del día ("estoy cansado", "hoy ansioso", "buen día").
- Hechos puntuales ("Sparck enviada", "tesis lista", "deadline X").
- **Proyectos, iniciativas, tareas específicas.**
- Pedidos one-off.
- Información ya en memoria igual.

## Casos borde

- **Nada nuevo:** decir "nada nuevo, no consolido" y salir.
- **Contradicción con memoria:** flagear y preguntar cuál mantener.
- **Invocado mid-conversación:** trabajar con lo dicho hasta ese momento.
- **Múltiples archivos posibles:** elegir un home único, enlazar desde otros con `[[name]]`.
- **Aprendizaje borderline (general o particular):** si dudo, lo flag en el preview y dejo que Raul decida.

## Cómo escribir las memorias

- Slug en frontmatter (`name:`) en kebab-case.
- Body: regla/observación primero, luego `**Why:**` y `**How to apply:**` (per regla de memorias tipo feedback).
- Link a memorias relacionadas con `[[name]]`.
- Actualizar `MEMORY.md` con una línea por archivo nuevo (≤150 chars).

## Tono al hacer preview

- Plain Spanish, sin jerga.
- Una recomendación clara, no menú.
- Breve.
- Síntesis: conectar puntos cuando aporte valor.
