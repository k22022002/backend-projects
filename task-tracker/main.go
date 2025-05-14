package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"task-tracker/component"
	"task-tracker/entity"
	"task-tracker/storage"
	"task-tracker/system"
)

func main() {
	// Load dữ liệu từ file khi khởi chạy
	storage.LoadTasks()

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  add <description>")
		fmt.Println("  list [status]")
		fmt.Println("  update <id> <new description>")
		fmt.Println("  delete <id>")
		fmt.Println("  mark <id> <todo|in-progress|done>")
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing task description.")
			return
		}
		desc := strings.Join(os.Args[2:], " ")
		system.AddTask(desc)

	case "list":
		if len(os.Args) == 3 {
			status := component.Status(os.Args[2])
			if status != component.Todo && status != component.InProgress && status != component.Done {
				fmt.Println("Error: Invalid status. Use todo, in-progress, or done.")
				return
			}
			system.ListTasks(status)
		} else {
			system.ListTasks("")
		}

	case "update":
		if len(os.Args) < 4 {
			fmt.Println("Error: Missing task ID or new description.")
			return
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("Error: Invalid task ID.")
			return
		}
		desc := strings.Join(os.Args[3:], " ")
		system.UpdateTask(entity.EntityID(id), desc)

	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing task ID.")
			return
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("Error: Invalid task ID.")
			return
		}
		system.DeleteTask(entity.EntityID(id))

	case "mark":
		if len(os.Args) < 4 {
			fmt.Println("Error: Missing task ID or status.")
			return
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("Error: Invalid task ID.")
			return
		}
		status := component.Status(os.Args[3])
		if status != component.Todo && status != component.InProgress && status != component.Done {
			fmt.Println("Error: Invalid status. Use todo, in-progress, or done.")
			return
		}
		system.MarkStatus(entity.EntityID(id), status)

	default:
		fmt.Println("Unknown command:", cmd)
	}
}
