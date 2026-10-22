package models

import (
	"strings"
	"time"
)

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) Test_Task_Position_PersistsAndOrdersDeterministically() {
	for _, task := range []*Task{
		{Title: "B", Position: 2},
		{Title: "A1", Position: 1},
		{Title: "A2", Position: 1},
	} {
		verrs, err := ms.DB.ValidateAndCreate(task)
		ms.NoError(err)
		ms.False(verrs.HasAny())
	}

	tasks := Tasks{}
	ms.NoError(ms.DB.Order(TaskOrder).All(&tasks))
	ms.Len(tasks, 3)
	ms.Equal(1, tasks[0].Position)
	ms.Equal("A1", tasks[0].Title)
	ms.Equal("A2", tasks[1].Title)
	ms.Equal("B", tasks[2].Title)
}

func (ms *ModelSuite) Test_Tasks_Report_Empty() {
	r := Tasks{}.Report()
	ms.Equal(0, r.Total)
	ms.Equal(0, r.Percent)
	ms.Len(r.ByDay, 0)
}

func (ms *ModelSuite) Test_Tasks_Report_Counts() {
	day1 := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)
	r := Tasks{
		{Title: "a", CreatedAt: day1},
		{Title: "b", CreatedAt: day1, Completed: true},
		{Title: "c", CreatedAt: day2},
	}.Report()
	ms.Equal(3, r.Total)
	ms.Equal(1, r.Completed)
	ms.Equal(2, r.Pending)
	ms.Equal(33, r.Percent)
	ms.Equal([]DayCount{{"2026-10-02", 1}, {"2026-10-01", 2}}, r.ByDay)
}
