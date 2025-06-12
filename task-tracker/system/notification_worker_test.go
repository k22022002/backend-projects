package system_test

import (
	"context"
	"testing"
	"time"

	"task-tracker/system"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNotificationWorkerPool_ProcessesJob(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Mock DB insert
	mock.ExpectExec("INSERT INTO notifications").
		WithArgs(1, "Task Created", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	pool := system.NewNotificationWorkerPool(db, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pool.Start(ctx)

	// Gửi 1 job
	pool.JobQueue <- system.NotificationJob{
		TaskID:  1,
		Message: "Task Created",
	}

	// Chờ worker xử lý
	time.Sleep(100 * time.Millisecond)

	// Shutdown pool
	pool.Stop()

	// Đảm bảo tất cả các câu lệnh SQL đã được thực thi
	assert.NoError(t, mock.ExpectationsWereMet())
}
