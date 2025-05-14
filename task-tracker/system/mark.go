package system

import (
	"fmt"

	"task-tracker/component"
	"task-tracker/entity"
	"task-tracker/storage"
)

func MarkStatus(id entity.EntityID, status component.Status) {
	if _, ok := component.Descriptions[int(id)]; !ok {
		fmt.Printf("Error: Task with ID %d not found.\n", id)
		return
	}
	component.SetStatus(int(id), status)
	component.UpdateTime(int(id))
	fmt.Printf("Task %d marked as %s\n", id, status)
	storage.SaveTasks()
}
