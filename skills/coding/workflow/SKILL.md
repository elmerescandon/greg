---
name: coding:workflow
description: Flujo completo de implementación para agentes de código en una tarea greg multi-agente. Cubre branch naming, commits atómicos, build obligatorio, push y creación de PR. Úsalo cuando estés en un worktree aislado de greg.
---

Estás en una tarea de implementación de código. Tu workspace es un **worktree git aislado** — trabaja ahí, nunca en el branch principal.

## 1. Setup inicial — antes de tocar cualquier archivo

**Crea un branch con el prefijo correcto según la naturaleza del trabajo:**

| Tipo | Prefijo | Cuándo |
|------|---------|--------|
| Feature nueva | `feat/` | Agrega funcionalidad que no existía |
| Bug conocido | `bugfix/` | Corrige comportamiento incorrecto no urgente |
| Bug crítico en producción | `hotfix/` | Requiere fix inmediato en prod |
| Refactor | `refactor/` | Mejora interna sin cambio de comportamiento |
| Documentación | `docs/` | Solo cambios en docs o comentarios |

```bash
git checkout -b <prefijo>/<descripción-corta-en-kebab-case>
# Ejemplo: feat/agregar-autenticacion-jwt
# Ejemplo: bugfix/corregir-calculo-de-glosas
```

Lee el issue o goal **completo** antes de abrir un solo archivo. Si algo no está claro, pregunta antes de empezar.

## 2. Durante el trabajo — commits atómicos

- Un cambio lógico = un commit. No acumules todo al final.
- Formato del mensaje: `tipo(scope): descripción del por qué`
  - `feat(auth): agregar validación de token expirado`
  - `fix(glosas): corregir redondeo en cálculo de diferencia`
- Si modificas una interfaz o contrato que usa otro agente, notifícalo **antes** de hacer commit.
- Si encuentras un bug fuera de tu scope, anótalo en `workspace/bugs-encontrados.md` — no lo arregles tú a menos que el director te lo asigne.

## 3. Build — obligatorio antes de push

Detecta el stack y corre el comando correcto. **Si el build falla, para. Corrige y vuelve a correr. No hagas push con build roto.**

| Stack | Comando de build | Comando de test |
|-------|-----------------|-----------------|
| Go | `go build ./...` | `go test ./...` |
| Node / npm | `npm run build` | `npm test` |
| Node / pnpm | `pnpm build` | `pnpm test` |
| Bash script | `bash -n <script>` | — |
| Python | `python -m py_compile <archivo>` | `pytest` |

Si no hay un comando de build obvio → pregunta antes de asumir.

Checklist antes de push:
1. Build pasa sin errores ni warnings nuevos
2. Tests relevantes a tu cambio pasan
3. `git diff HEAD` limpio — sin código de debug, sin `console.log`, sin TODOs sin resolver, sin cambios no intencionales
4. Si añadiste dependencias, están declaradas en el archivo correcto (`go.mod`, `package.json`, etc.)

## 4. Push

```bash
git push -u origin <nombre-del-branch>
```

Solo cuando el build y los tests pasan limpio.

## 5. Pull Request

```bash
gh pr create \
  --title "<tipo>(<scope>): <descripción imperativa corta>" \
  --body "$(cat <<'EOF'
## Qué cambia
[1-3 bullets con los cambios principales]

## Por qué
[Contexto del problema que resuelve]

## Cómo probar
- [ ] [Paso concreto]
- [ ] [Paso concreto]

## Notas
[Decisiones de diseño no obvias, limitaciones conocidas, o nada si no aplica]
EOF
)"
```

- Si el trabajo está **parcialmente completo**: `--draft`
- Target branch: `main` por defecto, salvo que el issue especifique otro
- No hagas merge tú — el humano decide cuándo mergear

## 6. Reporte en workspace

Escribe en `workspace/{{AGENT_ID}}.md`:
- Qué cambiaste y por qué, con referencias a archivos y líneas
- Link al PR creado
- Cualquier decisión de diseño relevante que el director deba conocer
