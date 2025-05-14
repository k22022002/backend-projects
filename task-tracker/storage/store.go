package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"task-tracker/component"
)

type Task struct {
	ID          int              `json:"eid"`
	Description string           `json:"description"`
	Status      component.Status `json:"status"`
	CreatedAt   string           `json:"createdAt"`
	UpdatedAt   string           `json:"updatedAt"`
}

const filePath = "tasks.json"

// SaveTasks writes all tasks from the in-memory component to a JSON file.
func SaveTasks() {
	tasks := make([]Task, 0)

	for id, desc := range component.Descriptions {
		status := component.GetStatus(id)
		timeData := component.GetTime(id)
		task := Task{
			ID:          id,
			Description: desc,
			Status:      status,
			CreatedAt:   timeData.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   timeData.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		tasks = append(tasks, task)
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Pretty print JSON to file
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		fmt.Println("Error encoding tasks to JSON:", err)
		return
	}
	_, err = file.Write(data)
	if err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

// LoadTasks reads tasks from a JSON file and loads them into the component.
func LoadTasks() {
	file, err := os.Open(filePath)
	if err != nil {
		// No existing file, this may be fine on first run
		return
	}
	defer file.Close()

	var tasks []Task
	err = json.NewDecoder(file).Decode(&tasks)
	if err != nil {
		fmt.Println("Error decoding tasks from file:", err)
		return
	}

	for _, task := range tasks {
		component.SetDescription(task.ID, task.Description)
		component.SetStatus(task.ID, task.Status)

		createdAt, err1 := time.Parse("2006-01-02 15:04:05", task.CreatedAt)
		updatedAt, err2 := time.Parse("2006-01-02 15:04:05", task.UpdatedAt)

		if err1 == nil && err2 == nil {
			component.SetTime(task.ID, createdAt, updatedAt)
		}
	}
}
