package system

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task-tracker/entity"

	"github.com/gorilla/mux"
)

func (h *Handler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("userID")
	userID, ok := userIDVal.(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.DB.Query("SELECT id, description, status, created_at, updated_at, user_id FROM tasks WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []entity.Task{}
	for rows.Next() {
		var task entity.Task
		rows.Scan(&task.ID, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.UserID)
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	prettyJSON, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}
	w.Write(prettyJSON)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	val := r.Context().Value("userID")
	userID, ok := val.(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	var task entity.Task
	err := h.DB.QueryRow("SELECT id, description, status, created_at, updated_at, user_id FROM tasks WHERE id = ? AND user_id = ?", id, userID).
		Scan(&task.ID, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.UserID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	prettyJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}
	w.Write(prettyJSON)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var task entity.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Kiểm tra description
	if strings.TrimSpace(task.Description) == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
		return
	}

	// Kiểm tra status hợp lệ
	validStatuses := map[string]bool{
		"todo":        true,
		"in_progress": true,
		"done":        true,
	}
	if _, ok := validStatuses[task.Status]; !ok {
		http.Error(w, "Status must be one of: todo, in_progress, done", http.StatusBadRequest)
		return
	}

	timeNow := time.Now().Format(time.RFC3339)

	res, err := h.DB.Exec(
		"INSERT INTO tasks (description, status, created_at, updated_at, user_id) VALUES (?, ?, ?, ?, ?)",
		task.Description, task.Status, timeNow, timeNow, userID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task.ID = int(lastInsertID)
	task.CreatedAt = timeNow
	task.UpdatedAt = timeNow
	task.UserID = userID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	prettyJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}
	w.Write(prettyJSON)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task entity.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Kiểm tra description không được rỗng
	if strings.TrimSpace(task.Description) == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
		return
	}

	// Kiểm tra status hợp lệ
	validStatuses := map[string]bool{
		"todo":        true,
		"in_progress": true,
		"done":        true,
	}
	if _, valid := validStatuses[task.Status]; !valid {
		http.Error(w, "Status must be one of: todo, in_progress, done", http.StatusBadRequest)
		return
	}

	updatedAt := time.Now().Format(time.RFC3339)
	res, err := h.DB.Exec(
		"UPDATE tasks SET description = ?, status = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		task.Description, task.Status, updatedAt, id, userID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Could not determine update result", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Task not found or unauthorized", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Task updated successfully"))
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	res, err := h.DB.Exec("DELETE FROM tasks WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Could not determine delete result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, fmt.Sprintf("Task with ID %d not found or unauthorized", id), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Task with ID %d deleted successfully", id)))
}

func (h *Handler) FilterTasksByStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

	validStatuses := map[string]bool{
		"todo":        true,
		"in_progress": true,
		"done":        true,
	}

	if !validStatuses[status] {
		http.Error(w, "Invalid status filter", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query("SELECT id, description, status, created_at, updated_at, user_id FROM tasks WHERE user_id = ? AND status = ?", userID, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []entity.Task{}
	for rows.Next() {
		var task entity.Task
		if err := rows.Scan(&task.ID, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.UserID); err != nil {
			http.Error(w, "Error scanning task", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	prettyJSON, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		http.Error(w, "Failed to format tasks", http.StatusInternalServerError)
		return
	}
	w.Write(prettyJSON)
}
