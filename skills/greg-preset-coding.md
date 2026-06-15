## Coding preset — Contexto de tarea de desarrollo

Estás en una tarea de implementación de código. Tu workspace es un **worktree git aislado** — trabaja ahí, nunca en el branch principal.

### Git workflow

- Haz commits pequeños y significativos mientras avanzas, no un commit gigante al final
- Mensajes de commit: describe el "por qué", no el "qué" — el diff ya muestra el qué
- No hagas push — el humano decide cuándo mergear el worktree

### Antes de marcar done

1. El código compila sin errores — corre `go build ./...`, `bash -n <script>`, `npm run build`, o el equivalente del stack
2. Los tests relevantes a tu cambio pasan — no hace falta correr toda la suite si tu cambio es acotado
3. `git diff HEAD` limpio — sin código de debug, sin TODOs sin resolver, sin cambios no intencionales
4. Si añadiste dependencias, están declaradas en el archivo correcto (go.mod, package.json, etc.)

### Estándares de calidad

- No dejes código comentado — elimínalo o conviértelo en un TODO documentado
- No añadas manejo de errores para escenarios imposibles — confía en las garantías del framework
- No diseñes para requerimientos futuros hipotéticos — resuelve lo que se pide
- Si algo no está claro, lee el código existente antes de asumir — la respuesta casi siempre está ahí

### Colaboración con el equipo

- Escribe en `workspace/{{AGENT_ID}}.md` qué cambiaste y por qué, con referencias a archivos y líneas
- Si tu cambio modifica una interfaz o contrato que usa otro agente, notifícalo antes de hacer commit
- Si encontraste un bug fuera de tu scope, anótalo en `workspace/bugs-encontrados.md` — no lo arregles tú a menos que el director te lo asigne
