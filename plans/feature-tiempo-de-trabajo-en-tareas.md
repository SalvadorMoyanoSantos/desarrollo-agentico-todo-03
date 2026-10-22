# Feature: Tiempo de trabajo en las tareas y en el informe

## Encargo y criterios de aceptación

Encargo literal: «Añade en las tareas un campo para el tiempo de trabajo y añádelo en el informe».

Requisitos explícitos:
- Cada tarea tiene un campo de tiempo de trabajo, que se puede introducir y editar.
- El informe (`/report`) muestra ese tiempo.

Inferencias (decisiones de este plan):
- El tiempo se guarda como **minutos enteros** (`work_minutes`, `integer NOT NULL DEFAULT 0`). 0 significa «sin tiempo registrado».
- Se introduce en minutos en un `<input type="number">` opcional, en el formulario de creación y en el de edición.
- Rango válido: 0 a 99999 minutos (`WorkMinutesMax`). Valores negativos o superiores dan error de validación.
- Se muestra con formato legible (`1 h 30 min`, `45 min`, `—` si es 0) en la lista, y en el informe por tarea y como total.

Criterios de aceptación:
1. Crear una tarea con minutos (p. ej. 90) los persiste; sin minutos o vacío se guarda 0 y la creación sigue funcionando.
2. Editar una tarea permite cambiar título y minutos; el estado completada y la posición se conservan.
3. Minutos negativos, mayores que 99999 o no numéricos devuelven 422 con mensaje en español y no guardan nada.
4. La lista de tareas muestra el tiempo de las tareas que lo tienen.
5. El informe muestra una métrica «Tiempo total» y una columna «Tiempo» en el listado de tareas.
6. Las tareas existentes quedan con 0 tras la migración.

## Comportamiento actual y propuesta

Actual: `Task` (`models/task.go`) tiene `title`, `completed`, `position` y timestamps. `TasksCreate` enlaza el formulario al modelo; `TasksUpdate` enlaza solo `title` mediante una struct anónima. El informe (`TasksReport` + `Tasks.Report()` + `templates/tasks/report.plush.html`) muestra métricas (total, completadas, pendientes, %), tareas por día y un listado con Tarea/Estado/Creada.

Propuesta: añadir `WorkMinutes` al modelo con su migración, validarlo, aceptarlo en crear y editar, formatearlo con un método del modelo y mostrarlo en lista e informe; `TaskReport` gana `TotalMinutes`.

## Alcance

Incluido: migración, modelo, validación, formateo, acciones crear/editar, templates index/edit/report, CSS mínimo, fixtures y tests.

Excluido: temporizador/cronómetro, desglose por sesiones, tiempo por día en el informe, unidades distintas de minutos en la entrada (horas:minutos), edición del tiempo desde la lista sin pasar por el formulario, API JSON nueva.

## Diseño por capas

### Persistencia
Nueva migración SQL (mismo patrón que `20261019000000_add_position_to_tasks`):
- `migrations/20261022000000_add_work_minutes_to_tasks.up.sql`: `ALTER TABLE tasks ADD COLUMN work_minutes integer NOT NULL DEFAULT 0;`
- `migrations/20261022000000_add_work_minutes_to_tasks.down.sql`: `ALTER TABLE tasks DROP COLUMN work_minutes;`

No tocar `migrations/schema.sql` (archivo sin seguimiento ajeno a esta tarea).

### Modelo (`models/task.go`)
- Constante `WorkMinutesMax = 99999`.
- Campo `WorkMinutes int \`json:"work_minutes" db:"work_minutes" form:"work_minutes"\``.
- En `Validate`: si `WorkMinutes < 0 || WorkMinutes > WorkMinutesMax`, `verrs.Add("work_minutes", fmt.Sprintf("El tiempo de trabajo debe estar entre 0 y %d minutos.", WorkMinutesMax))`.
- Función `FormatMinutes(m int) string`: `0 → "—"`, `<60 → "45 min"`, múltiplos de 60 → `"2 h"`, resto `"1 h 30 min"`. Método `(t Task) WorkTime() string` que la usa (Plush puede llamar métodos).
- `TaskReport` gana `TotalMinutes int` y `TotalTime string` (formateado); `Report()` suma `WorkMinutes`.

### Acciones (`actions/tasks.go`)
- `TasksCreate`: ya usa `c.Bind(task)`, el nuevo campo se enlaza solo. Comprobar que un valor vacío del input se enlaza como 0 (Buffalo usa gorilla/schema con `ZeroEmpty`); si no fuese así, enlazar con struct auxiliar y convertir. Validación vía `ValidateAndCreate`.
- `TasksUpdate`: ampliar la struct de entrada con `WorkMinutes int \`form:"work_minutes" json:"work_minutes"\`` y asignarlo a `task.WorkMinutes`. Con errores, volver a renderizar `edit` con 422 (ya existe). Actualizar el comentario del método («actualiza título y tiempo de trabajo, conservando los demás datos»).
- `TasksReport`: sin cambios (ya pasa `tasks` y `report`).
- Sin rutas nuevas.

### Renderizado
- `templates/tasks/index.plush.html`:
  - En el formulario de creación, añadir `<input type="number" name="work_minutes" min="0" max="99999" step="1" inputmode="numeric" placeholder="Min" aria-label="Tiempo de trabajo en minutos" value="<%= task.WorkMinutes %>">` dentro de `.new-task-row`, antes del botón; mostrar el error de `work_minutes` junto al de título. Si `WorkMinutes` es 0 el valor inicial no debe verse como «0» molesto: renderizar `value` solo si `task.WorkMinutes > 0`.
  - En cada `<li>`, tras `.task-title`, `<%= if (t.WorkMinutes > 0) { %><span class="task-time" title="Tiempo de trabajo"><%= t.WorkTime() %></span><% } %>`.
- `templates/tasks/edit.plush.html`: añadir `<label for="task-minutes">Tiempo de trabajo (minutos)</label>` y el input numérico (mismos atributos, valor `task.WorkMinutes`), con `aria-describedby`/mensaje de error `work_minutes`. Cambiar el hint «Los demás datos de la tarea no cambian» por «Completada y posición no cambian» (o equivalente).
- `templates/tasks/report.plush.html`: nueva métrica `<div class="report-metric"><dt>Tiempo total</dt><dd><%= report.TotalTime %></dd></div>` y columna `<th scope="col">Tiempo</th>` con `<td><%= t.WorkTime() %></td>` en el listado.

### CSS (`public/assets/app.css`)
- `.task-time`: texto secundario pequeño (reutilizar variables de color existentes, como `.task-status`/`list-count`), sin romper el layout flex de `.task`; revisar los media queries móviles de `.task`.
- Ajustar `.new-task-row` para el input numérico estrecho (~5 rem) y el input del formulario de edición para que tenga el mismo estilo que el de título.
- No hace falta JavaScript.

## Archivos relevantes

- `migrations/20261022000000_add_work_minutes_to_tasks.{up,down}.sql` — columna nueva (crear).
- `models/task.go` — campo, validación, formateo, suma en el informe.
- `actions/tasks.go` — `TasksUpdate` (y verificación del bind en `TasksCreate`).
- `templates/tasks/index.plush.html`, `edit.plush.html`, `report.plush.html` — entrada y visualización.
- `public/assets/app.css` — estilos.
- `fixtures/tasks.toml` — añadir `work_minutes` a alguna fila (p. ej. 90 y 45) para probar el informe.
- `models/task_test.go`, `actions/tasks_test.go` — tests.

## Implementación paso a paso

1. Crear la migración up/down.
2. Modelo: constante, campo, validación, `FormatMinutes`/`WorkTime`, `TotalMinutes`/`TotalTime` en `Report()`.
3. Acción `TasksUpdate`: aceptar y asignar `work_minutes`; verificar el bind vacío en `TasksCreate`.
4. Templates index, edit y report según el diseño.
5. CSS mínimo.
6. Fixtures: `work_minutes = 90` en «Comprar material de oficina», `45` en «Publicar la oferta de empleo», 0 (por defecto) en la otra. Revisar que los tests existentes que dependen de las fixtures no se rompan.
7. Escribir los tests y ejecutar las validaciones.

## Tests y validación

Modelo (`models/task_test.go`):
- Persistencia de `WorkMinutes` al crear y recargar.
- Validación: -1 y 100000 inválidos; 0 y 99999 válidos.
- `FormatMinutes`: 0, 45, 60, 90, 120.
- `Report()` suma `TotalMinutes` y rellena `TotalTime`; el caso vacío da 0 / «—».

Acciones (`actions/tasks_test.go`):
- Crear con `work_minutes=90` persiste 90; crear sin el campo / con vacío persiste 0 y devuelve 303.
- Crear con valor negativo → 422, mensaje del campo y nada guardado.
- Editar: el formulario precarga los minutos; `PUT` actualiza título y minutos conservando `Completed` y `Position`; valor inválido → 422.
- La lista muestra `1 h 30 min` para una tarea con 90 minutos.
- Informe: con fixtures contiene «Tiempo total», «2 h 15 min» (90 + 45) y la columna «Tiempo».
- Actualizar `Test_TasksUpdate_PersistsTitleOnly` si su semántica ya no es «solo título».

Comprobaciones: `gofmt -l .` (sin salida), `go vet ./...`, `go test -p 1 ./...` (requiere PostgreSQL y migraciones de test aplicadas; ver `database.yml` y `.github/workflows/ci.yaml`), `go build ./...`.

## Riesgos y recuperación

- Bind de input vacío a `int`: si Buffalo devolviese error, usar un campo auxiliar string y convertirlo manualmente (cubierto por test de creación sin minutos).
- Base de datos de test sin la migración nueva: aplicar migraciones de test antes de `go test`.
- Fixtures modificadas pueden alterar aserciones existentes: revisar los tests que cuenten o comparen contenido.
- Recuperación: revertir el commit; la migración down elimina la columna (se pierden los tiempos registrados).

## Incertidumbres

- Unidad y formato de entrada (minutos enteros) inferidos; si se prefiere `h:mm` o decimales de horas cambia la entrada y el formateo, no la persistencia.
- Ninguna bloquea la implementación.
