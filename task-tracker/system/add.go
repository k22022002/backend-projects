package system

import (
	"fmt"
	"time"

	"task-tracker/component"
	"task-tracker/storage"
)

func AddTask(desc string) {
	id := len(component.Descriptions) + 1
	component.SetDescription(id, desc)
	component.SetStatus(id, component.Todo)
	now := time.Now()
	component.SetTime(id, now, now)
	fmt.Printf("Task added successfully (ID: %d)\n", id)
	storage.SaveTasks()
}
