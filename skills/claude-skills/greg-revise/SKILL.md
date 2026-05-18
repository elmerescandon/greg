---
name: greg-revise
description: Closes a `greg task revise` session cleanly. Confirms the revision is done, marks the parent task as completed, then kills the current greg session via `greg kill`. Use when the user invokes /greg-revise or asks to finish/close this revision session.
---

Cierro esta sesión de revisión de forma limpia.

## Flujo

1. Doy un mensaje breve confirmando que la revisión terminó — qué se revisó y observación final si aplica.

2. Obtengo el ID de esta sesión:
   ```bash
   SESSION_ID=$(tmux display-message -p '#S')
   ```

3. Busco el nombre de la sesión en `sessions.json` para derivar el task ID:
   ```bash
   SESSION_NAME=$(jq -r ".[] | select(.id == \"$SESSION_ID\") | .name" ~/.greg/sessions.json)
   TASK_ID=$(echo "$SESSION_NAME" | grep -oE 'mtask-[a-f0-9]+')
   ```

4. Si encontré un task ID, marco la tarea como completada:
   ```bash
   META_FILE="$HOME/.greg/multi-tasks/${TASK_ID}.json"
   if [[ -f "$META_FILE" ]]; then
     tmp=$(mktemp)
     jq --arg ts "$(date '+%Y-%m-%d %H:%M:%S')" \
       '. + {coordinator_status: "completed", completed: $ts}' \
       "$META_FILE" > "$tmp" && mv "$tmp" "$META_FILE"
     echo "task $TASK_ID → completed"
   fi
   ```

5. Ejecuto `greg kill $SESSION_ID` para archivar en history y cerrar el proceso tmux.

## Notas

- Si `tmux display-message` retorna vacío (no estoy en tmux), aviso y no ejecuto nada.
- Si el nombre de la sesión no contiene un `mtask-*`, omito el paso 4 y solo hago el kill.
- No pido confirmación — invocar el skill es la confirmación implícita.
- Mensaje final: breve, sin florituras.
