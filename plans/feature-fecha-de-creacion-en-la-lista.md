# Feature: Fecha de creación de la tarea en el listado

## Encargo y criterios de aceptación

Encargo literal: «Añade la fecha de creación de la tarea».

Requisitos explícitos:
- Cada tarea debe mostrar su fecha de creación.

Inferencias (decisiones de este plan):
- La fecha ya se guarda (`created_at`); no hace falta migración ni cambios en el modelo.
- El listado principal (`/`, acción `TasksIndex`) no muestra la fecha; el informe (`/report`) ya la muestra en la columna «Creada». El encargo se entiende como añadirla al listado principal, en cada `<li>` de tarea.
- Formato: `dd/mm/aaaa`, el mismo que usa el informe (`Format("02/01/2006")`). Se muestra en UTC, igual que la agrupación por día de `Tasks.Report()`.
- Se expone en el HTML como `<time datetime="...">` para que sea semántico.
- No se añade un método `CreatedDate()` en el modelo: el informe ya formatea la fecha inline en su template y el cambio se mantiene mínimo.

Criterios de aceptación:
1. Cada tarea de la lista muestra su fecha de creación en formato `dd/mm/aaaa`.
2. La fecha mostrada coincide con `created_at` de la tarea (en UTC), y el `datetime` del `<time>` es su ISO.
3. Las tareas existentes muestran su fecha real sin migración.
4. La estructura de cada tarea (handle, toggle, título, acciones) mantiene su disposición en escritorio y móvil; las acciones no cambian de fila.
5. El informe no se modifica.

## Estado actual del árbol de trabajo

Antes de este plan ya existe una implementación parcial **sin commit** en la rama `feat/informe-de-tareas`:

- `templates/tasks/index.plush.html`: línea 37, `<time class="task-date" datetime="..." title="Fecha de creación">...</time>` como **hermano directo** de `<span class="task-title">`.
- `public/assets/app.css`: bloque `.task-date` (líneas ~312-318) con `align-self: center`, `color: var(--color-muted)`, `font-size: 0.8rem`, `white-space: nowrap`.
- `actions/tasks_test.go`: `Test_TasksIndex_ShowsCreatedDate` (línea ~474).

Esa implementación tiene un problema de layout (ver siguiente sección) y debe corregirse, no duplicarse. `/implement` debe partir de ese estado.

## Comportamiento actual y propuesta

Actual: cada `<li class="task">` es un grid de 4 columnas (`grid-template-columns: 28px 44px minmax(0, 1fr) auto` en escritorio y `32px 36px minmax(0, 1fr) auto` en móvil, `app.css` ~línea 122 y ~495). Sus hijos son: botón handle, formulario toggle, `span.task-title`, `span.task-status` (oculto con clip, posición absoluta), `div.task-actions`. No muestra la fecha.

Problema de la implementación parcial: al ser un hijo directo, el `<time>` ocupa la cuarta columna `auto` del grid y `div.task-actions` pasa a la segunda fila, rompiendo el layout de cada tarea. El plan anterior decía «no cambiar el layout», pero el grid sí se ve afectado.

Propuesta: mover el `<time>` **dentro** de `span.task-title`, después del tiempo de trabajo, para que sea parte de la columna de título, y mostrarlo como segunda línea pequeña bajo el título. Así el grid sigue teniendo los mismos hijos directos.

```html
<span class="task-title"><span class="task-title-text"><%= t.Title %></span><%= if (t.WorkMinutes > 0) { %> <small class="task-time" title="Tiempo de trabajo"><%= t.WorkTime() %></small><% } %><time class="task-date" datetime="<%= t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z") %>" title="Fecha de creación"><%= t.CreatedAt.UTC().Format("02/01/2006") %></time></span>
```

Nota: el `<time>` va dentro de `.task-title`, que tiene `padding: 0.65rem 0`. La clase `.task-date` debe ser `display: block` para quedar bajo el título.

## Alcance

Incluido: corrección de la posición del `<time>` en `templates/tasks/index.plush.html`, ajuste de `.task-date` en `public/assets/app.css`, test de acción (ya existe, se verifica y se mantiene).

Excluido: hora de creación, fecha de última modificación, ordenar o filtrar por fecha, cambios en el informe, cambios en modelo, migraciones o base de datos, JavaScript.

## Diseño por capas

### Persistencia
Sin cambios. `created_at` ya existe.

### Modelo (`models/task.go`)
Sin cambios.

### Acciones (`actions/tasks.go`)
Sin cambios. `TasksIndex` ya pasa `tasks` completos.

### Renderizado (`templates/tasks/index.plush.html`)
Mover el `<time>` de la línea 37 al interior de `span.task-title` según la propuesta. El texto usa el mismo formato que el informe.

### CSS (`public/assets/app.css`)
Sustituir el bloque `.task-date` actual por:

```css
.task-date {
  display: block;
  margin-top: 0.15rem;
  color: var(--color-muted);
  font-size: 0.8rem;
  line-height: 1.3;
}
```

Eliminar `align-self` y `white-space: nowrap` (no aplican en bloque y el título ya tiene `overflow-wrap: anywhere`). No tocar las reglas de `.task` ni sus media queries.

## Archivos relevantes

- `templates/tasks/index.plush.html` — línea 37 (`<time>`), mover dentro de `span.task-title`.
- `public/assets/app.css` — bloque `.task-date` (~312) y reglas de `.task` (~122, ~495) como referencia.
- `actions/tasks_test.go` — `Test_TasksIndex_ShowsCreatedDate` (~474), ya existente; mantener o ampliar.
- `actions/tasks_test.go` — `Test_TasksIndex_TitleTextExcludesTime` (~486) como regresión del texto del título.

## Implementación paso a paso

1. En `templates/tasks/index.plush.html`, quitar el `<time>` de su posición actual (línea 37) y colocarlo dentro de `span.task-title`, después del `<small class="task-time">`.
2. En `public/assets/app.css`, reemplazar `.task-date` según la sección CSS.
3. Revisar que `.task` mantiene 4 columnas: `div.task-actions` debe seguir en la misma fila que el título (comprobar en escritorio y a 375 px con la app en marcha, o por lectura del grid).
4. Mantener `Test_TasksIndex_ShowsCreatedDate`. Si el test de título comprueba el texto exacto de `.task-title`, ajustarlo solo si el nuevo `<time>` lo altera, sin debilitar la aserción.

## Tests y validación

- Acción: `/` contiene `<time class="task-date"`, `datetime="2026-10-22T12:00:00Z"` y `>22/10/2026</time>` para una tarea con `CreatedAt` fijado (ya cubierto por `Test_TasksIndex_ShowsCreatedDate`).
- Regresión: los tests existentes de `TasksIndex` (orden, tiempo de trabajo, estado, texto del título) siguen pasando.
- Visual (manual o con `/run`): la fila de cada tarea mantiene handle, toggle, título con fecha debajo y acciones alineadas, en escritorio y en móvil.

Comprobaciones: `gofmt -l .` (sin salida), `go vet ./...`, `go test -p 1 ./...` (requiere PostgreSQL y migraciones de test aplicadas; ver `database.yml` y `.github/workflows/ci.yaml`), `go build ./...`.

## Riesgos y recuperación

- Zona horaria: se usa UTC para coincidir con el informe; cerca de medianoche la fecha puede diferir de la local. Mostrar hora local requiere decidir la zona.
- Layout: el riesgo principal es el de la implementación parcial actual (hijo directo del grid). Corregido por la propuesta; verificar visualmente.
- Recuperación: revertir solo los cambios de esta feature (`git checkout -- templates/tasks/index.plush.html public/assets/app.css` y el test si se quiere eliminar). No hay cambios de datos ni de esquema. Ojo: `actions/tasks_test.go`, `public/assets/app.css` y `templates/tasks/index.plush.html` tienen también cambios ajenos a esta feature en el árbol de trabajo; no revertir archivos enteros sin revisarlos.

## Incertidumbres

- Interpretación: se asume que la fecha debe verse en el listado principal, porque el informe ya la muestra. Si el encargo se refería a otra vista, cambia solo el template.
- Formato `dd/mm/aaaa` y UTC inferidos del informe existente.
- Posición exacta (segunda línea bajo el título) es una decisión de diseño; alternativa: en línea junto al tiempo de trabajo, con `display: inline`.
