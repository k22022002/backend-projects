package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"task-tracker/cache"
	"task-tracker/common"
	"task-tracker/component"
	"task-tracker/storage"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setup Redis mock
func setupMockRedis() (redismock.ClientMock, func()) {
	db, mock := redismock.NewClientMock()
	cache.RedisClient = db
	return mock, func() {
		db.Close()
	}
}

// mock handler with dummy DB (nếu cần)
func mockHandler() *Handler {
	return &Handler{
		// gán các service mock khác nếu cần
	}
}

func TestGetTask_CacheHit(t *testing.T) {
	mock, cleanup := setupMockRedis()
	defer cleanup()

	handler := mockHandler()

	task := storage.Task{
		ID:          1,
		Description: "Cached Task",
		Status:      component.Status("todo"),
		CreatedAt:   "2025-06-13T00:00:00Z",
		UpdatedAt:   "2025-06-13T00:00:00Z",
	}
	taskJSON, _ := json.Marshal(task)

	// 🔁 Expect Redis hit
	mock.ExpectGet("task:1").SetVal(string(taskJSON))

	req := httptest.NewRequest("GET", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	// ⚠️ Mock context with userID (nếu handler dùng context để lấy userID)
	ctx := context.WithValue(req.Context(), common.ContextUserIDKey, 1)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetTask(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Cached Task")
}

func TestGetTask_CacheMiss(t *testing.T) {
	mock, cleanup := setupMockRedis()
	defer cleanup()

	handler := mockHandler()
	task := storage.Task{
		ID:          1,
		Description: "Cached Task",
		Status:      component.Status("todo"),
		CreatedAt:   "2025-06-13T00:00:00Z",
		UpdatedAt:   "2025-06-13T00:00:00Z",
	}
	taskJSON, _ := json.Marshal(task)
	// 🔁 Redis miss
	mock.ExpectGet("task:2").RedisNil()

	// 🔁 Simulate saving to Redis after DB query
	mock.ExpectSet("task:2", string(taskJSON), 5*time.Minute).SetVal("OK")

	// giả lập DB response (bạn phải gán taskService.GetByID nếu muốn)
	// Tạm thời giả lập phản hồi bằng cách sửa handler hoặc tách logic

	req := httptest.NewRequest("GET", "/tasks/2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "2"})
	w := httptest.NewRecorder()

	// bạn có thể thêm middleware/fake DB vào đây
	handler.GetTask(w, req)

	// vì chưa có DB, ta chỉ test Redis logic (nếu chưa mock DB)
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestCacheInvalidation_OnDelete(t *testing.T) {
	// 1. Redis mock
	redisClient, redisMock := redismock.NewClientMock()
	cache.RedisClient = redisClient
	defer redisClient.Close()

	// 2. SQL mock
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 3. Handler
	handler := &Handler{DB: db}

	// 4. Dữ liệu mock
	taskID := 1
	userID := 1

	// 5. Mock DB SELECT status
	sqlMock.ExpectQuery("SELECT status FROM tasks").
		WithArgs(taskID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("todo"))

	// 6. Mock Redis xóa cache
	redisMock.ExpectDel("task:1").SetVal(1)
	redisMock.ExpectDel("tasks:user:1:status:todo").SetVal(1)

	// 7. Mock DB DELETE
	sqlMock.ExpectExec("DELETE FROM tasks WHERE id = \\? AND user_id = \\?").
		WithArgs(taskID, userID).
		WillReturnResult(sqlmock.NewResult(1, 1)) // 1 row affected

	// 8. Tạo HTTP request
	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx := context.WithValue(req.Context(), common.ContextUserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// 9. Gọi handler
	handler.DeleteTask(w, req)

	// 10. Kiểm tra kết quả
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, redisMock.ExpectationsWereMet())
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
