// Refleja en la tarea el estado completado/pendiente.
function renderTaskState(task, completed) {
  var button = task.querySelector(".task-toggle button");
  var status = task.querySelector(".task-status");

  task.classList.toggle("task-completed", completed);

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
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

var dragState = null;

// El <li> solo es arrastrable mientras se pulsa el asa, para no interferir con la selección de texto.
document.addEventListener("pointerdown", function (event) {
  var handle = event.target.closest("[data-task-handle]");
  if (handle) {
    handle.closest(".task").setAttribute("draggable", "true");
  }
});

document.addEventListener("pointerup", function () {
  document.querySelectorAll(".task[draggable='true']").forEach(function (task) {
    if (!dragState) {
      task.removeAttribute("draggable");
    }
  });
});

document.addEventListener("dragstart", function (event) {
  var task = event.target.closest && event.target.closest(".task[draggable='true']");
  var list = task && task.closest("[data-task-list]");
  if (!list) {
    return;
  }

  dragState = { list: list, task: task, previous: taskItems(list) };
  task.classList.add("task-dragging");
  event.dataTransfer.effectAllowed = "move";
  event.dataTransfer.setData("text/plain", task.dataset.taskId);
});

document.addEventListener("dragover", function (event) {
  if (!dragState || !dragState.list.contains(event.target)) {
    return;
  }

  event.preventDefault();
  var over = event.target.closest(".task");
  if (!over || over === dragState.task) {
    return;
  }

  var rect = over.getBoundingClientRect();
  var after = event.clientY > rect.top + rect.height / 2;
  dragState.list.insertBefore(dragState.task, after ? over.nextSibling : over);
});

document.addEventListener("dragend", function () {
  if (!dragState) {
    return;
  }

  var state = dragState;
  dragState = null;
  state.task.classList.remove("task-dragging");
  state.task.removeAttribute("draggable");

  if (!sameOrder(state.previous, taskItems(state.list))) {
    saveOrder(state.list, state.previous);
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
  var items = taskItems(list);
  var status = document.getElementById("reorder-status");
  if (status) {
    status.textContent = "Tarea movida a la posición " + (items.indexOf(task) + 1) + " de " + items.length;
  }
  saveOrder(list, previous);
});
