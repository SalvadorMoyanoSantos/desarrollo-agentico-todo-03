package models

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// WorkMinutesMax es el máximo de minutos de trabajo permitido por tarea.
const WorkMinutesMax = 99999

// TaskOrder es el orden en que se listan las tareas.
const TaskOrder = "position asc, created_at asc, id asc"

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Position  int       `json:"position" db:"position" form:"-"`
	// WorkMinutes es el tiempo de trabajo dedicado a la tarea, en minutos.
	WorkMinutes int       `json:"work_minutes" db:"work_minutes" form:"-"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	if t.WorkMinutes < 0 || t.WorkMinutes > WorkMinutesMax {
		verrs.Add("work_minutes", fmt.Sprintf("El tiempo de trabajo debe estar entre 0 y %d minutos.", WorkMinutesMax))
	}

	return verrs, nil
}

// FormatMinutes da formato legible a una duración en minutos; 0 se muestra como "—".
func FormatMinutes(m int) string {
	h, min := m/60, m%60
	switch {
	case m <= 0:
		return "—"
	case h == 0:
		return fmt.Sprintf("%d min", min)
	case min == 0:
		return fmt.Sprintf("%d h", h)
	}
	return fmt.Sprintf("%d h %d min", h, min)
}

// ParseWorkMinutes convierte el texto del formulario en minutos: vacío es 0 y un valor
// no numérico devuelve -1, que la validación rechaza por estar fuera de rango.
func ParseWorkMinutes(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	m, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return m
}

// WorkTime devuelve el tiempo de trabajo con formato legible.
func (t Task) WorkTime() string {
	return FormatMinutes(t.WorkMinutes)
}

// DayCount es el número de tareas creadas en un día.
type DayCount struct {
	Day   string
	Count int
}

// TaskReport resume el estado de una colección de tareas.
type TaskReport struct {
	Total     int
	Completed int
	Pending   int
	Percent   int
	ByDay     []DayCount
	// TotalMinutes suma el tiempo de trabajo de todas las tareas; TotalTime es su formato legible.
	TotalMinutes int
	TotalTime    string
}

// Report calcula el resumen de las tareas; los días van de más reciente a más antiguo.
func (ts Tasks) Report() TaskReport {
	r := TaskReport{Total: len(ts)}
	counts := map[string]int{}
	for _, t := range ts {
		if t.Completed {
			r.Completed++
		}
		r.TotalMinutes += t.WorkMinutes
		counts[t.CreatedAt.UTC().Format("2006-01-02")]++
	}
	r.Pending = r.Total - r.Completed
	r.TotalTime = FormatMinutes(r.TotalMinutes)
	if r.Total > 0 {
		r.Percent = r.Completed * 100 / r.Total
	}
	for day, n := range counts {
		r.ByDay = append(r.ByDay, DayCount{Day: day, Count: n})
	}
	sort.Slice(r.ByDay, func(i, j int) bool { return r.ByDay[i].Day > r.ByDay[j].Day })
	return r
}
