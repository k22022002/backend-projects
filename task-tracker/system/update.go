package system

import (
	"fmt"
	"task-tracker/component"
	"task-tracker/entity"
	"task-tracker/storage"
)

func UpdateTask(id entity.EntityID, desc string) {
	if desc == "" {
		fmt.Println("Error: Description cannot be empty.")
		return
	}
	if _, ok := component.Descriptions[int(id)]; !ok {
		fmt.Println("Error: Task not found.")
		return
	}
	component.SetDescription(int(id), desc)
	component.UpdateTime(int(id))
	fmt.Println("Task updated successfully")
	storage.SaveTasks()
}
