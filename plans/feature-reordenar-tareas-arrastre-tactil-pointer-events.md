# Feature: Arrastre táctil para reordenar tareas con Pointer Events

## Encargo y criterios de aceptación

Encargo: «Escribe un plan nuevo de todos modos, para un alcance distinto: arrastre táctil con Pointer Events». Parte de la funcionalidad ya implementada en `plans/feature-reordenar-tareas-drag-and-drop.md` (commit `ad9b3de`), que excluyó el táctil porque HTML5 Drag and Drop no funciona en la mayoría de navegadores móviles.

Criterios (explícitos → E, inferidos → I):

1. (E) En pantallas táctiles el usuario puede arrastrar una tarea desde su asa y soltarla en otra posición, usando Pointer Events.
2. (I) El mismo código sirve para ratón, lápiz y táctil: el arrastre con ratón sigue funcionando igual de cara al usuario.
3. (I) El orden se persiste con el endpoint existente `PUT /tasks/reorder`; no hay cambios de servidor.
4. (I) Mientras se arrastra con el dedo la página no hace scroll por el gesto sobre el asa, y la lista hace auto-scroll si el dedo llega al borde superior/inferior de la ventana.
5. (I) Un gesto cancelado (`pointercancel`, p. ej. llamada entrante o gesto del sistema) devuelve la lista al orden previo sin llamar al servidor.
6. (I) Si el guardado falla se restaura el orden previo y se avisa (comportamiento actual de `saveOrder`).
7. (I) El teclado (↑/↓ en el asa) y el aviso `aria-live` siguen funcionando sin cambios.
8. (I) Tocar el asa sin mover no guarda nada, y tocar el resto de la tarea (toggle, editar, eliminar, scroll vertical sobre el título) sigue funcionando como ahora.

## Comportamiento actual y propuesta

Actual (`public/assets/app.js`): el arrastre usa HTML5 DnD (`pointerdown` pone `draggable="true"` en el `<li>`, y se escuchan `dragstart`, `dragover`, `dragend`). En táctil esos eventos no se disparan de forma fiable, así que no se puede reordenar con el dedo. `.task-handle` ya tiene `touch-action: none` en `public/assets/app.css:171`, así que el CSS base está preparado.

Propuesta: sustituir la ruta HTML5 DnD por Pointer Events para todos los tipos de puntero (una sola ruta de código en lugar de mantener dos). `pointerdown` en el asa inicia el arrastre y captura el puntero (`setPointerCapture`); `pointermove` recoloca el `<li>` según la posición vertical del puntero; `pointerup` guarda si el orden cambió; `pointercancel` revierte. Se reutilizan `taskItems`, `restoreOrder`, `sameOrder` y `saveOrder` tal cual.

Decisión: reemplazar en vez de añadir en paralelo. Mantener HTML5 DnD junto a Pointer Events provoca eventos duplicados en ratón (y en algunos navegadores `dragstart` cancela los pointer events), y duplica la lógica de reubicación.

## Alcance

Incluido:
- Reescritura de la sección de arrastre de `public/assets/app.js` con Pointer Events (ratón + táctil + lápiz).
- Auto-scroll cerca de los bordes de la ventana durante el arrastre.
- Ajustes mínimos de CSS (estado de arrastre, `user-select`, área táctil del asa) y de la plantilla (texto del `title`, versión de assets).
- Verificación manual en emulación táctil; test de Go que fija el marcado necesario.

Excluido:
- Cambios en servidor, modelo, migraciones, rutas o `TasksReorder`.
- Pulsación larga sobre toda la fila para arrastrar (solo el asa inicia el arrastre).
- Animaciones de desplazamiento (FLIP) y arrastre con "fantasma" que sigue al dedo; se mantiene el reordenado en vivo actual.
- Librerías externas y framework de tests JS (no existe en el repo).
- Reordenación del teclado: no cambia.

## Diseño por capas

### JavaScript (`public/assets/app.js`)
Eliminar `dragState` basado en DnD y los listeners `pointerdown`/`pointerup` que ponen/quitan `draggable`, y `dragstart`/`dragover`/`dragend`. Sustituir por:

- `pointerdown` (delegado en `document`): ignorar si `event.button !== 0` (solo en ratón; táctil siempre es 0) o si el objetivo no está en `[data-task-handle]`. Guardar `drag = {list, task, handle, pointerId, previous: taskItems(list)}`, llamar a `handle.setPointerCapture(event.pointerId)`, añadir clase `task-dragging` al `<li>` y `task-list-dragging` a la lista (para `user-select: none`). No llamar a `preventDefault` en `pointerdown` si hace falta que el foco llegue al asa (en táctil el `touch-action: none` ya evita el scroll).
- `pointermove`: ignorar si no hay `drag` o `event.pointerId` no coincide. Buscar la tarea bajo el puntero con `document.elementFromPoint(list.getBoundingClientRect().left + 1, event.clientY)?.closest(".task")` (X fija dentro de la lista, para que el dedo no tenga que quedar encima de la fila). Con captura de puntero, `event.target` es siempre el asa, por eso no se puede usar. Si hay `over` distinto del arrastrado, reubicar con `insertBefore` igual que el handler `dragover` actual (mitad superior/inferior de `over`). Hecho en `requestAnimationFrame` para no recolocar en cada evento.
- Auto-scroll: si `event.clientY` está a menos de ~60 px del borde superior/inferior de la ventana, `window.scrollBy(0, ±velocidad)` en un bucle `requestAnimationFrame` que se cancela al soltar; el bucle usa el último `clientY` conocido y vuelve a ejecutar la reubicación, porque el scroll mueve las filas bajo el dedo sin que haya `pointermove`.
- `pointerup`: soltar captura, quitar clases, cancelar el bucle de auto-scroll; si `!sameOrder(drag.previous, taskItems(list))` llamar a `saveOrder(list, drag.previous)`. Limpiar `drag`.
- `pointercancel` y `lostpointercapture` sin `pointerup` previo: `restoreOrder(list, drag.previous)`, limpiar estado, no llamar al servidor.
- `contextmenu` sobre el asa: `preventDefault` (la pulsación larga en táctil abre el menú contextual en algunos navegadores).
- Foco: tras soltar, devolver el foco al asa solo si el arrastre fue con ratón/lápiz (`event.pointerType !== "touch"`), para no abrir foco/teclado virtual innecesariamente.
- Anuncio `aria-live`: tras soltar con cambio, escribir en `#reorder-status` el mismo texto que el teclado («Tarea movida a la posición N de M»); extraer una función `announceMove(list, task)` compartida con el handler de teclado.
- Sin dependencias; ES5 en el estilo del archivo (`var`, funciones normales, sin `?.` ni flechas) y comentarios en español como los existentes.

### CSS (`public/assets/app.css`)
- `.task-handle`: ya tiene `touch-action: none` y `cursor: grab`. Ampliar el ancho táctil: en el `@media (max-width: 760px)` el asa hereda 28 px de ancho (24 px en la rejilla móvil `.task`, línea ~465), por debajo de los 44 px de las demás dianas; subir a 44 px de alto (ya) y al menos 32 px de ancho ajustando `grid-template-columns` de esas reglas (p. ej. `32px 36px minmax(0,1fr) auto`) y comprobar que no desborda en ≤380 px.
- `.task-dragging`: añadir `cursor: grabbing` y mantener `opacity: 0.5`. Opcionalmente `background` con el color de superficie para distinguirla.
- `.task-list-dragging { user-select: none; -webkit-user-select: none; }` y, mientras dura el arrastre, `cursor: grabbing` en la lista.
- Respetar `prefers-reduced-motion` (ya hay un bloque en el archivo): no se añaden transiciones, así que no requiere cambio.

### Renderizado (`templates/tasks/index.plush.html`, `templates/application.plush.html`)
- Cambiar `title` del asa de «Arrastra para reordenar (o usa ↑ y ↓)» a un texto válido también en táctil (p. ej. «Arrastra para reordenar (o usa ↑ y ↓ con el teclado)»). Tocar el texto exige actualizar cualquier test que lo asserte.
- Subir cache-busting a `app.css?v=5` y `app.js?v=5` (valor actual `?v=4`).
- No hay cambios de datos ni de atributos `data-*`; el `draggable` ya no se usa y no aparece en el marcado.

### Persistencia, modelo, acción y ruta
Sin cambios. `TasksReorder` y `PUT /tasks/reorder` valen tal cual; el JS conserva el formato de petición (`_method=PUT`, `authenticity_token`, `ids` repetidos).

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `public/assets/app.js` | Sustituir DnD HTML5 por Pointer Events; auto-scroll; `announceMove` compartida con teclado. |
| `public/assets/app.css` | Área táctil del asa, estados `task-dragging`/`task-list-dragging`. |
| `templates/tasks/index.plush.html` | Texto del `title` del asa. |
| `templates/application.plush.html` | Versión de assets `?v=5`. |
| `actions/tasks_test.go` | Test de marcado (asa, `data-reorder-url`, ausencia de `draggable`). |
| `actions/tasks.go`, `actions/app.go` | Sin cambios (referencia: `TasksReorder`). |

## Implementación paso a paso

1. En `app.js`, extraer `announceMove(list, task)` del handler de teclado y usarlo desde allí (sin cambiar comportamiento).
2. Eliminar el bloque DnD HTML5 (`dragState`, `pointerdown`/`pointerup` de `draggable`, `dragstart`, `dragover`, `dragend`).
3. Implementar `pointerdown`/`pointermove`/`pointerup`/`pointercancel`/`lostpointercapture` según el diseño, con la lógica de reubicación y `saveOrder` existentes.
4. Añadir el auto-scroll con `requestAnimationFrame` y cancelarlo en todos los finales de arrastre (up, cancel, lost capture).
5. Ajustar CSS (rejilla móvil, `cursor`, `user-select`) y plantilla (`title`, `?v=5`).
6. Añadir/ajustar el test de marcado y ejecutar la validación.
7. Verificación manual en emulación táctil y con ratón (sección siguiente).

## Tests y validación

No hay framework de tests JS en el repo; no se introduce uno. Lo que sí se automatiza:

- `actions/tasks_test.go`: test del índice que comprueba que el HTML contiene `data-task-handle`, `data-reorder-url="/tasks/reorder"`, `data-token`, `id="reorder-status"` y `app.js?v=5`, y **no** contiene `draggable`. Los tests existentes de `TasksReorder` cubren el servidor y deben seguir pasando sin cambios.
- Requiere PostgreSQL accesible y la base de test con migraciones aplicadas (como el resto de tests de acción).

Comandos: `gofmt -l .`, `go vet ./...`, `go test -p 1 ./...`, `go build ./...`.

Validación manual (la parte decisiva; usar el skill `playwright-cli` o el navegador con emulación táctil, y si es posible un móvil real en la red local):
- Táctil (emulación o dispositivo): arrastrar el asa hacia arriba y abajo, soltar, recargar y comprobar que el orden persiste; que la página no hace scroll durante el gesto sobre el asa; que scroll vertical sobre el título sigue funcionando.
- Lista larga: arrastrar cerca del borde y comprobar el auto-scroll y que se detiene al soltar.
- `pointercancel`: simular con DevTools o `dispatchEvent(new PointerEvent("pointercancel", ...))`; el orden vuelve al previo y no hay petición a `/tasks/reorder` en la pestaña Red.
- Fallo de red (servidor detenido o offline): se restaura el orden y aparece la alerta.
- Ratón: arrastrar, soltar, recargar; confirmar que no hay regresión (sin doble guardado, una única petición por arrastre).
- Teclado y lector de pantalla: ↑/↓ sigue moviendo, el `aria-live` anuncia la posición también tras soltar con puntero.
- Toggle, editar y eliminar siguen funcionando tras reordenar.
- Anchos 375 px y ≤380 px: sin desbordes en la fila con la nueva columna del asa.

## Riesgos y recuperación

- **Safari iOS / Chrome Android**: `touch-action: none` en el asa es imprescindible; si el scroll sigue disparándose, `pointercancel` revertirá el orden (por diseño). Comprobar en dispositivo real porque la emulación no reproduce el scroll nativo.
- **Captura de puntero**: con `setPointerCapture` el `event.target` no sirve para localizar la fila destino; el diseño usa `elementFromPoint`. Si en algún navegador no se soporta, degradar a no iniciar el arrastre (el teclado sigue disponible).
- **Eventos duplicados / doble guardado**: asegurar que solo `pointerup` llama a `saveOrder`, y que `pointercancel`/`lostpointercapture` limpian el estado antes de que pueda dispararse `pointerup`.
- **Cache de assets**: sin subir a `?v=5` los navegadores pueden servir el JS antiguo (que aún usa DnD).
- **Pérdida del foco del asa** tras reordenar el DOM: el `insertBefore` mantiene el nodo, pero verificar el foco en teclado y ratón.
- **Árbol de trabajo**: `plans/bug-toggle-no-persiste-al-recargar.md` está sin versionar y es de otro asunto; no tocarlo.
- Recuperación: el cambio es solo de frontend (JS/CSS/plantilla); revertir el commit y volver a `?v=4` devuelve el arrastre HTML5 anterior, sin migraciones ni datos afectados.

## Incertidumbres pendientes

- ¿Se tiene un dispositivo móvil real (iOS/Android) para la verificación manual? La emulación no cubre el scroll nativo.
- ¿Basta con iniciar el arrastre desde el asa, o se quiere también pulsación larga sobre toda la fila? (Este plan solo cubre el asa.)
- ¿Se desea un "fantasma" visual que siga al dedo en lugar del reordenado en vivo actual? (Fuera de alcance; sería una iteración aparte.)
