## define-issue — Cómo construir un issue bien definido para un agente de código

El 82% de los fallos de un agente ocurren antes de escribir la primera línea — por un issue mal definido. Este skill guía a construir el issue correcto antes de ejecutar.

---

### Cuándo usarlo

Usa este skill cuando el goal de la tarea o el issue de GitHub no tenga suficiente claridad para actuar. Si no puedes responder SÍ a estas 5 preguntas, el issue necesita refinarse:

1. ¿Sabes qué está roto o falta HOY (comportamiento actual observable)?
2. ¿Sabes qué debe pasar DESPUÉS (comportamiento deseado observable)?
3. ¿Sabes qué usuario se afecta y cómo?
4. ¿Conoces al menos UN archivo del codebase involucrado?
5. ¿El scope cabe en esta sesión (un párrafo claro, menos de 5 criterios)?

**Si hay una o más respuestas negativas → detente y pregunta al humano antes de continuar.**

---

### Cómo preguntar (de a una)

No hagas todas las preguntas juntas. Identifica cuál es el hueco más crítico y pregunta solo eso.

- Si no está claro qué está roto hoy: "¿Qué pasa HOY cuando el usuario intenta hacer eso? ¿Qué ve o qué falla?"
- Si no está claro el resultado esperado: "¿Cómo sabrías que está resuelto? ¿Qué debería poder hacer el usuario que hoy no puede?"
- Si no están claros los archivos: "¿En qué componente, endpoint o página ocurre esto?"
- Si el scope parece grande: "¿Esto son varias cosas? ¿Cuál es la más importante resolver primero?"

---

### Estructura del issue bien formado

Una vez que tienes claridad, formula el issue así:

```
Título: [verbo] [qué] para que [quién] pueda [resultado]

Problema
  Qué está roto hoy + quién se afecta. 2-3 oraciones.

Comportamiento actual
  Observable y específico.

Comportamiento deseado
  Observable y específico.

Scope
  Archivos a cambiar: [rutas exactas]
  No tocar: [exclusiones explícitas]
  Patrón a seguir: [ruta de código existente a replicar]

Criterios de aceptación
  - Given [contexto] / When [acción] / Then [resultado]
  - Given [contexto] / When [acción] / Then [resultado]

Done checklist
  1. [comando de build o test a correr]
  2. Confirmar: ningún archivo fuera del scope fue modificado

Guardrails
  Nunca: force-push, comandos destructivos en DB, commitear .env
  Preguntar antes de: [cambios de alto impacto no cubiertos arriba]
```

---

### Reglas críticas

- **El issue describe el PROBLEMA, no la implementación.** Si el humano describe una solución, reformúlala como problema.
- **El scope es una cerca, no una sugerencia.** Nombra explícitamente qué NO tocar.
- **Un issue = una sesión de agente.** Si caben más de 5 criterios o toca más de 3 módulos no relacionados, propón dividirlo.
- **No inventes archivos o métodos.** Lee el codebase para confirmar que existen antes de referenciarlos.
- **Siempre declara el "done checklist".** Sin él, el agente declara victoria demasiado pronto.

---

### Failure modes a prevenir

| El agente hace esto | Causa en el issue | Fix |
|---|---|---|
| Declara done al 80% | Sin checklist de done | Agrega comandos explícitos |
| Rompe tests existentes | Sin instrucción de baseline | "Corre la suite antes y después" |
| Se va de scope | Sin cerca explícita | Agrega "No tocar: X" |
| Sobre-ingenia | Sin línea de exclusiones | Nombra qué no debe cambiar |
| Duplica código existente | Sin referencia a patrón | Apunta al archivo a reutilizar |
| Ejecuta comandos peligrosos | Sin guardrails | Agrega sección "Nunca / Preguntar" |

---

### Referencia completa

Los 12 lineamientos con ejemplos están en el vault: `Salmona/lineamientos-issues-agentes.md`
