package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task-tracker/entity"
	"task-tracker/storage/sqlite"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("your-secret-key")

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func Register(w http.ResponseWriter, r *http.Request) {
	var user entity.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user.Username == "" || user.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hash)

	db := sqlite.InitDB()
	_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user.Username, user.Password)
	if err != nil {
		http.Error(w, "Username may already exist", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Registration successful"})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var creds entity.User
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	db := sqlite.InitDB()
	var user entity.User
	err = db.QueryRow("SELECT id, password FROM users WHERE username = ?", creds.Username).
		Scan(&user.ID, &user.Password)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"token":   tokenString,
	})
}

func GetAllTasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)
	db := sqlite.InitDB()

	rows, err := db.Query("SELECT id, description, status, created_at, updated_at, user_id FROM tasks WHERE user_id = ?", userID)
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

func GetTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	db := sqlite.InitDB()

	var task entity.Task
	err := db.QueryRow("SELECT id, description, status, created_at, updated_at, user_id FROM tasks WHERE id = ? AND user_id = ?", id, userID).
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

func CreateTask(w http.ResponseWriter, r *http.Request) {
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

	db := sqlite.InitDB()
	defer db.Close()

	timeNow := time.Now().Format(time.RFC3339)

	res, err := db.Exec(
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

func UpdateTask(w http.ResponseWriter, r *http.Request) {
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

	db := sqlite.InitDB()
	defer db.Close()

	updatedAt := time.Now().Format(time.RFC3339)
	res, err := db.Exec(
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

func DeleteTask(w http.ResponseWriter, r *http.Request) {
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

	db := sqlite.InitDB()
	defer db.Close()

	res, err := db.Exec("DELETE FROM tasks WHERE id = ? AND user_id = ?", id, userID)
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

func FilterTasksByStatus(w http.ResponseWriter, r *http.Request) {
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

	db := sqlite.InitDB()
	defer db.Close()

	rows, err := db.Query("SELECT id, description, status, created_at, updated_at, user_id FROM tasks WHERE user_id = ? AND status = ?", userID, status)
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
