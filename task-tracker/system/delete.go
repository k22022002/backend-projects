package system

import (
	"fmt"

	"task-tracker/component"
	"task-tracker/entity"
	"task-tracker/storage"
)

func DeleteTask(id entity.EntityID) {
	if _, ok := component.Descriptions[int(id)]; !ok {
		fmt.Printf("Error: Task with ID %d not found.\n", id)
		return
	}
	delete(component.Descriptions, int(id))
	delete(component.Statuses, int(id))
	delete(component.Times, int(id))
	fmt.Println("Task deleted successfully")
	storage.SaveTasks()
}
