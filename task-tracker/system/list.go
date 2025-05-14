package system

import (
	"fmt"
	"task-tracker/component"
)

func ListTasks(statusFilter component.Status) {
	found := false
	for id, desc := range component.Descriptions {
		status := component.GetStatus(id)
		if statusFilter == "" || status == statusFilter {
			times := component.GetTime(id)
			fmt.Printf("[%d] %s\n  Status: %s\n  Created: %s\n  Updated: %s\n\n",
				id, desc, status, times.CreatedAt.Format("2006-01-02 15:04"), times.UpdatedAt.Format("2006-01-02 15:04"))
			found = true
		}
	}
	if !found {
		fmt.Println("No tasks found.")
	}
}
