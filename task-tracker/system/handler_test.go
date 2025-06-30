package system

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"task-tracker/cache"
	"task-tracker/common"
	"task-tracker/entity"
	"task-tracker/ws"

	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

	req := httptest.NewRequest("POST", "/register", strings.NewReader("invalid"))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_EmptyUsernameOrPassword(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

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
	h := NewHandler(db, nil)

	req := httptest.NewRequest("POST", "/login", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTask_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Vô hiệu hóa Redis
	cache.RedisClient = nil
	cache.Ctx = nil

	// Vô hiệu hóa WebSocket
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil

	// Vô hiệu hóa Pool (tránh block JobQueue)
	h := &Handler{DB: db, Pool: nil}

	task := entity.Task{
		Description: "New task",
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("INSERT INTO tasks").
		WithArgs(task.Description, task.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, task.Description, resp["description"])
	assert.Equal(t, task.Status, resp["status"])
	assert.Equal(t, float64(1), resp["id"])
	assert.Equal(t, float64(1), resp["user_id"])

	_, createdAtOk := resp["createdAt"].(string)
	_, updatedAtOk := resp["updatedAt"].(string)
	assert.True(t, createdAtOk)
	assert.True(t, updatedAtOk)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestCreateTask_InvalidInput(t *testing.T) {
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer([]byte("invalid-json")))
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	h := &Handler{DB: db}

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
func TestCreateTask_EmptyDescription(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &Handler{DB: db}

	task := entity.Task{
		Description: "   ", // trống sau khi trim
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.CreateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Description is required")
}

func TestCreateTask_InvalidStatus(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &Handler{DB: db}

	task := entity.Task{
		Description: "Valid description",
		Status:      "invalid_status",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
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
	h := &Handler{DB: db}

	h.CreateTask(rec, req)

	// Giả sử handler trả 401 khi userID không có
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateTask_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	h := &Handler{DB: db}

	task := entity.Task{
		Description: "New task",
		Status:      "todo",
	}
	taskJSON, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskJSON))
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
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

	h := &Handler{DB: db}
	req := httptest.NewRequest("GET", "/tasks/filter?status=todo", nil)
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
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
	h := &Handler{DB: db}

	req := httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	rec := httptest.NewRecorder()

	h.GetAllTasks(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
func TestUpdateTask_InvalidInput(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &Handler{DB: db}

	req := httptest.NewRequest("PUT", "/tasks/1", bytes.NewBuffer([]byte("invalid")))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateTask_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &Handler{DB: db}
	cache.RedisClient = nil
	cache.Ctx = nil

	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	body := `{"description":"Updated Task","status":"in_progress"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
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
	h := &Handler{DB: db}

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
	h := &Handler{DB: db}

	body := `{"description":"Updated Task","status":"in_progress"}`
	req := httptest.NewRequest("PUT", "/tasks/abc", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid task ID")
}
func TestUpdateTask_BothFieldsEmpty_ShouldReturnBadRequest(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	h := &Handler{DB: db}

	body := `{"description":"   ","status":"  "}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "At least description or status must be provided")
}

func TestUpdateTask_OnlyDescriptionProvided_ShouldSucceed(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	// Vô hiệu hóa Redis
	cache.RedisClient = nil
	cache.Ctx = nil

	// Vô hiệu hóa WebSocket
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db, Pool: nil}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tasks SET description = ?, status = ?, updated_at = ? WHERE id = ? AND user_id = ?")).
		WithArgs("New description", "", sqlmock.AnyArg(), 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"description":"New description"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task updated successfully")
}
func TestUpdateTask_NotFoundOrUnauthorized(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	// Vô hiệu hóa Redis
	cache.RedisClient = nil
	cache.Ctx = nil

	// Vô hiệu hóa WebSocket
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	body := `{"description":"Updated Task","status":"todo"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
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
	// Vô hiệu hóa Redis
	cache.RedisClient = nil
	cache.Ctx = nil

	// Vô hiệu hóa WebSocket
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	body := `{"description":"Updated Task","status":"done"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	mock.ExpectExec("UPDATE tasks").
		WithArgs("Updated Task", "done", sqlmock.AnyArg(), 1, 1).
		WillReturnError(fmt.Errorf("database error"))

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "database error")
}
func TestUpdateTask_OnlyStatusProvided_ShouldSucceed(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	// Vô hiệu hóa Redis
	cache.RedisClient = nil
	cache.Ctx = nil

	// Vô hiệu hóa WebSocket
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tasks SET description = ?, status = ?, updated_at = ? WHERE id = ? AND user_id = ?")).
		WithArgs("", "done", sqlmock.AnyArg(), 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"status":"done"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task updated successfully")
}
func TestUpdateTask_EmptyDescriptionButValidStatus_ShouldSucceed(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	// Vô hiệu hóa Redis
	cache.RedisClient = nil
	cache.Ctx = nil

	// Vô hiệu hóa WebSocket
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tasks SET description = ?, status = ?, updated_at = ? WHERE id = ? AND user_id = ?")).
		WithArgs("   ", "done", sqlmock.AnyArg(), 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"description":"   ","status":"done"}`
	req := httptest.NewRequest("PUT", "/tasks/1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.UpdateTask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task updated successfully")
}

func TestDeleteTask_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := &Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	// Mock: không tìm thấy task
	mock.ExpectQuery("SELECT status FROM tasks").
		WithArgs(999, 1).
		WillReturnError(sql.ErrNoRows)

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task not found")
}

type mockRedisClient struct{}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(1, nil)
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return redis.NewStringResult("mocked-value", nil)
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}

func (m *mockRedisClient) Decr(ctx context.Context, key string) *redis.IntCmd {
	return redis.NewIntResult(99, nil)
}

func (m *mockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	return redis.NewIntResult(1, nil)
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, nil)
}

func (m *mockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return redis.NewDurationResult(30*time.Second, nil)
}

func TestDeleteTask_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Gán mock Redis client và context
	cache.RedisClient = &mockRedisClient{}
	cache.Ctx = context.Background()

	// Tắt WebSocket Hub để tránh treo
	originalHub := ws.WsHub
	ws.WsHub = nil
	defer func() { ws.WsHub = originalHub }()

	h := &Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM tasks WHERE id = ? AND user_id = ?")).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("todo"))

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks WHERE id = ? AND user_id = ?")).
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Task with ID 1 deleted successfully")

	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestDeleteTask_Unauthorized(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	cache.RedisClient = nil
	cache.Ctx = nil

	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	// Không set userID vào context
	rec := httptest.NewRecorder()

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestDeleteTask_InvalidID(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	cache.RedisClient = nil
	cache.Ctx = nil

	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid task ID")
}

func TestDeleteTask_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	// 🛠️ Gán RedisClient giả để tránh panic
	cache.RedisClient = &mockRedisClient{}
	cache.Ctx = context.Background()

	// Tắt WebSocket để tránh channel block
	originalHub := ws.WsHub
	defer func() { ws.WsHub = originalHub }()
	ws.WsHub = nil
	h := &Handler{DB: db}

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	rec := httptest.NewRecorder()

	// Mock: query lấy status thành công
	mock.ExpectQuery("SELECT status FROM tasks").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("todo"))

	// Mock: lỗi khi delete
	mock.ExpectExec("DELETE FROM tasks").
		WithArgs(1, 1).
		WillReturnError(fmt.Errorf("database error"))

	h.DeleteTask(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "database error")
}

type MockRedisClient struct {
	GetFunc    func(ctx context.Context, key string) *redis.StringCmd
	SetFunc    func(ctx context.Context, key string, value interface{}, exp time.Duration) *redis.StatusCmd
	DelFunc    func(ctx context.Context, keys ...string) *redis.IntCmd
	DecrFunc   func(ctx context.Context, key string) *redis.IntCmd
	IncrFunc   func(ctx context.Context, key string) *redis.IntCmd
	ExpireFunc func(ctx context.Context, key string, exp time.Duration) *redis.BoolCmd
	TTLFunc    func(ctx context.Context, key string) *redis.DurationCmd
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return redis.NewStringResult("", redis.Nil)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, exp time.Duration) *redis.StatusCmd {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, exp)
	}
	return redis.NewStatusResult("OK", nil)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	if m.DelFunc != nil {
		return m.DelFunc(ctx, keys...)
	}
	return redis.NewIntResult(1, nil)
}

func (m *MockRedisClient) Decr(ctx context.Context, key string) *redis.IntCmd {
	if m.DecrFunc != nil {
		return m.DecrFunc(ctx, key)
	}
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	if m.IncrFunc != nil {
		return m.IncrFunc(ctx, key)
	}
	return redis.NewIntResult(1, nil)
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, exp time.Duration) *redis.BoolCmd {
	if m.ExpireFunc != nil {
		return m.ExpireFunc(ctx, key, exp)
	}
	return redis.NewBoolResult(true, nil)
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	if m.TTLFunc != nil {
		return m.TTLFunc(ctx, key)
	}
	return redis.NewDurationResult(5*time.Minute, nil)
}

func TestGetAllTasks_WithPaginationAndSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// ✅ Giả lập Redis cache miss (không có dữ liệu)
	cache.RedisClient = &MockRedisClient{
		GetFunc: func(ctx context.Context, key string) *redis.StringCmd {
			// key có dạng: tasks:user:<userID>:status:<status>
			require.Equal(t, "tasks:user:1:status:", key)
			return redis.NewStringResult("", redis.Nil)
		},
		SetFunc: func(ctx context.Context, key string, value interface{}, exp time.Duration) *redis.StatusCmd {
			// Đảm bảo key được set đúng format
			require.Equal(t, "tasks:user:1:status:", key)
			require.NotEmpty(t, value)
			return redis.NewStatusResult("OK", nil)
		},
	}
	cache.Ctx = context.Background()

	h := &Handler{DB: db}

	userID := 1
	page := 2
	limit := 2
	offset := (page - 1) * limit
	search := "test"

	// ----- Mock count query -----
	countQuery := regexp.QuoteMeta(`
		SELECT COUNT(*) FROM tasks WHERE user_id = ? AND LOWER(description) LIKE ?
	`)
	mock.ExpectQuery(countQuery).
		WithArgs(userID, "%test%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// ----- Mock data query -----
	dataQuery := regexp.QuoteMeta(`
		SELECT id, description, status, created_at, updated_at, user_id
		FROM tasks WHERE user_id = ? AND LOWER(description) LIKE ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`)
	createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt

	mock.ExpectQuery(dataQuery).
		WithArgs(userID, "%test%", limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "description", "status", "created_at", "updated_at", "user_id",
		}).AddRow(1, "Test task", "todo", createdAt, updatedAt, userID))

	// ----- Request -----
	req := httptest.NewRequest("GET", fmt.Sprintf("/tasks?page=%d&limit=%d&search=%s", page, limit, search), nil)
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, userID))
	rec := httptest.NewRecorder()

	// ----- Call handler -----
	h.GetAllTasks(rec, req)

	// ----- Validate response -----
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err, "Invalid JSON: %s", rec.Body.String())

	require.Equal(t, float64(page), resp["page"])
	require.Equal(t, float64(limit), resp["limit"])
	require.Equal(t, float64(5), resp["total"])
	require.Equal(t, float64(3), resp["totalPages"]) // 5 / 2 = 2.5 => ceil = 3
	require.Contains(t, resp, "tasks")

	tasks, ok := resp["tasks"].([]interface{})
	require.True(t, ok)
	require.Len(t, tasks, 1)

	task := tasks[0].(map[string]interface{})
	require.Equal(t, float64(1), task["id"])
	require.Equal(t, "Test task", task["description"])
	require.Equal(t, "todo", task["status"])
	require.Equal(t, float64(userID), task["user_id"])
	require.IsType(t, "", task["createdAt"]) // JSON encode time.Time thành string
	require.NotEmpty(t, task["createdAt"])
}

func TestHandler_GetNotifications(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()

	// ✅ Query trong test phải giống y chang handler
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT n.id, n.task_id, n.message, n.created_at
		FROM notifications n
		JOIN tasks t ON t.id = n.task_id
		WHERE t.user_id = ?
		ORDER BY n.created_at DESC
	`)).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "message", "created_at"}).
			AddRow(1, 1, "Task created", now).
			AddRow(2, 1, "Task updated", now))

	h := &Handler{DB: db}

	// ✅ Đảm bảo đúng key như handler dùng
	req := httptest.NewRequest("GET", "/notifications", nil)
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 42))
	rec := httptest.NewRecorder()

	h.GetNotifications(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var notifications []entity.Notification
	err = json.Unmarshal(rec.Body.Bytes(), &notifications)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Equal(t, "Task created", notifications[0].Message)

	require.NoError(t, mock.ExpectationsWereMet())
}
func TestCreateTaskV2(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := &Handler{DB: db}

	// Vô hiệu hóa Redis, WebSocket
	cache.RedisClient = nil
	cache.Ctx = nil
	ws.WsHub = nil

	// Tạo payload đúng định dạng RFC3339
	dueDate := time.Now().AddDate(0, 0, 7).UTC()
	payload := fmt.Sprintf(`{"description":"New task","status":"todo","due_date":"%s"}`, dueDate.Format(time.RFC3339))

	req := httptest.NewRequest("POST", "/v2/tasks", strings.NewReader(payload))
	req = req.WithContext(context.WithValue(req.Context(), common.ContextUserIDKey, 1))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Do handler truyền due_date dưới dạng string: "YYYY-MM-DD"
	mock.ExpectExec("INSERT INTO tasks").
		WithArgs(
			"New task",
			"todo",
			sqlmock.AnyArg(),             // created_at
			sqlmock.AnyArg(),             // updated_at
			1,                            // user_id
			dueDate.Format("2006-01-02"), // 🟢 chuỗi, không phải time.Time
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	http.HandlerFunc(h.CreateTaskV2).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	// fmt.Println("RESPONSE BODY:", rr.Body.String())

	var task entity.Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	require.NoError(t, err)
	assert.Equal(t, "New task", task.Description)
	assert.Equal(t, "todo", task.Status)
	assert.NotNil(t, task.DueDate)
	assert.Equal(t, dueDate.Format("2006-01-02"), task.DueDate.Format("2006-01-02"))

	require.NoError(t, mock.ExpectationsWereMet())
}
