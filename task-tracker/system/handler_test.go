package system_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"task-tracker/entity"
	"task-tracker/system"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// func setupTestDB() *sql.DB {
// 	db, err := sqlite.InitDB()
// 	if err != nil {
// 		panic(err)
// 	}
// 	return db
// }

func TestRegister_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	user := entity.User{Username: "testuser", Password: "password123"}
	body, _ := json.Marshal(user)

	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "Registration successful")
}

func TestRegister_InvalidBody(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	req := httptest.NewRequest("POST", "/register", strings.NewReader("invalid"))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_EmptyUsernameOrPassword(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	user := entity.User{}
	body, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
func TestRegister_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	user := entity.User{Username: "testuser", Password: "password123"}
	body, _ := json.Marshal(user)

	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("DB error"))

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "DB error")
}

func TestLogin_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	password := "password123"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	mock.ExpectQuery("SELECT id, password FROM users").
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password"}).
			AddRow(1, string(hashed)))

	user := entity.User{Username: "testuser", Password: password}
	body, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "token")
}

func TestLogin_InvalidPassword(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)

	mock.ExpectQuery("SELECT id, password FROM users").
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password"}).
			AddRow(1, string(hashed)))

	user := entity.User{Username: "testuser", Password: "wrong-password"}
	body, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid username or password")
}

func TestLogin_InvalidUser(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	mock.ExpectQuery("SELECT id, password FROM users").
		WithArgs("unknown").
		WillReturnError(sql.ErrNoRows)

	user := entity.User{Username: "unknown", Password: "password"}
	body, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
func TestLogin_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	mock.ExpectQuery("SELECT id, password FROM users").
		WithArgs("testuser").
		WillReturnError(errors.New("some DB error"))

	user := entity.User{Username: "testuser", Password: "any"}
	body, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "DB query error")
}
func TestLogin_InvalidBody(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := system.NewHandler(db)

	req := httptest.NewRequest("POST", "/login", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTask_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	h := &system.Handler{DB: db}

	task := entity.Task{
		Description: "New task",
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("INSERT INTO tasks").
		WithArgs(task.Description, task.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	// Parse response body thành map để test các trường cơ bản
	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, task.Description, resp["description"])
	assert.Equal(t, task.Status, resp["status"])
	assert.Equal(t, float64(1), resp["id"])      // json.Unmarshal số int trả về float64
	assert.Equal(t, float64(1), resp["user_id"]) // json.Unmarshal số int trả về float64

	// Có thể thêm assert kiểm tra createdAt, updatedAt tồn tại
	_, createdAtOk := resp["createdAt"].(string)
	_, updatedAtOk := resp["updatedAt"].(string)
	assert.True(t, createdAtOk)
	assert.True(t, updatedAtOk)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestCreateTask_InvalidInput(t *testing.T) {
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer([]byte("invalid-json")))
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	h := &system.Handler{DB: db}

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
func TestCreateTask_EmptyDescription(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	task := entity.Task{
		Description: "   ", // trống sau khi trim
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Description is required")
}

func TestCreateTask_InvalidStatus(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	task := entity.Task{
		Description: "Valid description",
		Status:      "invalid_status",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Status must be one of:")
}

func TestCreateTask_NoUserIDInContext(t *testing.T) {
	task := entity.Task{
		Description: "New task",
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	// Không gán userID vào context
	rec := httptest.NewRecorder()
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	h := &system.Handler{DB: db}

	h.CreateTask(rec, req)

	// Giả sử handler trả 401 khi userID không có
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateTask_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	h := &system.Handler{DB: db}

	task := entity.Task{
		Description: "New task",
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("INSERT INTO tasks").
		WithArgs(task.Description, task.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnError(fmt.Errorf("db error"))

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestFilterTasksByStatus_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := &system.Handler{DB: db}
	req := httptest.NewRequest("GET", "/tasks/filter?status=todo", nil)
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	rows := sqlmock.NewRows([]string{"id", "description", "status", "created_at", "updated_at", "user_id"}).
		AddRow(1, "Do laundry", "todo", time.Now(), time.Now(), 1)

	mock.ExpectQuery("SELECT (.+) FROM tasks").
		WithArgs(1, "todo").
		WillReturnRows(rows)

	h.FilterTasksByStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Do laundry")
}
func TestAccess_WithInvalidJWT(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	rec := httptest.NewRecorder()

	h.GetAllTasks(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
func TestUpdateTask_InvalidInput(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("PUT", "/tasks/1", bytes.NewBuffer([]byte("invalid")))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateTask_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"Updated Task","status":"in_progress"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("UPDATE tasks").
		WithArgs("Updated Task", "in_progress", sqlmock.AnyArg(), 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task updated successfully")
}
func TestUpdateTask_Unauthorized(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"Updated Task","status":"in_progress"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	// Không set userID trong context
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}
func TestUpdateTask_InvalidTaskID(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"Updated Task","status":"in_progress"}`
	req := httptest.NewRequest("PUT", "/tasks/abc", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid task ID")
}
func TestUpdateTask_EmptyDescription(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"   ","status":"todo"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Description is required")
}
func TestUpdateTask_InvalidStatus(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"Some Task","status":"invalid_status"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Status must be one of: todo, in_progress, done")
}
func TestUpdateTask_NotFoundOrUnauthorized(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"Updated Task","status":"todo"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("UPDATE tasks").
		WithArgs("Updated Task", "todo", sqlmock.AnyArg(), 1, 1).
		WillReturnResult(sqlmock.NewResult(0, 0)) // affected rows = 0

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task not found or unauthorized")
}
func TestUpdateTask_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	body := `{"description":"Updated Task","status":"done"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("UPDATE tasks").
		WithArgs("Updated Task", "done", sqlmock.AnyArg(), 1, 1).
		WillReturnError(fmt.Errorf("database error"))

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "database error")
}

func TestDeleteTask_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("DELETE FROM tasks").
		WithArgs(999, 1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteTask_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("DELETE FROM tasks").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "deleted successfully")
}
func TestDeleteTask_Unauthorized(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	// Không gán userID vào context
	rec := httptest.NewRecorder()

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestDeleteTask_InvalidID(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid task ID")
}

func TestDeleteTask_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &system.Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), "userID", 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("DELETE FROM tasks").
		WithArgs(1, 1).
		WillReturnError(fmt.Errorf("database error"))

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "database error")
}
func TestGetAllTasks_WithPaginationAndSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open mock sql db: %s", err)
	}
	defer db.Close()

	h := &system.Handler{DB: db}

	// ----- Test parameters -----
	userID := 1
	page := 2
	limit := 2
	offset := (page - 1) * limit
	search := "test"

	// ----- Mock count query -----
	countQuery := `
		SELECT COUNT\(\*\)
		FROM tasks
		WHERE user_id = \?
		AND LOWER\(description\) LIKE \?
	`

	mock.ExpectQuery(countQuery).
		WithArgs(userID, "%"+strings.ToLower(search)+"%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// ----- Mock data query -----
	dataQuery := `
		SELECT id, description, status, created_at, updated_at, user_id
		FROM tasks
		WHERE user_id = \?
		AND LOWER\(description\) LIKE \?
		ORDER BY created_at DESC
		LIMIT \? OFFSET \?
	`

	mock.ExpectQuery(dataQuery).
		WithArgs(userID, "%"+strings.ToLower(search)+"%", limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "description", "status", "created_at", "updated_at", "user_id",
		}).AddRow(1, "Test task", "pending", "2023-01-01", "2023-01-01", userID))

	// ----- Create HTTP request -----
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks?page=%d&limit=%d&search=%s", page, limit, search), nil)
	// Inject context with userID
	ctx := context.WithValue(req.Context(), "userID", userID)
	req = req.WithContext(ctx)

	// ----- Recorder -----
	rec := httptest.NewRecorder()

	// ----- Call handler -----
	h.GetAllTasks(rec, req)

	// ----- Validate -----
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if int(resp["page"].(float64)) != page {
		t.Errorf("Expected page %d, got %v", page, resp["page"])
	}
	if int(resp["limit"].(float64)) != limit {
		t.Errorf("Expected limit %d, got %v", limit, resp["limit"])
	}
	if int(resp["total"].(float64)) != 5 {
		t.Errorf("Expected total 5, got %v", resp["total"])
	}
}
