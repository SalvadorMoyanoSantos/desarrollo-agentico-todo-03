# Feature: Reordenar tareas con drag and drop

## Encargo y criterios de aceptación

Encargo literal: «Añadir drag and drop a la lista de todas para poder ordenarlas».

Criterios (explícitos → E, inferidos → I):

1. (E) El usuario puede arrastrar una tarea de la lista y soltarla en otra posición.
2. (I) El nuevo orden se persiste: al recargar, `GET /` muestra las tareas en el orden elegido.
3. (I) El orden aplica a tareas pendientes y completadas por igual (la lista actual no las separa).
4. (I) Una tarea nueva se añade al final de la lista.
5. (I) Si el guardado falla, la lista vuelve visualmente al orden previo y se avisa al usuario (mismo patrón que el toggle).
6. (I) El reordenamiento es accesible por teclado (asa enfocable que mueve la tarea con ↑/↓), no solo con ratón.
7. (I) Editar, completar y eliminar siguen funcionando igual; las tareas existentes conservan su orden actual (`created_at, id`) tras la migración.
8. (I) Una petición de reordenación con IDs inválidos, repetidos o que no coinciden con las tareas existentes se rechaza sin modificar nada.

## Comportamiento actual y propuesta

Actual: `TasksIndex` y el ramo de error de `TasksCreate` (`actions/tasks.go`) ordenan con `created_at asc, id asc`. No existe columna de orden. La lista es un `<ul class="task-list">` de `<li class="task">` (`templates/tasks/index.plush.html`). `public/assets/app.js` intercepta el `submit` del toggle y hace `fetch` POST urlencoded con `_method=PUT` y `authenticity_token`, con actualización optimista y rollback.

Propuesta: columna `position` en `tasks`; ordenar por `position asc, created_at asc, id asc`; nuevo endpoint `PUT /tasks/reorder` que recibe la lista completa de IDs en el nuevo orden y reasigna `position` en una transacción (la de `popmw.Transaction` ya envuelve la petición); JS con HTML5 Drag and Drop nativo (sin dependencias) + movimiento por teclado, que envía el orden tras cada cambio con el mismo patrón de fetch/rollback que el toggle.

## Alcance

Incluido:
- Migración de `position` con backfill.
- Modelo: campo `Position`.
- `TasksCreate` asigna posición al final; ambos `Order(...)` actualizados (extraer a un helper para no duplicar).
- Acción y ruta `PUT /tasks/reorder`.
- Asa de arrastre en la plantilla, JS y CSS.
- Tests de acción y modelo; fixture con posiciones.

Excluido:
- Soporte de arrastre táctil (HTML5 DnD no funciona en la mayoría de navegadores móviles); se cubre con el teclado. Ver incertidumbres.
- Librerías externas (SortableJS), reordenación entre listas, filtros, paginación.
- Reordenar en bloque o multiusuario/concurrencia avanzada (última escritura gana).

## Diseño por capas

### Persistencia y migración
Nuevo par `migrations/20261019000000_add_position_to_tasks.up.fizz` / `.down.fizz` (patrón de `20260901000000_create_tasks`; posterior a las existentes):
- up: `add_column("tasks", "position", "integer", {"default": 0})` y después `raw` SQL de backfill:
  `UPDATE tasks SET position = s.rn FROM (SELECT id, row_number() OVER (ORDER BY created_at, id) AS rn FROM tasks) s WHERE tasks.id = s.id;`
  Si fizz no admite `raw` junto a `add_column` en el mismo archivo, usar un segundo par `.sql` (como `seed_tasks`) con un timestamp posterior. `add_index("tasks", "position")` es opcional (lista pequeña); omitir.
- down: `drop_column("tasks", "position")`.
- `NOT NULL`: poner `{"default": 0}` sin `null: true` para que sea NOT NULL con default.

### Modelo (`models/task.go`)
- Añadir `Position int \`json:"position" db:"position" form:"-"\``. Sin validación nueva.
- Helper `func OrderedTasks(tx *pop.Connection) (Tasks, error)` o constante `TaskOrder = "position asc, created_at asc, id asc"` usada por las acciones.

### Acción y ruta (`actions/tasks.go`, `actions/app.go`)
- `TasksCreate`: antes de `ValidateAndCreate`, calcular `max(position)+1` (`tx.RawQuery("SELECT COALESCE(MAX(position), 0) FROM tasks").First(&max)` o equivalente) y asignarlo a `task.Position`. Reemplazar los dos `tx.Order(...)` por el helper.
- `TasksReorder`: 
  - Bind a `struct{ IDs []string \`form:"ids" json:"ids"\` }` (el JS envía `ids` repetido en urlencoded).
  - Cargar todas las tareas (`All`), validar: mismo número de IDs que de tareas, sin duplicados, todos existen (parsear con `uuid.FromString`); si no, `c.Error(http.StatusUnprocessableEntity, ...)` sin escribir.
  - Actualizar `Position = i+1` solo en las que cambian (`tx.Update`) y responder `204 No Content` (`c.Render(http.StatusNoContent, nil)` o `c.Response().WriteHeader`); el cliente usa `fetch`, no navega. Sin flash.
- Ruta: `app.PUT("/tasks/reorder", TasksReorder)` registrada **antes** de `app.PUT("/tasks/{task_id}", ...)` para que no la capture `{task_id}`. CSRF: sigue en `authenticity_token` urlencoded como el toggle.

### Renderizado (`templates/tasks/index.plush.html`, `templates/application.plush.html`)
- `<ul class="task-list" data-task-list data-reorder-url="/tasks/reorder">`; token CSRF en `data-token="<%= authenticity_token %>"`.
- En cada `<li class="task" data-task-id="<%= t.ID %>" draggable="false">`, añadir antes del toggle una asa `<button type="button" class="task-handle" aria-label="Mover tarea: <%= t.Title %>" title="Arrastra para reordenar" data-task-handle>` con icono SVG de 6 puntos. El `draggable` se activa en el `<li>` solo al pulsar el asa (evita arrastrar al seleccionar texto).
- Ajustar `grid-template-columns` de `.task` para la nueva columna (p. ej. `28px 44px minmax(0,1fr) auto`).
- Subir la versión de cache: `app.css?v=4` y `app.js?v=4` (convención vigente de cache-busting; en el árbol de trabajo ya hay un cambio sin commitear a `?v=3` — ver riesgos).
- Con un solo elemento o lista vacía no es necesario ocultar el asa.

### JavaScript (`public/assets/app.js`)
- Añadir una función `saveOrder(list, previousOrder)` que arma `URLSearchParams` con `_method=PUT`, `authenticity_token` (de `data-token`) y un `ids` por cada `li` en orden, hace `fetch(list.dataset.reorderUrl, {method:"POST", credentials:"same-origin"})`; si `!response.ok` o falla la red, restaura el DOM al orden previo y `window.alert("No se pudo guardar el orden. Inténtalo de nuevo.")`.
- Drag: `dragstart` (sobre `li`, solo si se originó en el asa; guardar orden previo y añadir clase `task-dragging`), `dragover` sobre la lista (`preventDefault`, reubicar el `li` arrastrado antes/después del vecino según la posición vertical del cursor), `dragend` (quitar clase; si el orden cambió respecto al previo, llamar a `saveOrder`).
- Teclado: `keydown` en el asa con `ArrowUp`/`ArrowDown` mueve el `li` con `insertBefore`, devuelve el foco al asa y llama a `saveOrder`. Anunciar el cambio con una región `aria-live="polite"` visualmente oculta (`sr-only` ya existe) con texto «Tarea movida a la posición N de M».
- Añadir los listeners con delegación en `document`, igual que el handler de `submit` existente; no romper `renderTaskState`.

### CSS (`public/assets/app.css`)
- `.task-handle` (botón sin borde, cursor `grab`, color `var(--color-muted)`, foco visible coherente con `.icon-action`), `.task-dragging` (opacidad reducida), y que el asa sea visible siempre en táctil y atenuada en hover-capable (reutilizar el patrón de `@media (hover: hover)` de `.task-actions`). Respetar `prefers-reduced-motion` si ya se usa en el archivo.
- Ajustar la rejilla en el `@media (max-width: 760px)` donde se redefine `.task`.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `migrations/*_create_tasks.up.fizz` | Patrón de migración fizz existente. |
| `migrations/*_add_position_to_tasks.{up,down}.fizz` (nuevos) | Columna `position` + backfill. |
| `models/task.go` | Struct `Task`; añadir `Position` y helper de orden. |
| `actions/tasks.go` | `TasksIndex`, `TasksCreate` (orden/posición), nueva `TasksReorder`. |
| `actions/app.go` | Registro de rutas; `PUT /tasks/reorder` antes de `/tasks/{task_id}`. |
| `templates/tasks/index.plush.html` | Lista, asa, atributos `data-*`. |
| `templates/application.plush.html` | Versión de assets. |
| `public/assets/app.js` | Drag and drop, teclado, guardado y rollback. |
| `public/assets/app.css` | Estilos del asa y del estado de arrastre. |
| `fixtures/tasks.toml` | Añadir `position` al escenario «tres tareas» (1, 2, 3). |
| `actions/tasks_test.go`, `models/task_test.go` | Tests nuevos. |

## Implementación paso a paso

1. Migración `add_position_to_tasks` (up con backfill, down). Ejecutar las migraciones de desarrollo y de test (`buffalo pop migrate` / `-e test`, según el entorno del repo).
2. Añadir `Position` al modelo y el helper/constante de orden; actualizar `fixtures/tasks.toml`.
3. Cambiar `TasksIndex` y `TasksCreate` para usar el orden nuevo; asignar posición al final al crear.
4. Implementar `TasksReorder` y registrar la ruta antes de `/tasks/{task_id}`.
5. Plantilla: `data-*` en la lista y `<li>`, asa de arrastre, versiones de assets.
6. CSS del asa y del estado de arrastre (incluida la rejilla móvil).
7. JS: drag, teclado, región `aria-live`, `saveOrder` con rollback.
8. Escribir tests (siguiente sección) y ejecutar validación; comprobación manual en navegador (arrastrar, recargar, error de red simulado, teclado).

## Tests y validación

Requieren PostgreSQL accesible (`DB_HOST`, usuario/clave de `database.yml`) y la base `todo_test` con las migraciones aplicadas, incluida la nueva.

Tests de acción (`actions/tasks_test.go`, patrón `as.HTML(...).Put(url.Values{...})`):
- `Test_TasksIndex_OrdersByPosition`: crear tareas con posiciones no coincidentes con `created_at` y comprobar el orden de aparición en el HTML (`strings.Index`).
- `Test_TasksCreate_AppendsAtEnd`: tras crear con tareas existentes, la nueva tiene `position` mayor que todas.
- `Test_TasksReorder_Persists`: `Put` a `/tasks/reorder` con `ids` permutados → 204 y `Position` reasignadas; `GET /` refleja el orden.
- `Test_TasksReorder_InvalidID`, `_DuplicateIDs`, `_MissingTask`, `_UnknownID`: 422 y ninguna posición cambiada.
- `Test_TasksReorder_DoesNotMatchTaskID`: confirma que `/tasks/reorder` no se interpreta como `{task_id}` (404/500 por UUID inválido sería el fallo).
- `Test_TasksIndex_RendersHandle`: el HTML contiene `data-task-handle` y `data-reorder-url`.

Test de modelo (`models/task_test.go`): `Position` se persiste y el orden `position, created_at, id` es determinista con empates.

Validación manual (sin framework JS de tests en el repo): arrastrar con ratón, recargar y ver orden persistente; reordenar con teclado; cortar el servidor y comprobar rollback + alerta; comprobar que toggle/editar/eliminar siguen bien tras reordenar.

Comandos: `gofmt -l .`, `go vet ./...`, `go test -p 1 ./...`, `go build ./...`.

## Riesgos y recuperación

- **Árbol de trabajo con cambios previos** (`actions/tasks_test.go`, `public/assets/app.js`, `templates/application.plush.html`, `plans/bug-toggle-no-persiste-al-recargar.md` sin commitear, del plan de un bug distinto). Implementar sobre ellos o tras commitearlos; no revertirlos. Los números de versión de assets (`?v=3`) pueden entrar en conflicto: usar el siguiente número libre.
- **Orden de rutas**: si `/tasks/reorder` se registra después de `/tasks/{task_id}`, el `PUT` se enruta a `TasksUpdate`. Cubierto por test.
- **Migración sobre datos reales**: backfill determinista con `row_number()`. Recuperación: `down` elimina la columna; el orden vuelve a `created_at, id`.
- **Concurrencia**: dos pestañas reordenando → gana la última escritura; si se creó/borró una tarea entre medias, el reorder devuelve 422 y el cliente revierte con aviso (el usuario recarga).
- **Rollback optimista**: si `fetch` falla tras mover el DOM, el orden previo se reconstruye reinsertando los `li` en el orden guardado.
- **Táctil**: sin arrastre en móviles; mitigado solo por teclado. Si se necesita, añadir Pointer Events en una iteración aparte.
- Recuperación general: el cambio es aditivo; revertir el commit y hacer `migrate down` de la última migración.

## Incertidumbres pendientes

- ¿Se requiere arrastre táctil en móvil? (Este plan no lo incluye.)
- ¿Las tareas completadas deben seguir mezcladas en el orden manual (supuesto actual) o agruparse al final?
- ¿Las nuevas tareas van al final (supuesto actual) o al principio?
