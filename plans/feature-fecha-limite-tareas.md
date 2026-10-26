# Feature: fecha límite en las tareas

Incidencia: https://github.com/SalvadorMoyanoSantos/desarrollo-agentico-todo-03/issues/4 (título: "Añade la fecha límite a las tareas"; sin descripción).

Base del plan: `main` en `ad9b3de` (`feat/informe-de-tareas` tiene 4 commits que `main` no tiene). La rama de trabajo propuesta es `feat/fecha-limite-tareas` creada desde `main`, no desde `feat/informe-de-tareas`, para que la pull request contenga solo este cambio.

## Encargo y criterios de aceptación

Encargo explícito: añadir una fecha límite a las tareas.

Criterios de aceptación:

1. Cada tarea puede tener una fecha límite opcional (día, sin hora).
2. Al crear una tarea desde el formulario de la lista se puede indicar la fecha límite; si se deja vacía, la tarea no tiene fecha.
3. Al editar una tarea se puede cambiar o borrar su fecha límite, además del título.
4. La lista muestra la fecha límite de cada tarea que la tenga.
5. Una fecha con formato inválido se rechaza con 422 y un mensaje en español, sin crear ni modificar la tarea.
6. Las tareas existentes quedan sin fecha límite tras la migración; no se pierde ningún dato.

Inferencias (no explícitas en la incidencia; elegidas por defecto):

- La fecha es solo día (`date`), sin zona horaria ni hora.
- Se admiten fechas pasadas; no se marca nada como "vencida" en esta iteración.
- La fecha se edita con `<input type="date">` nativo, sin JavaScript nuevo.
- La fecha no afecta al orden de la lista (sigue ordenada por `position`).

## Comportamiento actual y propuesta

Actual (en `main`):

- `models.Task` tiene `ID`, `Title`, `Completed`, `Position`, `CreatedAt`, `UpdatedAt` (`models/task.go`).
- `TasksCreate` (`actions/tasks.go`) solo enlaza `Title`; el formulario de `templates/tasks/index.plush.html` tiene solo el campo de título.
- `TasksUpdate` (`actions/tasks.go`) actualiza únicamente el título; los tests `Test_TasksUpdate_PersistsTitleOnly` exigen que el resto de campos no cambie cuando el formulario solo envía `title`.
- `templates/tasks/edit.plush.html` solo tiene el campo de título.

Propuesta:

- Nueva columna `due_date date NULL` en `tasks`.
- `Task.DueDate *time.Time` (nulo = sin fecha).
- Creación y edición aceptan `due_date` en formato `YYYY-MM-DD`. Cadena vacía = sin fecha. Campo ausente en `TasksUpdate` = no tocar la fecha (preserva el contrato de `Test_TasksUpdate_PersistsTitleOnly`).
- Las listas muestran "Vence el DD/MM/AAAA" junto al título cuando hay fecha.

## Alcance

Incluido:

- Migración up/down de la columna.
- Campo en el modelo y validación de formato de fecha.
- Campo de fecha en el formulario de creación (index) y de edición (edit).
- Procesado de `due_date` en `TasksCreate` y `TasksUpdate`.
- Visualización de la fecha en la lista.
- Tests de modelo y de acciones.

Excluido:

- Marcar tareas vencidas, ordenar por fecha, filtros, recordatorios, notificaciones.
- Hora o zona horaria.
- Cambios en el informe (`/report`), que no existe en `main`.
- Cambios en `config/adw.yml`, `.claude/`, `migrations/schema.sql` y `plans/bug-toggle-no-persiste-al-recargar.md` del árbol de trabajo actual (no pertenecen a esta incidencia).

## Diseño por capas

- **Persistencia**: migración `migrations/20261026000000_add_due_date_to_tasks.{up,down}.sql`. Up: `ALTER TABLE tasks ADD COLUMN due_date date NULL;`. Down: `ALTER TABLE tasks DROP COLUMN due_date;`. Sigue el estilo de `20261019000000_add_position_to_tasks` (SQL plano).
- **Modelo** (`models/task.go`): campo `DueDate *time.Time` con tags `db:"due_date" form:"-"` (la fecha se parsea en la acción, no por el binder). Constante o función de formato `DueDateLayout = "2006-01-02"`. Validación en `Validate`: si `DueDate` no es nil, nada más que comprobar (el formato ya se valida en la acción). Un helper `FormatDueDate` opcional para la plantilla, o formatear en la plantilla con `t.DueDate.Format("02/01/2006")`.
- **Acciones** (`actions/tasks.go`):
  - `TasksIndex`/`TasksCreate` (errores) y `TasksEdit` no cambian salvo que el formulario de creación necesite re-renderizar con el valor enviado.
  - `TasksCreate`: leer `due_date` con un struct de entrada `struct { Title string; DueDate string }` o leerlo de `c.Request().FormValue`. Si no vacío, `time.Parse(models.DueDateLayout, s)`; si falla, añadir error `due_date` ("La fecha límite no es válida.") y renderizar 422 como hoy.
  - `TasksUpdate`: usar `DueDate *string` en el struct de entrada (nil = ausente → no tocar; "" → borrar; valor → parsear). Mismo manejo de error y 422.
- **Plantillas**:
  - `templates/tasks/index.plush.html`: input `type="date" name="due_date"` con `<label>` sr-only "Fecha límite (opcional)" en el formulario de nueva tarea; mostrar `Vence el ...` en `<li>` cuando `t.DueDate != nil`.
  - `templates/tasks/edit.plush.html`: input `type="date" name="due_date"` con `value` prefijado en formato `YYYY-MM-DD` y un label visible "Fecha límite".
- **CSS** (`public/`): localizar la hoja de estilos que define `.new-task-row` y `.edit-task`, y añadir solo lo mínimo para que el input de fecha quepa en la fila. Sin tocar otras reglas.

## Archivos relevantes

- `migrations/` (nuevo par up/down): columna `due_date`. `migrations/embed.go` embebe el directorio; no requiere cambios.
- `models/task.go`: campo y tipo de la fecha.
- `models/task_test.go`: tests de validación del modelo (si el modelo añade validación).
- `actions/tasks.go`: creación, edición y parseo de la fecha.
- `actions/tasks_test.go`: tests de acción.
- `templates/tasks/index.plush.html`: campo y visualización.
- `templates/tasks/edit.plush.html`: campo de edición.
- `public/` (hoja de estilos, a localizar): ajuste mínimo de layout.
- `fixtures/tasks.toml`: sin cambios (la columna es nullable).

## Implementación paso a paso

1. Crear rama `feat/fecha-limite-tareas` desde `main`, sin tocar el árbol de trabajo de `feat/informe-de-tareas` (usar `git worktree` o una copia limpia de `main`).
2. Añadir la migración up/down.
3. Añadir `DueDate` y la constante de formato al modelo.
4. En `TasksCreate`, parsear `due_date` opcional; error de formato → 422 con mensaje.
5. En `TasksUpdate`, tratar `due_date` ausente/vacío/valor; mantener el resto del comportamiento.
6. Añadir el campo de fecha al formulario de creación y a la edición; mostrar la fecha en la lista.
7. Ajustar CSS mínimo.
8. Añadir tests (ver siguiente sección) y ejecutar las validaciones.

## Tests y validación

Tests de acción nuevos en `actions/tasks_test.go` (siguiendo el estilo existente con `as.HTML`, `as.DB`, `createPositioned`):

- `Test_TasksCreate_WithDueDate`: crear con `due_date=2026-11-15` → 303 y `DueDate` persistido en esa fecha.
- `Test_TasksCreate_EmptyDueDate`: `due_date=""` → 303 y `DueDate == nil`.
- `Test_TasksCreate_InvalidDueDate`: `due_date=31/12/2026` → 422, mensaje "La fecha límite no es válida.", y 0 tareas creadas.
- `Test_TasksUpdate_SetsDueDate`: PUT con `title` y `due_date` → fecha actualizada.
- `Test_TasksUpdate_ClearsDueDate`: PUT con `due_date=""` → `DueDate == nil`.
- `Test_TasksUpdate_WithoutDueDateKeepsIt`: PUT solo con `title` → fecha intacta (el contrato existente).
- `Test_TasksUpdate_InvalidDueDate`: 422 y fecha original intacta.
- `Test_TasksIndex_ShowsDueDate`: la lista contiene "Vence el 15/11/2026".
- `Test_TasksEdit_PrefillsDueDate`: el input contiene `value="2026-11-15"`.

Validación (requiere PostgreSQL con migraciones de test, ver `database.yml` y `.github/workflows/ci.yaml`):

1. `gofmt -l .` (sin salida)
2. `go vet ./...`
3. `go test -p 1 ./...`
4. `go build ./...`

Comprobación manual opcional: `/env:start` y crear/editar una tarea con fecha en el navegador.

## Riesgos y recuperación

- **Ausente vs vacío en `TasksUpdate`**: si el binder de Buffalo no distingue `nil` de `""` para `*string`, cambiar a leer `c.Request().Form` con `_, ok := ...`. Cubierto por `Test_TasksUpdate_WithoutDueDateKeepsIt`.
- **Zona horaria**: al parsear `YYYY-MM-DD` con `time.Parse` se obtiene UTC medianoche; al guardar como `date` no hay desplazamiento. Verificar en el test de persistencia.
- **Conflicto con `feat/informe-de-tareas`**: ese branch también toca `actions/tasks.go`, `models/task.go` y `templates/tasks/index.plush.html`; al fusionar ambas ramas habrá que resolver conflictos. Esta rama parte de `main`, así que es la ruta limpia para la PR.
- **Recuperación**: la migración `down` elimina la columna; si la fecha ya se usa en producción, el down pierde datos (aceptable en esta fase, pero conviene indicarlo).
