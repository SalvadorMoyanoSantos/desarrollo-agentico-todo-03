package actions

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

const msgDueDateInvalid = "La fecha límite no es válida."

// dueDateField lee la fecha límite enviada en el formulario. present indica si el campo
// llegó: si no llegó, la fecha de la tarea no se toca. Un valor vacío significa "sin fecha".
func dueDateField(c buffalo.Context) (raw string, present bool, err error) {
	if err := c.Request().ParseForm(); err != nil {
		return "", false, err
	}
	values, present := c.Request().Form["due_date"]
	if !present || len(values) == 0 {
		return "", false, nil
	}
	return strings.TrimSpace(values[0]), true, nil
}

// parseDueDate convierte el valor del formulario en fecha límite (YYYY-MM-DD). Vacío = nil.
func parseDueDate(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	date, err := time.Parse(models.DueDateLayout, raw)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

// TasksIndex muestra la lista de tareas y el formulario de creación.
func TasksIndex(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	tasks := models.Tasks{}
	if err := tx.Order(models.TaskOrder).All(&tasks); err != nil {
		return err
	}

	c.Set("tasks", tasks)
	c.Set("task", &models.Task{})
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/index.plush.html"))
}

// TasksCreate crea una tarea nueva a partir del formulario.
func TasksCreate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := c.Bind(task); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(task.Title)

	raw, _, err := dueDateField(c)
	if err != nil {
		return err
	}
	dueDate, dueErr := parseDueDate(raw)
	task.DueDate = dueDate

	var last int
	if err := tx.RawQuery("SELECT COALESCE(MAX(position), 0) FROM tasks").First(&last); err != nil {
		return err
	}
	task.Position = last + 1

	verrs, err := task.Validate(tx)
	if err != nil {
		return err
	}
	if dueErr != nil {
		verrs.Add("due_date", msgDueDateInvalid)
	}

	if verrs.HasAny() {
		tasks := models.Tasks{}
		if err := tx.Order(models.TaskOrder).All(&tasks); err != nil {
			return err
		}

		c.Set("tasks", tasks)
		c.Set("task", task)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/index.plush.html"))
	}

	if err := tx.Create(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea creada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksEdit muestra el formulario de edición de una tarea existente.
func TasksEdit(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	c.Set("task", task)
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/edit.plush.html"))
}

// TasksUpdate actualiza el título y la fecha límite si el formulario la envía,
// conservando los demás datos de la tarea.
func TasksUpdate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	input := struct {
		Title string `form:"title" json:"title"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(input.Title)

	raw, hasDueDate, err := dueDateField(c)
	if err != nil {
		return err
	}
	var dueErr error
	if hasDueDate {
		task.DueDate, dueErr = parseDueDate(raw)
	}

	verrs, err := task.Validate(tx)
	if err != nil {
		return err
	}
	if dueErr != nil {
		verrs.Add("due_date", msgDueDateInvalid)
	}
	if verrs.HasAny() {
		c.Set("task", task)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/edit.plush.html"))
	}

	if err := tx.Update(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea actualizada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksToggle alterna una tarea entre completada y pendiente.
func TasksToggle(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	task.Completed = !task.Completed
	if err := tx.Update(task); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksReorder reasigna la posición de todas las tareas según la lista de IDs recibida.
func TasksReorder(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	input := struct {
		IDs []string `form:"ids" json:"ids"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}

	tasks := models.Tasks{}
	if err := tx.Order(models.TaskOrder).All(&tasks); err != nil {
		return err
	}

	if len(input.IDs) != len(tasks) {
		return c.Error(http.StatusUnprocessableEntity, errors.New("la lista de tareas no coincide"))
	}

	byID := make(map[uuid.UUID]*models.Task, len(tasks))
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
	}

	ordered := make([]*models.Task, 0, len(input.IDs))
	for _, raw := range input.IDs {
		id, err := uuid.FromString(raw)
		if err != nil {
			return c.Error(http.StatusUnprocessableEntity, err)
		}
		task, ok := byID[id]
		if !ok {
			return c.Error(http.StatusUnprocessableEntity, errors.New("tarea desconocida o repetida"))
		}
		delete(byID, id)
		ordered = append(ordered, task)
	}

	for i, task := range ordered {
		if task.Position == i+1 {
			continue
		}
		task.Position = i + 1
		if err := tx.Update(task); err != nil {
			return err
		}
	}

	return c.Render(http.StatusNoContent, nil)
}

// TasksDestroy elimina una tarea.
func TasksDestroy(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	if err := tx.Destroy(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea eliminada.")
	return c.Redirect(http.StatusSeeOther, "/")
}
