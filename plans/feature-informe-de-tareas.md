# Feature: página de informe de las tareas creadas

## Encargo y criterios de aceptación

Encargo: «Crea una página con un informe de las tareas creadas».

Criterios (los marcados *inferido* son interpretación del encargo):

1. `GET /report` responde 200 con una página "Informe de tareas" dentro del layout existente.
2. Muestra el total de tareas, las completadas, las pendientes y el porcentaje completado (*inferido*; 0 % si no hay tareas, sin dividir por cero).
3. Muestra cuántas tareas se crearon por día (fecha de `created_at`), de más reciente a más antigua (*inferido*).
4. Muestra el listado de tareas (título, estado, fecha de creación) en el orden `models.TaskOrder`.
5. Con la base vacía muestra un estado vacío ("No hay tareas todavía") con ceros en los contadores.
6. La barra lateral incluye un enlace "Informe" con `aria-current` en la página activa; el enlace "Tareas" deja de marcarse como activo en el informe.
7. Solo lectura: no modifica datos.

## Comportamiento actual y propuesta

Actual: solo existe la lista en `/` (`TasksIndex`), con `tasks/index.plush.html`. El layout `templates/application.plush.html` tiene un único enlace de navegación con `aria-current="true"` fijo. No hay estadísticas.

Propuesta: nueva acción `TasksReport` en `GET /report`, que carga las tareas con una única consulta, calcula las métricas en Go (función pura en el modelo, testeable sin HTTP) y renderiza `tasks/report.plush.html`.

## Alcance

Incluido: ruta, acción, función de resumen en el modelo, template, enlace en la barra lateral, estilos mínimos, tests.

Excluido: migraciones (no hay cambios de esquema), exportación CSV/PDF, filtros por fechas, gráficos con JS, autenticación, cambios al flujo de reordenación.

## Diseño por capas

- **Persistencia**: sin cambios. Se lee `tasks` con `tx.Order(models.TaskOrder).All(&tasks)`, igual que `TasksIndex`.
- **Modelo** (`models/task.go`): añadir
  - `type TaskReport struct { Total, Completed, Pending, Percent int; ByDay []DayCount }`
  - `type DayCount struct { Day time.Time; Count int }`
  - `func (ts Tasks) Report() TaskReport`: cuenta completadas/pendientes; `Percent = Completed*100/Total` (0 si Total==0, división entera); agrupa por `CreatedAt` truncado a día (UTC, `Format("2006-01-02")` como clave) y ordena de más reciente a más antiguo.
- **Acción y ruta**: `TasksReport` en `actions/tasks.go`; registrar `app.GET("/report", TasksReport)` en `actions/app.go` junto a las demás rutas. Establece `tasks` y `report` en el contexto y renderiza con `r.HTML("tasks/report.plush.html")`.
- **Render**: `templates/tasks/report.plush.html` con `page-header`, bloque de métricas (total, completadas, pendientes, porcentaje), tabla "Tareas creadas por día" y tabla/lista de tareas. Reutilizar la clase `empty-state` y el patrón de `index.plush.html`. Formato de fechas con `Format("02/01/2006")`. Si el template Plush no admite llamar a un método con args, precalcular las cadenas en la acción.
- **Layout**: en `application.plush.html` añadir segundo `<a class="nav-link">` "Informe" a `/report`. Para `aria-current` usar `current_path` (disponible en Buffalo) comparado con `"/report"`; mantener el icono SVG inline con el estilo existente. Subir el cache-buster `?v=5` → `?v=6` de CSS si se toca CSS.
- **CSS** (`public/assets/app.css`): clases `.report-metrics` (grid de tarjetas) y `.report-table`, reutilizando las variables de color existentes y con comportamiento responsive como el resto del archivo.

## Archivos relevantes

- `actions/app.go` — registro de rutas.
- `actions/tasks.go` — acciones de tareas; aquí va `TasksReport`.
- `actions/tasks_test.go` — tests de acción (`ActionSuite`, fixture "tres tareas").
- `models/task.go` — modelo `Task`/`Tasks`; aquí va `Report()`.
- `models/task_test.go` — tests del modelo.
- `templates/application.plush.html` — layout y navegación.
- `templates/tasks/index.plush.html` — patrón de referencia.
- `templates/tasks/report.plush.html` — nuevo.
- `public/assets/app.css` — estilos.
- `fixtures/tasks.toml` — escenario "tres tareas": 2 pendientes, 1 completada.

## Implementación paso a paso

1. Modelo: añadir `TaskReport`, `DayCount` y `Tasks.Report()` en `models/task.go`.
2. Acción: añadir `TasksReport` y la ruta `GET /report`.
3. Template `report.plush.html` con métricas, tabla por día, listado y estado vacío.
4. Layout: enlace "Informe" y `aria-current` según `current_path`.
5. CSS de las métricas y tablas; actualizar `?v=`.
6. Tests (ver abajo) y ejecutar validaciones.

## Tests y validación

Modelo (`models/task_test.go`, sin fixtures, con `Tasks` construidas a mano):
- vacío → todo a 0 y `Percent == 0`;
- 3 tareas con 1 completada → Total 3, Completed 1, Pending 2, Percent 33;
- tareas en dos días distintos → `ByDay` con dos entradas, la más reciente primero.

Acción (`actions/tasks_test.go`):
- `Test_TasksReport_EmptyState`: 200 y contiene "No hay tareas todavía".
- `Test_TasksReport_Metrics`: con `LoadFixture("tres tareas")`, 200, contiene "Informe de tareas", los tres títulos y los totales (3, 1, 2, 33 %).
- `Test_TasksReport_NavLink`: el layout contiene `href="/report"`.

Comandos: `gofmt -l .` (sin salida), `go vet ./...`, `go test -p 1 ./...` (requiere PostgreSQL y migraciones de test según `database.yml` y CI), `go build ./...`.

## Riesgos y recuperación

- Zona horaria: el día de agrupación se calcula en UTC; un usuario en otra zona puede ver tareas cercanas a medianoche en el día contiguo. Aceptado para esta versión.
- Con muchas tareas el listado completo es largo; aceptable por el tamaño actual de la app (sin paginación en `/`).
- `current_path` en Plush: si no está disponible en el contexto, usar una variable `active` fijada en cada acción.
- Recuperación: el cambio es aditivo y sin migraciones; se revierte eliminando la ruta, la acción, el template, el enlace y los estilos.

## Incertidumbres

Ninguna bloqueante. El contenido exacto del "informe" es una inferencia (métricas, creación por día y listado); ajustable sin cambios de arquitectura.
