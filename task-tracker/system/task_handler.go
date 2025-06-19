package system

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task-tracker/cache"
	"task-tracker/common"
	"task-tracker/entity"
	"task-tracker/ws"

	"github.com/gorilla/mux"
)

func (h *Handler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	// Xác thực user
	userIDVal := r.Context().Value("userID")
	userID, ok := userIDVal.(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Lấy query params
	query := r.URL.Query()
	status := query.Get("status") // nếu có filter
	cacheKey := fmt.Sprintf("tasks:user:%d:status:%s", userID, status)

	// 🧠 Try Redis cache
	if cached, err := cache.Get(cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	search := query.Get("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Xây dựng query
	whereClause := "WHERE user_id = ?"
	args := []interface{}{userID}

	if search != "" {
		whereClause += " AND LOWER(description) LIKE ?"
		args = append(args, "%"+strings.ToLower(search)+"%")
	}

	// Đếm tổng số task
	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	var total int
	if err := h.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Lấy task theo phân trang
	dataQuery := fmt.Sprintf(`
		SELECT id, description, status, created_at, updated_at, user_id
		FROM tasks %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, whereClause)
	args = append(args, limit, offset)

	rows, err := h.DB.Query(dataQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Lấy kết quả
	tasks := []entity.Task{}
	for rows.Next() {
		var task entity.Task
		if err := rows.Scan(&task.ID, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	// Tính tổng số trang
	totalPages := (total + limit - 1) / limit

	// Response với metadata
	response := map[string]interface{}{
		"tasks":      tasks,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}
	// Set Redis cache
	jsonResp, _ := json.Marshal(response)
	cache.Set(cacheKey, string(jsonResp), 5*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value("userID")
	userID, ok := userIDVal.(int)
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
	cacheKey := fmt.Sprintf("task:%d", id)

	if cached, err := cache.Get(cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}
	var task entity.Task
	err = h.DB.QueryRow(`
		SELECT id, description, status, created_at, updated_at, user_id
		FROM tasks
		WHERE id = ? AND user_id = ?`,
		id, userID).
		Scan(&task.ID, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt, &task.UserID)

	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	jsonResp, _ := json.Marshal(task)
	cache.Set(cacheKey, string(jsonResp), 5*time.Minute)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}
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
	if ws.WsHub != nil {
		ws.WsHub.Broadcast <- common.NotificationJob{
			TaskID:  userID,
			Message: fmt.Sprintf(`{"event":"task_created", "task_id":%d, "description":"%s"}`, task.ID, task.Description),
		}
	}
	if h.Pool != nil && h.Pool.JobQueue != nil {
		h.Pool.JobQueue <- NotificationJob{
			TaskID:  task.ID,
			Message: fmt.Sprintf("New task created: %s", task.Description),
		}
	}
	cache.Delete("tasks:user:*") // xóa toàn bộ danh sách (có thể tinh chỉnh theo user cụ thể)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	cache.Delete("task:" + strconv.Itoa(id))
	cache.Delete("tasks:user:*")
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task entity.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// ít nhất 1 trong 2 phải có
	if strings.TrimSpace(task.Description) == "" && strings.TrimSpace(task.Status) == "" {
		http.Error(w, "At least description or status must be provided", http.StatusBadRequest)
		return
	}

	// Kiểm tra status hợp lệ
	if task.Status != "" {
		validStatuses := map[string]bool{
			"todo":        true,
			"in_progress": true,
			"done":        true,
		}
		if _, valid := validStatuses[task.Status]; !valid {
			http.Error(w, "Status must be one of: todo, in_progress, done", http.StatusBadRequest)
			return
		}
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
	if ws.WsHub != nil {
		ws.WsHub.Broadcast <- common.NotificationJob{
			TaskID:  userID,
			Message: fmt.Sprintf(`{"event":"task_updated", "task_id":%d, "description":"%s"}`, id, task.Description),
		}
	}

	if task.Status == "done" && h.Pool != nil && h.Pool.JobQueue != nil {
		h.Pool.JobQueue <- NotificationJob{
			TaskID:  id,
			Message: fmt.Sprintf("Task '%s' marked as done.", task.Description),
		}
	}
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

	// Truy vấn status để xóa cache theo key đúng
	var status string
	err = h.DB.QueryRow("SELECT status FROM tasks WHERE id = ? AND user_id = ?", id, userID).Scan(&status)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Xóa cache
	cache.RedisClient.Del(context.Background(), fmt.Sprintf("task:%d", id))
	cache.RedisClient.Del(context.Background(), fmt.Sprintf("tasks:user:%d:status:%s", userID, status))

	// Xóa DB
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
	if ws.WsHub != nil {
		ws.WsHub.Broadcast <- common.NotificationJob{
			TaskID:  userID,
			Message: fmt.Sprintf(`{"event":"task-deleted", "task_id":%d, "description":"Task deleted"}`, id),
		}
	}

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
