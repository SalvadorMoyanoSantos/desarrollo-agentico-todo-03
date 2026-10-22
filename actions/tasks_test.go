package actions

import (
	"net/url"
	"strings"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_GetDoesNotChangeState() {
	task := as.taskForEditing(false)

	res := as.HTML("/tasks/%s/toggle", task.ID).Get()
	as.GreaterOrEqual(res.Code, 400)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_PostWithMethodOverride() {
	task := as.taskForEditing(false)

	res := as.HTML("/tasks/%s/toggle", task.ID).Post(url.Values{"_method": {"PUT"}})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) createPositioned(title string, position int) *models.Task {
	task := &models.Task{Title: title, Position: position}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	return task
}

func (as *ActionSuite) taskPositions() map[uuid.UUID]int {
	tasks := models.Tasks{}
	as.NoError(as.DB.All(&tasks))
	positions := map[uuid.UUID]int{}
	for _, t := range tasks {
		positions[t.ID] = t.Position
	}
	return positions
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPosition() {
	as.createPositioned("Primera creada", 3)
	as.createPositioned("Segunda creada", 1)
	as.createPositioned("Tercera creada", 2)

	body := as.HTML("/").Get().Body.String()
	second := strings.Index(body, "Segunda creada")
	third := strings.Index(body, "Tercera creada")
	first := strings.Index(body, "Primera creada")
	as.True(second >= 0 && second < third && third < first)
}

func (as *ActionSuite) Test_TasksIndex_RendersHandle() {
	as.createPositioned("Con asa", 1)

	body := as.HTML("/").Get().Body.String()
	as.Contains(body, "data-task-handle")
	as.Contains(body, "data-reorder-url=\"/tasks/reorder\"")
	as.Contains(body, "data-token")
	as.Contains(body, "id=\"reorder-status\"")
	as.Contains(body, "app.js?v=6")
	as.NotContains(body, "draggable")
}

func (as *ActionSuite) Test_TasksCreate_AppendsAtEnd() {
	a := as.createPositioned("Existente A", 1)
	b := as.createPositioned("Existente B", 5)

	res := as.HTML("/tasks").Post(&models.Task{Title: "Al final"})
	as.Equal(303, res.Code)

	created := &models.Task{}
	as.NoError(as.DB.Where("title = ?", "Al final").First(created))
	as.Greater(created.Position, 5)
	as.Equal(1, as.taskPositions()[a.ID])
	as.Equal(5, as.taskPositions()[b.ID])
}

func (as *ActionSuite) Test_TasksReorder_Persists() {
	a := as.createPositioned("Tarea A", 1)
	b := as.createPositioned("Tarea B", 2)
	c := as.createPositioned("Tarea C", 3)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {c.ID.String(), a.ID.String(), b.ID.String()}})
	as.Equal(204, res.Code)

	positions := as.taskPositions()
	as.Equal(1, positions[c.ID])
	as.Equal(2, positions[a.ID])
	as.Equal(3, positions[b.ID])

	body := as.HTML("/").Get().Body.String()
	as.True(strings.Index(body, "Tarea C") < strings.Index(body, "Tarea A"))
	as.True(strings.Index(body, "Tarea A") < strings.Index(body, "Tarea B"))
}

func (as *ActionSuite) assertReorderRejected(ids func(a, b *models.Task) []string) {
	a := as.createPositioned("Tarea A", 1)
	b := as.createPositioned("Tarea B", 2)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": ids(a, b)})
	as.Equal(422, res.Code)

	positions := as.taskPositions()
	as.Equal(1, positions[a.ID])
	as.Equal(2, positions[b.ID])
}

func (as *ActionSuite) Test_TasksReorder_InvalidID() {
	as.assertReorderRejected(func(a, b *models.Task) []string {
		return []string{b.ID.String(), "no-es-un-uuid"}
	})
}

func (as *ActionSuite) Test_TasksReorder_DuplicateIDs() {
	as.assertReorderRejected(func(a, b *models.Task) []string {
		return []string{b.ID.String(), b.ID.String()}
	})
}

func (as *ActionSuite) Test_TasksReorder_MissingTask() {
	as.assertReorderRejected(func(a, b *models.Task) []string {
		return []string{b.ID.String()}
	})
}

func (as *ActionSuite) Test_TasksReorder_UnknownID() {
	as.assertReorderRejected(func(a, b *models.Task) []string {
		return []string{b.ID.String(), uuid.Must(uuid.NewV4()).String()}
	})
}

func (as *ActionSuite) Test_TasksReorder_DoesNotMatchTaskID() {
	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {}})
	as.Equal(204, res.Code)
}

func (as *ActionSuite) Test_TasksReport_EmptyState() {
	res := as.HTML("/report").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksReport_Metrics() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/report").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Informe de tareas")
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "<dd>3</dd>")
	as.Contains(body, "<dd>33 %</dd>")
	as.Contains(body, `href="/report"`)
}

func (as *ActionSuite) Test_TasksCreate_WithWorkMinutes() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Con tiempo"}, "work_minutes": {"90"}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal(90, task.WorkMinutes)
}

func (as *ActionSuite) Test_TasksCreate_EmptyWorkMinutes() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Sin tiempo"}, "work_minutes": {""}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal(0, task.WorkMinutes)
}

func (as *ActionSuite) Test_TasksCreate_NegativeWorkMinutes() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Negativo"}, "work_minutes": {"-5"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El tiempo de trabajo debe estar entre 0 y 99999 minutos.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksEdit_PrefillsWorkMinutes() {
	task := as.taskForEditing(false)
	task.WorkMinutes = 75
	as.NoError(as.DB.Update(task))

	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="75"`)
	as.Contains(res.Body.String(), "Tiempo de trabajo (minutos)")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsWorkMinutes() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Igual"}, "work_minutes": {"120"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(120, task.WorkMinutes)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidWorkMinutes() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Igual"}, "work_minutes": {"100000"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El tiempo de trabajo debe estar entre 0 y 99999 minutos.")
	as.NoError(as.DB.Reload(task))
	as.Equal(0, task.WorkMinutes)
}

func (as *ActionSuite) Test_TasksIndex_ShowsWorkTime() {
	as.LoadFixture("tres tareas")
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "1 h 30 min")
}

func (as *ActionSuite) Test_TasksReport_ShowsWorkTime() {
	as.LoadFixture("tres tareas")
	res := as.HTML("/report").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Contains(body, "Tiempo total")
	as.Contains(body, "<dd>2 h 15 min</dd>")
	as.Contains(body, `<th scope="col">Tiempo</th>`)
	as.Contains(body, "<td>1 h 30 min</td>")
}

func (as *ActionSuite) Test_TasksIndex_TitleTextExcludesTime() {
	task := &models.Task{Title: "Con tiempo", WorkMinutes: 90}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Contains(body, `<span class="task-title-text">Con tiempo</span>`)
	as.Contains(body, "1 h 30 min")
}
