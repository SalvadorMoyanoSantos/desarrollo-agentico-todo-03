# Chore: Extraer los estilos repetidos en clases CSS reutilizables

## Encargo y resultado esperado

Encargo: «Extraer los estilos repetidos en clases CSS reutilizables y aplicarlas sin cambiar el aspecto ni el comportamiento de la aplicación».

Resultado: `public/assets/app.css` deja de repetir bloques de declaraciones y valores literales. Esos bloques pasan a clases/variables reutilizables, y las plantillas usan las clases nuevas donde haga falta. Es un refactor puro: los estilos calculados (`getComputedStyle`) de cada elemento en cada pantalla, estado y punto de ruptura deben ser idénticos antes y después, y el JS y los tests Go siguen pasando sin cambios.

## Estado actual y restricciones

- Rama `feat/reordenar-tareas-drag-and-drop`; solo hay sin seguimiento `migrations/schema.sql` y `plans/bug-toggle-no-persiste-al-recargar.md`. No tocarlos.
- Aplicación Go/Buffalo. Todo el CSS está en un único archivo, `public/assets/app.css` (521 líneas), sin preprocesador ni build. No añadir herramientas ni dependencias.
- Plantillas Plush: `templates/application.plush.html` (layout, enlaza `/assets/app.css?v=5` y `/assets/app.js?v=5`), `templates/_flash.plush.html`, `templates/tasks/index.plush.html`, `templates/tasks/edit.plush.html`. No hay atributos `style=` en las plantillas.
- `public/assets/app.js` depende de estas clases, que **no se pueden renombrar ni eliminar**: `task`, `task-completed`, `task-title`, `task-status`, `task-actions`, `task-toggle` (selector `.task-toggle button`), `toggle-label`, `task-dragging`, `task-list-dragging`, y los atributos `data-task-*`. JS añade `task-status` por `className`, así que `.task-status` debe seguir siendo autosuficiente (hoy comparte regla con `.sr-only`).
- `actions/tasks_test.go:299` exige `app.js?v=5` en el HTML. Por tanto, **no se cambia la versión de `app.js`**. Sí se sube `app.css?v=5` a `?v=6` para invalidar la caché (ningún test lo comprueba).
- No hay tests de CSS; la validación visual hay que hacerla con el navegador (ver más abajo).
- Puntos de ruptura existentes que hay que preservar: `(max-width: 760px)`, `(max-width: 380px)`, `(hover: hover) and (pointer: fine)`, `(prefers-reduced-motion: reduce)`.

### Repeticiones detectadas (evidencia en `app.css`)

1. Valores literales repetidos:
   - `border-radius: 6px` (~10 veces) → `--radius: 6px`.
   - Tamaño táctil `44px` (`min-height`/`width`/`height`, ~12 veces) → `--tap-size: 44px`. Ojo: en móvil algunas anchuras pasan a 36px/32px; esos valores se quedan como literales.
   - `#ffffff` (body, input, `.button-primary`, `.completion-icon`) → `--color-surface: #ffffff`.
   - `#f7e9e4` (`.nav-link`, hover de `.completion-icon`) → `--color-primary-tint`.
   - `120ms ease` (`.completion-icon`, `.task-actions`) → `--transition-fast`.
   - `1px solid var(--color-border)` (`.task`, `.flash`, `.edit-task-actions`, `.sidebar` móvil) → `--border: 1px solid var(--color-border)`. Ojo: `.sidebar` usa `#efefeb`, que **no** es `--color-border`; se deja literal.
   - `font-size: 0.85rem` (~6 veces) y `0.8rem` (~3) → clases utilitarias de texto.
2. Bloques de declaraciones duplicados:
   - `.task-toggle-button`/`.icon-action` y `.task-handle` repiten `display:inline-flex; align-items:center; justify-content:center; min-height:44px; padding:0; color:muted; background:transparent; border:0; border-radius:6px`. → clase base `.icon-button`. `.task-handle` conserva solo `width:28px`, `cursor:grab`, `touch-action:none`.
   - `.flash` y `.field-error` repiten `padding`, `border-radius` y `font-size:0.85rem` (padding distinto: 0.75rem 0.9rem vs 0.7rem 0.85rem; **se mantienen sus valores**, solo se comparte lo idéntico).
   - `.back-link`, `.nav-link`, `.button`: `display:inline-flex/flex; align-items:center; text-decoration:none; min-height:44px`.
   - Iconos SVG en plantillas: `fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"` repetido en 4 SVG (nav, completar, editar, eliminar). `stroke-width` varía (1.7 / 2), así que se queda como atributo.

## Alcance

Incluido:
- Añadir variables en `:root` y clases reutilizables en `app.css`; reescribir las reglas existentes para apoyarse en ellas.
- Ajustar plantillas solo para añadir la clase compartida (p. ej. `icon-button`, `icon-outline`) sin cambiar estructura, atributos de accesibilidad ni textos.
- Subir `app.css?v=5` → `v=6` en `application.plush.html`.

Excluido:
- Cambiar colores, tamaños, espaciados o cualquier valor visible; rediseñar; renombrar clases usadas por `app.js`; tocar `app.js`, Go, migraciones, tests (salvo que fallen por un motivo ajeno, que se reportaría).
- Unificar valores *casi* iguales (p. ej. `#a03123` vs `#912a20` vs `#a52b25`, paddings de `.flash`/`.field-error`, `#efefeb` vs `--color-border`): hacerlo cambiaría el aspecto.
- Introducir frameworks CSS, preprocesadores o linters.

## Archivos relevantes

- `public/assets/app.css`: única hoja de estilos; se refactoriza.
- `templates/application.plush.html`: layout; SVG del nav y versión del CSS.
- `templates/tasks/index.plush.html`: lista, botones de icono, SVG, formulario de alta.
- `templates/tasks/edit.plush.html`: formulario de edición, `back-link`, botones.
- `templates/_flash.plush.html`: usa `flash flash-<k>`; no debería requerir cambios.
- `public/assets/app.js`: solo lectura; define las clases que no pueden cambiar.
- `actions/tasks_test.go`: restricción `app.js?v=5`.

## Implementación paso a paso

0. **Línea base (antes de cambiar nada).** Con la app en marcha (skill `env:start` o la forma habitual del proyecto), capturar el snapshot de estilos calculados (ver «Tests y validación») y guardarlo fuera del repo (scratchpad), más capturas de pantalla de las pantallas y viewports listados.
1. **Variables.** Añadir a `:root`: `--radius`, `--tap-size`, `--color-surface`, `--color-primary-tint`, `--transition-fast`, `--border`. Sustituir los literales correspondientes uno a uno (mismo valor resuelto). No tocar `.sidebar` `#efefeb`.
2. **Base de botón de icono.** Crear `.icon-button` con el bloque común descrito arriba (más `position:relative; flex-shrink:0; width:var(--tap-size); font:inherit; cursor:pointer; text-decoration:none`, que hoy tiene `.task-toggle-button/.icon-action`; el `.task-handle` no los tiene todos, así que `.task-handle` debe sobrescribir `width:28px` y `cursor:grab`, y **no** debe heredar `position:relative`/`flex-shrink`/`font`/`text-decoration` si eso cambia algo; si hay duda, dejar esos extras en un modificador de `.task-toggle-button, .icon-action` y que la base solo lleve el bloque realmente común). Aplicar `icon-button` en `index.plush.html` a la asa, al botón de alternar y a los iconos de editar/eliminar, conservando `task-toggle-button`, `icon-action`, `icon-danger`, `task-handle`.
   - Cuidado con la **cascada**: las reglas posteriores (`.task-handle:hover`, `.icon-action:hover`, media queries móviles `.task-handle{width:32px}`, `.task-toggle-button,.add-mark{margin-left:-4px}`) deben seguir apareciendo después de la base y con igual especificidad o mayor.
3. **Superficie de aviso.** Extraer lo idéntico de `.flash` y `.field-error` (`border-radius`, `font-size`) a una clase compartida, p. ej. `.notice`; añadirla en `_flash.plush.html` y en las dos plantillas que usan `field-error`. Mantener padding y colores en cada clase propia.
4. **Utilidades de texto.** `.text-sm` (0.85rem) y `.text-xs` (0.8rem) solo donde el valor se repite y el elemento ya tiene una clase propia compatible; si mover el valor altera el orden de la cascada o la especificidad, dejarlo donde está.
5. **Enlaces/controles de 44px.** Extraer `display:inline-flex; align-items:center; min-height:var(--tap-size); text-decoration:none` en una clase (p. ej. `.tap-target`) únicamente si `.back-link`, `.nav-link` y `.button` quedan con el mismo resultado calculado (ojo: `.nav-link` usa `display:flex`, no `inline-flex`; dejarlo como override o no incluirlo).
6. **Iconos SVG.** Crear `.icon-outline { fill:none; stroke:currentColor; stroke-linecap:round; stroke-linejoin:round }`, aplicarla a los 4 SVG de outline y quitar esos tres atributos de presentación (se mantiene `stroke-width`). No tocar los SVG de relleno (`fill="currentColor"`, asa).
7. **Versión de caché.** `application.plush.html`: `app.css?v=5` → `app.css?v=6`. No tocar `app.js?v=5`.
8. **Revisión final.** Confirmar que no sobran reglas huérfanas ni quedan literales sustituibles (`grep -n "6px\|44px" public/assets/app.css`), que las media queries siguen al final y que el orden de la cascada es el esperado.

Cada paso es un cambio pequeño e independiente: tras cada uno, volver a pasar el snapshot (ver validación) para localizar regresiones de inmediato.

## Tests y validación

Automáticos (disponibles en el repo; los tests de acciones necesitan la base de datos de pruebas, ver `compose.yaml`/`database.yml`):

```bash
gofmt -l .
go vet ./...
go test -p 1 ./...
go build ./...
```

Resultado esperado: sin salida de `gofmt`, y todo en verde (`Test_TasksIndex_RendersHandle` sigue pasando porque `app.js?v=5` no cambia).

Comparación visual/computada antes y después (principal, porque no hay tests de CSS). En el navegador, sobre cada pantalla y estado, ejecutar en consola y guardar el JSON:

```js
JSON.stringify([...document.querySelectorAll('body *')].map(e => {
  const s = getComputedStyle(e);
  return [e.tagName, e.className.baseVal ?? e.className,
    ['display','position','width','minHeight','padding','margin','border','borderRadius','color','backgroundColor','fontSize','fontWeight','lineHeight','cursor','opacity','textDecoration','fill','stroke','strokeWidth','strokeLinecap','strokeLinejoin','touchAction','transition']
      .map(p => s[p]).join('|')];
}))
```

Diff del JSON de línea base contra el posterior: debe ser idéntico (salvo el `?v=` del `<link>`, que no es un estilo).

Matriz a cubrir (línea base y final):
- Pantallas/estados: lista con tareas (una completada y otra no), lista vacía, `/tasks/:id/edit`, alta con título vacío (error `field-error`), edición con título vacío (error), mensaje flash de éxito tras crear/editar/borrar.
- Viewports: 1280 px, 761 px, 760 px, 380 px, 375 px (`resize_window`).
- Estados interactivos (comprobar visualmente, la consola no los fuerza): hover de `.task-toggle-button` (círculo y check), hover de `.icon-action`/`.icon-danger`/`.button`/`.button-primary`/`.nav-link`/`.back-link`/`.task-handle`; foco por teclado (`:focus-visible`); acciones ocultas hasta hover en puntero fino; visibles en móvil.
- Comportamiento JS: alternar tarea (clase `task-completed`, `task-status`), reordenar arrastrando y con teclado (↑/↓), clases `task-dragging`/`task-list-dragging`, confirmación de borrado.
- `prefers-reduced-motion`: sin transiciones.

## Riesgos y recuperación

- **Cambio de cascada/especificidad** al mover bloques a una base (p. ej. `.icon-button` antes que `.task-handle`/`.icon-action`): mitigado con el diff de estilos calculados tras cada paso y manteniendo el orden fuente.
- **Diferencias sutiles entre elementos casi iguales** (`display:flex` vs `inline-flex`, paddings, `#efefeb`): no se unifican; solo se extrae lo idéntico.
- **Sustitución de atributos SVG por CSS**: los estilos CSS tienen prioridad sobre atributos de presentación; equivale al resultado actual, pero se verifica con `stroke*` en el snapshot.
- **Caché del navegador**: por eso se sube `app.css?v=6`.
- **Rotura de `app.js`** por renombrar clases: prohibido; comprobar con la prueba funcional del apartado anterior.
- Recuperación: todo el cambio está en 4 archivos de texto; `git restore public/assets/app.css templates/` o revertir el commit devuelve el estado anterior sin migraciones ni datos afectados.
