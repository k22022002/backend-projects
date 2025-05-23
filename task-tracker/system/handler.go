package system

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"task-tracker/entity"
	"task-tracker/storage/sqlite"

	"github.com/gorilla/mux"
)

// CreateTask handles POST /tasks
func CreateTask(w http.ResponseWriter, r *http.Request) {
	var task entity.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	stmt, err := sqlite.DB.Prepare("INSERT INTO tasks(description, status, created_at, updated_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		http.Error(w, "Failed to prepare query", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(task.Description, task.Status, now, now)
	if err != nil {
		http.Error(w, "Failed to insert task", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	task.ID = int(id)
	task.CreatedAt = now
	task.UpdatedAt = now

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// GetAllTasks handles GET /tasks and supports filtering by status
func GetAllTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = sqlite.DB.Query("SELECT id, description, status, created_at, updated_at FROM tasks WHERE status = ?", status)
	} else {
		rows, err = sqlite.DB.Query("SELECT id, description, status, created_at, updated_at FROM tasks")
	}

	if err != nil {
		http.Error(w, "Failed to query tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []entity.Task
	for rows.Next() {
		var t entity.Task
		err := rows.Scan(&t.ID, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan task", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTaskByID handles GET /tasks/{id}
func GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	var t entity.Task
	err := sqlite.DB.QueryRow("SELECT id, description, status, created_at, updated_at FROM tasks WHERE id = ?", id).
		Scan(&t.ID, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to query task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// UpdateTask handles PUT /tasks/{id}
func UpdateTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	var task entity.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	stmt, err := sqlite.DB.Prepare("UPDATE tasks SET description = ?, status = ?, updated_at = ? WHERE id = ?")
	if err != nil {
		http.Error(w, "Failed to prepare update", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(task.Description, task.Status, now, id)
	if err != nil {
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	task.ID = id
	task.UpdatedAt = now
	task.CreatedAt = "" // optional: omit or re-fetch

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// DeleteTask handles DELETE /tasks/{id}
func DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	stmt, err := sqlite.DB.Prepare("DELETE FROM tasks WHERE id = ?")
	if err != nil {
		http.Error(w, "Failed to prepare delete", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking affected rows", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	//  Gửi phản hồi JSON sau khi xóa thành công
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Task with ID %d deleted successfully.", id),
	})
}
