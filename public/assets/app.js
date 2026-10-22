// Refleja en la tarea el estado completado/pendiente.
function renderTaskState(task, completed) {
  var button = task.querySelector(".task-toggle button");
  var status = task.querySelector(".task-status");

  task.classList.toggle("task-completed", completed);

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title-text").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }
}

// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var previous = task.classList.contains("task-completed");
  renderTaskState(task, !previous);

  // Mismo formato que el envío nativo: POST urlencoded con _method=PUT y token CSRF.
  fetch(form.action, {
    method: "POST",
    body: new URLSearchParams(new FormData(form)),
    credentials: "same-origin",
  }).then(function (response) {
    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }
  }).catch(function () {
    renderTaskState(task, previous);
    window.alert("No se pudo guardar el cambio. Inténtalo de nuevo.");
  });
});

// Orden actual de las tareas de la lista, como array de elementos <li>.
function taskItems(list) {
  return Array.prototype.slice.call(list.querySelectorAll(".task"));
}

// Reinserta las tareas en el orden indicado.
function restoreOrder(list, order) {
  order.forEach(function (task) {
    list.appendChild(task);
  });
}

function sameOrder(a, b) {
  return a.length === b.length && a.every(function (task, i) {
    return task === b[i];
  });
}

// Guarda el orden actual; si falla, vuelve al orden previo y avisa.
function saveOrder(list, previousOrder) {
  var params = new URLSearchParams();
  params.append("_method", "PUT");
  params.append("authenticity_token", list.dataset.token);
  taskItems(list).forEach(function (task) {
    params.append("ids", task.dataset.taskId);
  });

  fetch(list.dataset.reorderUrl, {
    method: "POST",
    body: params,
    credentials: "same-origin",
  }).then(function (response) {
    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }
  }).catch(function () {
    restoreOrder(list, previousOrder);
    window.alert("No se pudo guardar el orden. Inténtalo de nuevo.");
  });
}

// Anuncia la nueva posición de la tarea a los lectores de pantalla.
function announceMove(list, task) {
  var items = taskItems(list);
  var status = document.getElementById("reorder-status");
  if (status) {
    status.textContent = "Tarea movida a la posición " + (items.indexOf(task) + 1) + " de " + items.length;
  }
}

var drag = null;
var AUTOSCROLL_EDGE = 60;
var AUTOSCROLL_STEP = 12;

// Coloca la tarea arrastrada según la posición vertical del puntero.
function moveDraggedTask() {
  if (!drag) {
    return;
  }

  // Con la captura del puntero event.target es siempre el asa, así que se localiza la fila por coordenadas.
  var rect = drag.list.getBoundingClientRect();
  var element = document.elementFromPoint(rect.left + 1, drag.y);
  var over = element && element.closest(".task");
  if (!over || over === drag.task || !drag.list.contains(over)) {
    return;
  }

  var box = over.getBoundingClientRect();
  var after = drag.y > box.top + box.height / 2;
  var reference = after ? over.nextSibling : over;
  if (reference !== drag.task) {
    drag.list.insertBefore(drag.task, reference);
  }
}

// Desplaza la ventana mientras el puntero está cerca de un borde.
function autoScroll() {
  if (!drag) {
    return;
  }

  if (drag.y < AUTOSCROLL_EDGE) {
    window.scrollBy(0, -AUTOSCROLL_STEP);
    moveDraggedTask();
  } else if (drag.y > window.innerHeight - AUTOSCROLL_EDGE) {
    window.scrollBy(0, AUTOSCROLL_STEP);
    moveDraggedTask();
  }
  drag.frame = window.requestAnimationFrame(autoScroll);
}

// Termina el arrastre; con save=false vuelve al orden previo sin llamar al servidor.
function finishDrag(save, pointerType) {
  var state = drag;
  if (!state) {
    return;
  }
  drag = null;

  window.cancelAnimationFrame(state.frame);
  state.task.classList.remove("task-dragging");
  state.list.classList.remove("task-list-dragging");
  if (state.handle.hasPointerCapture && state.handle.hasPointerCapture(state.pointerId)) {
    state.handle.releasePointerCapture(state.pointerId);
  }

  if (!save) {
    restoreOrder(state.list, state.previous);
    return;
  }

  if (!sameOrder(state.previous, taskItems(state.list))) {
    announceMove(state.list, state.task);
    saveOrder(state.list, state.previous);
  }
  if (pointerType !== "touch") {
    state.handle.focus();
  }
}

// El arrastre empieza en el asa y sirve para ratón, lápiz y táctil.
document.addEventListener("pointerdown", function (event) {
  var handle = event.target.closest && event.target.closest("[data-task-handle]");
  var task = handle && handle.closest(".task");
  var list = task && task.closest("[data-task-list]");
  if (!list || drag || event.button !== 0 || !handle.setPointerCapture) {
    return;
  }

  handle.setPointerCapture(event.pointerId);
  drag = {
    list: list,
    task: task,
    handle: handle,
    pointerId: event.pointerId,
    previous: taskItems(list),
    y: event.clientY,
    frame: 0,
  };
  task.classList.add("task-dragging");
  list.classList.add("task-list-dragging");
  drag.frame = window.requestAnimationFrame(autoScroll);
});

document.addEventListener("pointermove", function (event) {
  if (!drag || event.pointerId !== drag.pointerId) {
    return;
  }

  drag.y = event.clientY;
  moveDraggedTask();
});

document.addEventListener("pointerup", function (event) {
  if (drag && event.pointerId === drag.pointerId) {
    finishDrag(true, event.pointerType);
  }
});

document.addEventListener("pointercancel", function (event) {
  if (drag && event.pointerId === drag.pointerId) {
    finishDrag(false);
  }
});

// Reubicar el <li> en el DOM libera la captura: se recupera mientras el arrastre siga activo.
document.addEventListener("lostpointercapture", function (event) {
  if (drag && event.pointerId === drag.pointerId && drag.handle.isConnected) {
    drag.handle.setPointerCapture(event.pointerId);
  }
});

// La pulsación larga en táctil abre el menú contextual sobre el asa.
document.addEventListener("contextmenu", function (event) {
  if (event.target.closest && event.target.closest("[data-task-handle]")) {
    event.preventDefault();
  }
});

// Mueve la tarea con ↑/↓ cuando el asa tiene el foco.
document.addEventListener("keydown", function (event) {
  var handle = event.target.closest && event.target.closest("[data-task-handle]");
  if (!handle || (event.key !== "ArrowUp" && event.key !== "ArrowDown")) {
    return;
  }

  event.preventDefault();
  var task = handle.closest(".task");
  var list = task.closest("[data-task-list]");
  var previous = taskItems(list);

  if (event.key === "ArrowUp" && task.previousElementSibling) {
    list.insertBefore(task, task.previousElementSibling);
  } else if (event.key === "ArrowDown" && task.nextElementSibling) {
    list.insertBefore(task.nextElementSibling, task);
  } else {
    return;
  }

  handle.focus();
  announceMove(list, task);
  saveOrder(list, previous);
});
