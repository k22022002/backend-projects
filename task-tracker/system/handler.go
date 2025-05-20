package system

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"task-tracker/component"
	"task-tracker/storage"
	"time"

	"github.com/gorilla/mux"
)

var mu sync.Mutex

type Task struct {
	ID          int              `json:"id"`
	Description string           `json:"description"`
	Status      component.Status `json:"status"`
	CreatedAt   string           `json:"createdAt"`
	UpdatedAt   string           `json:"updatedAt"`
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	id := len(component.Descriptions) + 1
	component.SetDescription(id, t.Description)
	component.SetStatus(id, component.Todo)
	now := time.Now()
	component.SetTime(id, now, now)
	storage.SaveTasks()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

func GetAllTasks(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	statusFilter := r.URL.Query().Get("status") // Lấy tham số lọc trạng thái từ URL

	var result []Task
	for id, desc := range component.Descriptions {
		status := component.GetStatus(id)

		// Nếu có lọc trạng thái mà task không khớp thì bỏ qua
		if statusFilter != "" && string(status) != statusFilter {
			continue
		}

		time := component.GetTime(id)
		result = append(result, Task{
			ID:          id,
			Description: desc,
			Status:      status,
			CreatedAt:   time.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   time.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ") // Định dạng JSON đẹp cho Postman
	if err := encoder.Encode(result); err != nil {
		http.Error(w, "Failed to encode tasks", http.StatusInternalServerError)
	}
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if desc, ok := component.Descriptions[id]; ok {
		time := component.GetTime(id)
		json.NewEncoder(w).Encode(Task{
			ID:          id,
			Description: desc,
			Status:      component.GetStatus(id),
			CreatedAt:   time.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   time.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
		return
	}
	http.Error(w, "Task not found", http.StatusNotFound)
}

func UpdateTask(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if _, ok := component.Descriptions[id]; !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	component.SetDescription(id, t.Description)
	component.SetStatus(id, t.Status)
	component.UpdateTime(id)
	storage.SaveTasks()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task updated"})
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if _, ok := component.Descriptions[id]; !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	delete(component.Descriptions, id)
	delete(component.Statuses, id)
	delete(component.Times, id)
	storage.SaveTasks()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task deleted"})
}
