# Bug: el tiempo de trabajo no se ve con estilo y el botón de completar lee mal la tarea

## Encargo y resultado esperado

Descripción recibida: «Tengo un bug en la aplicación. Encuéntralo». No indica síntoma, así que se investigó el código en busca de defectos reales. Se encontraron dos, ambos introducidos por el commit `d6802bd` (tiempo de trabajo) y presentes en la rama `feat/informe-de-tareas`.

Resultado esperado:
1. Tras desplegar un cambio en `app.css`, los navegadores cargan el CSS nuevo (el tiempo de la lista y el input «Min» aparecen con su estilo).
2. Tras completar o reabrir una tarea con tiempo registrado, el botón conserva su etiqueta accesible limpia («Reabrir tarea: Enviar el informe mensual»), sin el tiempo.

## Evidencia y causa raíz

### Defecto 1: CSS nuevo no llega a navegadores con caché

Hechos:
- `templates/application.plush.html:7` carga `/assets/app.css?v=6`.
- `d6802bd` modificó `public/assets/app.css` (clases `.task-time`, `.minutes-input`, `input[type="number"]`, `.field-label-spaced`) pero **no cambió** `?v=6` (el commit no toca `application.plush.html`).
- El servidor responde `Cache-Control: max-age=31536000` (un año) para `/assets/app.css?v=6` (comprobado con `curl -I` contra la app en `localhost:3000`).
- Un navegador que ya tenía `app.css?v=6` en caché sigue usando el CSS antiguo: el tiempo aparece sin estilo y el input numérico con el aspecto por defecto del navegador. Es la misma razón por la que el repositorio ya versiona los assets (`v=6`, `v=5`).

Causa raíz: cambio de un asset sin subir su parámetro de versión, con caché de un año.

### Defecto 2: la etiqueta accesible del toggle incluye el tiempo

Hechos:
- `templates/tasks/index.plush.html` renderiza ahora `<span class="task-title">Título <small class="task-time">10 min</small></span>` cuando `WorkMinutes > 0` (visto en la respuesta real de `localhost:3000/`).
- `public/assets/app.js:11`, en `renderTaskState`, calcula `aria-label` con `task.querySelector(".task-title").textContent`, que ahora incluye el texto del tiempo.
- Resultado tras pulsar el toggle en «Enviar el informe mensual» (10 min): `aria-label` pasa de «Completar tarea: Enviar el informe mensual» (servidor) a «Reabrir tarea: Enviar el informe mensual 10 min» (cliente). Tras recargar vuelve a la forma correcta; la inconsistencia solo existe entre acciones. No afecta a la persistencia.

Causa raíz: el JS asume que `.task-title` contiene solo el título; el commit anterior metió dentro el tiempo.

### Reproducción mínima
1. Defecto 1: abrir `/` en un navegador con `app.css?v=6` ya cacheado; el tiempo sale sin color/tamaño y el input «Min» sin el estilo nuevo. En un perfil limpio sí se ve estilado.
2. Defecto 2: en la lista, con una tarea que tenga minutos, pulsar «Completar» y leer el atributo `aria-label` del botón en el inspector.

### Descartado
`plans/bug-toggle-no-persiste-al-recargar.md` (sin commitear, de otra sesión) describe un fallo del toggle que **ya no existe**: `HEAD` ya envía `POST` con `_method=PUT` y token. No se reutiliza.

### Incertidumbres
- No he abierto un navegador; el defecto 1 se infiere de las cabeceras de caché y del diff. El defecto 2 se infiere del código y del HTML servido.
- No hay síntoma concreto del usuario: si el bug que tiene en mente es otro, este plan no lo cubre.

## Alcance

Incluido:
- Subir las versiones de los assets en el layout (`app.css?v=7`; `app.js?v=6` porque también cambia).
- Que el JS lea el título real, sin el tiempo.

Excluido:
- Cabeceras de caché del servidor, hash automático de assets, cambios de modelo, migraciones o del informe.
- El plan obsoleto del toggle.

## Archivos relevantes

- `templates/application.plush.html` — versiones de `app.css` y `app.js`.
- `templates/tasks/index.plush.html` — estructura del título en cada tarea.
- `public/assets/app.js` — `renderTaskState` construye la etiqueta.
- `actions/tasks_test.go` — test de regresión del HTML.

## Implementación paso a paso

1. `templates/tasks/index.plush.html`: envolver el texto del título en `<span class="task-title-text"><%= t.Title %></span>` dentro de `.task-title`, dejando `<small class="task-time">` fuera del nuevo span:
   `<span class="task-title"><span class="task-title-text"><%= t.Title %></span><%= if (t.WorkMinutes > 0) { %> <small class="task-time" ...>...</small><% } %></span>`
2. `public/assets/app.js:11`: usar `task.querySelector(".task-title-text").textContent`.
3. `templates/application.plush.html`: `app.css?v=7` y `app.js?v=6`.
4. No modificar nada más (el CSS no necesita cambios: `.task-title-text` es un `span` sin estilo propio).

## Tests y validación

Test nuevo en `actions/tasks_test.go`, `Test_TasksIndex_TitleTextExcludesTime`: crear una tarea con título «Con tiempo» y `WorkMinutes: 90`; `GET /`; comprobar que el cuerpo contiene `<span class="task-title-text">Con tiempo</span>` y `1 h 30 min`, y que el tiempo está fuera de ese span (el texto «1 h 30 min» no aparece dentro de él). Falla antes del arreglo (no existe la clase) y pasa después.

Limitaciones: el repositorio no tiene tests de JavaScript ni del caché del navegador, así que:
- Verificación manual: pulsar el toggle en una tarea con tiempo y comprobar `aria-label` sin el tiempo.
- Verificación manual: en una pestaña con caché previa, tras recargar, `curl`/pestaña de red muestran `app.css?v=7` y el CSS nuevo.

Comprobaciones (con PostgreSQL y `soda migrate -e test`, y `GO_ENV=test`):

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -p 1 ./actions ./models
go build ./...
```

Nota: en Windows `gofmt -l .` lista todos los archivos por los finales CRLF, y los tests de `adws` fallan por el script `claude` falso (ya ocurría antes); ninguno está relacionado.

## Riesgos y recuperación

- Subir la versión fuerza una descarga única del CSS/JS; sin riesgo funcional.
- Cualquier asset futuro necesita el mismo cuidado: el servidor cachea un año.
- Recuperación: `git revert` del commit; no hay cambios de esquema ni de datos.
