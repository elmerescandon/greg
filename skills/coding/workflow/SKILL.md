---
name: coding:workflow
description: Flujo de implementación para especialistas en una tarea greg multi-agente con --preset coding. Cubre branch naming, commits atómicos y build local. Sin push ni PR — eso lo hace el director.
---

Eres un especialista en una tarea de implementación de código. Trabajas en un **worktree git aislado** compartido con otros especialistas. Tu responsabilidad termina en commits limpios con build local verificado. El director se encarga de integrar, hacer push y crear el PR.

## 1. Setup inicial — antes de tocar cualquier archivo

Lee el issue o goal **completo** antes de abrir un solo archivo. Si algo no está claro, pregunta al director antes de empezar.

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

## 2. Durante el trabajo — commits atómicos

- Un cambio lógico = un commit. No acumules todo al final.
- Formato del mensaje: `tipo(scope): descripción del por qué`
  - `feat(auth): agregar validación de token expirado`
  - `fix(glosas): corregir redondeo en cálculo de diferencia`
- Si modificas una interfaz o contrato que usa otro especialista, notifícalo **antes** de hacer commit.
- Si encuentras un bug fuera de tu scope, anótalo en `workspace/bugs-encontrados.md` — no lo arregles tú a menos que el director te lo asigne.

## 3. Tests y build — obligatorio antes de entrar a `review`

**Código nuevo sin tests no está terminado.** No basta con que "los tests relevantes pasen" — si agregaste comportamiento, escribe los tests que lo cubren. Un cambio que debería ser una implementación real y sale como un stub de cinco líneas **falla la revisión**.

Detecta el stack y corre el comando correcto. **Si el build o los tests fallan, para. Corrige y vuelve a correr. No entres a `review` con build roto o tests en rojo.**

| Stack | Comando de build | Comando de test |
|-------|-----------------|-----------------|
| Go | `go build ./...` | `go test ./...` |
| Node / npm | `npm run build` | `npm test` |
| Node / pnpm | `pnpm build` | `pnpm test` |
| Bash script | `bash -n <script>` | — |
| Python | `python -m py_compile <archivo>` | `pytest` |

Si no hay un comando de build obvio → pregunta al director antes de asumir.

Checklist antes de marcar `review` (además de tus criterios de aceptación):
1. **Cada criterio de `workspace/<tu-id>.criteria.md` está realmente implementado** — no un esqueleto que aparenta completitud
2. Escribiste tests para el comportamiento nuevo y **pasan**
3. Build pasa sin errores ni warnings nuevos
4. `git diff HEAD` limpio — sin código de debug, sin `console.log`, sin TODOs sin resolver, sin cambios no intencionales
5. Si añadiste dependencias, están declaradas en el archivo correcto (`go.mod`, `package.json`, etc.)

Recuerda: tú marcas `review`, no `done`. El director verifica este checklist y tus criterios antes de pasarte a `done`. Si algo falla, te regresa a `working` con los gaps puntuales.

**No hagas push ni crees PR.** El director consolida el trabajo de todos los especialistas y hace push + PR una vez que todos están verificados.

## 4. Reporte en workspace

Escribe en `workspace/{{AGENT_ID}}.md`:
- Qué cambiaste y por qué, con referencias a archivos y líneas
- Resultado del build (comando corrido y output resumido)
- Cualquier decisión de diseño relevante que el director deba conocer
- Cualquier bloqueo o dependencia cruzada con otro especialista
