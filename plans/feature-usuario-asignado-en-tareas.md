# Feature: Usuario asignado en las tareas

## Encargo y criterios de aceptación

Encargo literal: «Añade el usuario al que se asigna la tarea; será un campo de texto».

Requisitos explícitos:
- Cada tarea tiene un campo para el usuario al que se asigna.
- Ese campo es de texto libre.
- Límite de 50 caracteres.
- El campo es opcional.

Inferencias (decisiones de este plan):
- Opcional: una tarea sin asignar guarda cadena vacía (`''`), igual que `work_minutes` usa 0 para «sin tiempo». No se valida que el texto corresponda a un usuario existente, porque no hay modelo de usuarios en la aplicación.
- Nombre de columna y campo: `assignee` (`Assignee` en Go), en inglés como el resto de columnas. La etiqueta visible es «Asignado a».
- Longitud máxima: 50 caracteres (`AssigneeMaxLength`), contados en runas como el título. Se recorta con `strings.TrimSpace` antes de validar.
- Se introduce en el formulario de creación y en el de edición. Se muestra en la lista (solo si tiene valor) y en el informe (columna «Asignado a»).
- No se añade filtro, ni búsqueda, ni resumen por usuario en el informe.

Criterios de aceptación:
1. Crear una tarea con un nombre en «Asignado a» lo persiste recortado de espacios; sin valor se guarda cadena vacía y la creación sigue funcionando.
2. Editar una tarea permite cambiar título, minutos y asignado; el estado completada y la posición se conservan.
3. Un asignado de más de 50 caracteres devuelve 422 con mensaje en español y no guarda nada.
4. La lista muestra el asignado de las tareas que lo tienen y no muestra nada para las demás.
5. El informe muestra una columna «Asignado a» en el listado de tareas (vacío como «—»).
6. Las tareas existentes quedan con cadena vacía tras la migración.

## Comportamiento actual y propuesta

Actual: `Task` (`models/task.go`) tiene `title`, `completed`, `position`, `work_minutes` y timestamps. `TasksCreate` enlaza el formulario al modelo con `c.Bind(task)` (los campos con `form:"-"` no se enlazan). `TasksUpdate` enlaza una struct anónima con `title` y `work_minutes`. La lista (`templates/tasks/index.plush.html`) muestra el título con el tiempo opcional; el formulario de creación tiene título y minutos; el de edición (`templates/tasks/edit.plush.html`) tiene título y minutos.

Propuesta: añadir la columna `assignee` con su migración, el campo `Assignee` en el modelo con validación de longitud, aceptarlo en crear y editar, mostrarlo en lista y en el informe. Replicar exactamente el patrón de `work_minutes`.

## Alcance

Incluido: migración, modelo, validación, acciones crear/editar, templates index/edit/report, CSS mínimo si hace falta, fixtures y tests.

Excluido: relación con una tabla de usuarios, autocompletado o lista de usuarios, validación de existencia, varios asignados por tarea, filtros o agrupación por asignado, API JSON nueva, cambio de asignado desde la lista sin pasar por el formulario.

## Diseño por capas

### Persistencia
Nueva migración SQL con el mismo patrón que `20261022000000_add_work_minutes_to_tasks`. Elegir una marca de tiempo posterior a la última existente (`20261022000000`), por ejemplo `20261029000000`:
- `migrations/20261029000000_add_assignee_to_tasks.up.sql`: `ALTER TABLE tasks ADD COLUMN assignee varchar(50) NOT NULL DEFAULT '';`
- `migrations/20261029000000_add_assignee_to_tasks.down.sql`: `ALTER TABLE tasks DROP COLUMN assignee;`

No tocar `migrations/schema.sql` ni las migraciones existentes.

### Modelo (`models/task.go`)
- Constante `AssigneeMaxLength = 50`.
- Campo en `Task`, justo tras `Title`, con el mismo estilo de tags que `Title`: `Assignee string \`json:"assignee" db:"assignee" form:"assignee"\``.
- En `Validate`: `assignee := strings.TrimSpace(t.Assignee)`; si `len([]rune(assignee)) > AssigneeMaxLength`, `verrs.Add("assignee", fmt.Sprintf("El nombre del usuario asignado no puede superar los %d caracteres.", AssigneeMaxLength))`. No es obligatorio, así que no hay mensaje de vacío.
- Método `(t Task) AssigneeName() string` que devuelve `"—"` si está vacío y el valor si no. Así la plantilla del informe no tiene lógica (igual que `WorkTime()`).

### Acciones (`actions/tasks.go`)
- `TasksCreate`: el campo se enlaza con `c.Bind(task)` gracias a `form:"assignee"`. Añadir `task.Assignee = strings.TrimSpace(task.Assignee)` junto a la línea de `Title`, para que la validación y el guardado usen el valor recortado.
- `TasksUpdate`: ampliar la struct de entrada con `Assignee string \`form:"assignee" json:"assignee"\``, asignar `task.Assignee = strings.TrimSpace(input.Assignee)` y actualizar el comentario del método para mencionar el asignado («actualiza el título, el tiempo de trabajo y el asignado, conservando…»).
- `TasksReport`: sin cambios.
- Sin rutas nuevas.

### Renderizado
- `templates/tasks/index.plush.html`:
  - En el formulario de creación, añadir `<input type="text" name="assignee" maxlength="50" placeholder="Asignado a" aria-label="Asignado a" value="<%= task.Assignee %>" ...>` dentro de `.new-task-row`, entre el título y los minutos. Marcar `aria-invalid` y `aria-describedby="title-error"` si `errors.Get("assignee")` tiene mensajes, como ya se hace con `work_minutes`, y añadir sus mensajes al bloque `#title-error`.
  - En cada `<li>`, dentro de `.task-title`, tras el tiempo opcional: `<%= if (t.Assignee != "") { %> <small class="task-assignee" title="Asignado a"><%= t.Assignee %></small><% } %>`.
- `templates/tasks/edit.plush.html`: añadir `<label for="task-assignee" class="field-label-spaced">Asignado a</label>` y el input de texto con `maxlength="50"`, `value="<%= task.Assignee %>"`, y `aria-invalid`/`aria-describedby="assignee-error"` con su bloque `<p id="assignee-error" class="field-error" role="alert">`. Colocarlo entre el de minutos y las acciones, o entre título y minutos; mantener el orden título → asignado → minutos para que se lea como en la lista. Actualizar el texto del hint del título si menciona qué no cambia, para que siga siendo correcto.
- `templates/tasks/report.plush.html`: en el listado, añadir `<th scope="col">Asignado a</th>` y `<td><%= t.AssigneeName() %></td>` entre «Estado» y «Tiempo». No hace falta métrica nueva.
- `templates/tasks/index.plush.html`: el `value` del input de asignado en el formulario de creación debe reflejar `task.Assignee` tras un error de validación, igual que el título.

### CSS (`public/assets/app.css`)
- `.task-assignee`: mismo tratamiento que `.task-time` (texto secundario pequeño). Si `.task-time` ya tiene estilo propio, reutilizar el mismo selector compartido en lugar de duplicarlo.
- `.new-task-row`: el input de asignado ocupa un ancho intermedio (p. ej. ~10 rem), y en móvil debe envolver sin romper el layout. Revisar el media query existente del formulario de creación.
- No hace falta JavaScript.

## Archivos relevantes

- `migrations/20261029000000_add_assignee_to_tasks.{up,down}.sql` — columna nueva (crear).
- `models/task.go` — campo, constante, validación, `AssigneeName()`.
- `actions/tasks.go` — `TasksCreate` (recorte) y `TasksUpdate` (enlace y recorte).
- `templates/tasks/index.plush.html` — entrada en creación y visualización en lista.
- `templates/tasks/edit.plush.html` — entrada en edición.
- `templates/tasks/report.plush.html` — columna en el listado.
- `public/assets/app.css` — estilos de `.task-assignee` y del input nuevo.
- `fixtures/tasks.toml` — añadir `assignee` a una fila (p. ej. «Ana») para probar la lista y el informe.
- `models/task_test.go`, `actions/tasks_test.go` — tests siguiendo el patrón `Test_…` existente.

## Implementación paso a paso

1. Crear la migración `up` y `down` con los nombres indicados.
2. En `models/task.go`: añadir `AssigneeMaxLength`, el campo `Assignee`, la validación y `AssigneeName()`.
3. En `actions/tasks.go`: recortar `Assignee` en `TasksCreate` y ampliar la struct de `TasksUpdate`.
4. Actualizar `templates/tasks/index.plush.html`: input en el formulario de creación, visualización en la lista y mensajes de error.
5. Actualizar `templates/tasks/edit.plush.html` con el campo y su error.
6. Actualizar `templates/tasks/report.plush.html` con la columna.
7. Ajustar `public/assets/app.css` solo donde haga falta.
8. Añadir `assignee` a una fila de `fixtures/tasks.toml`.
9. Añadir tests (ver siguiente sección).

## Tests y validación

Tests a añadir:
- `models/task_test.go`: asignado de 50 caracteres válido; de 51 caracteres inválido con el mensaje de `assignee`; cadena vacía válida; `AssigneeName()` devuelve `"—"` para vacío y el nombre si no.
- `actions/tasks_test.go`:
  - `Test_TasksCreate_WithAssignee`: crear con `Assignee: "  Ana  "` guarda `"Ana"`.
  - `Test_TasksCreate_AssigneeTooLong_Invalid`: 422, no guarda nada, mensaje en español.
  - `Test_TasksUpdate_Assignee`: cambia el asignado y conserva `completed` y `position`.
  - `Test_TasksIndex_ShowsAssignee`: con el fixture, la lista contiene el asignado de la fila correspondiente.
  - `Test_TasksReport_ShowsAssignee`: el informe contiene la columna «Asignado a» y el nombre.

Comprobaciones:
- `gofmt -l .` (sin salida).
- `go vet ./...`.
- `go test -p 1 ./...`. Los tests de `actions` y `models` necesitan PostgreSQL en marcha y las migraciones de test aplicadas, incluida la nueva; si no están disponibles, la validación no se puede completar y hay que indicarlo.
- `go build ./...`.

Verificación manual opcional: arrancar la app y comprobar en `/` y `/report` que el asignado aparece solo cuando tiene valor.

## Riesgos y recuperación

- **Migración sobre datos existentes**: `DEFAULT ''` con `NOT NULL` rellena las filas actuales sin romper nada. Si la base ya tiene datos de prueba con otra convención, no afecta.
- **Binding de Buffalo**: si el campo no llega en el formulario, `Assignee` queda vacío y se guarda `''`; esto es el comportamiento esperado, no un error.
- **Desbordamiento de columna**: la validación de 50 runas corta antes de la base de datos. `varchar(50)` cuenta caracteres, no bytes, en PostgreSQL, así que coincide con el límite de la validación.
- **Layout móvil**: el formulario de creación ya tiene varios controles en una fila. Riesgo de que el nuevo input rompa el layout; revisarlo en ancho de teléfono.
- **Recuperación**: revertir con la migración `down` (`ALTER TABLE tasks DROP COLUMN assignee;`) y revertir los cambios de código. No hay datos irrecuperables: la columna es nueva y opcional.

## Incertidumbres pendientes

- Ninguna pendiente: el límite de 50 caracteres y la opcionalidad vienen del encargo. Cambiar el límite exige tocar la constante `AssigneeMaxLength`, el `varchar(50)` de la migración y los dos `maxlength="50"` de los templates.
