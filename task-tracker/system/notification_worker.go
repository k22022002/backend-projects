package system

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"task-tracker/entity"
	"time"
)

type NotificationJob struct {
	TaskID  int
	Message string
}

type NotificationWorkerPool struct {
	DB        *sql.DB
	JobQueue  chan NotificationJob
	NumWorker int
	wg        sync.WaitGroup
}

func NewNotificationWorkerPool(db *sql.DB, numWorker int) *NotificationWorkerPool {
	return &NotificationWorkerPool{
		DB:        db,
		JobQueue:  make(chan NotificationJob, 100), // buffered channel
		NumWorker: numWorker,
	}
}

// 🟢 Chính là phương thức này bạn đang thiếu
func (p *NotificationWorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.NumWorker; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Gọi khi muốn shutdown
func (p *NotificationWorkerPool) Stop() {
	close(p.JobQueue)
	p.wg.Wait()
}

// Xử lý 1 worker
func (p *NotificationWorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		case job, ok := <-p.JobQueue:
			if !ok {
				log.Printf("Worker %d channel closed", id)
				return
			}
			err := p.insertNotification(job)
			if err != nil {
				log.Printf("Worker %d error: %v", id, err)
			}
		}
	}
}

func (p *NotificationWorkerPool) insertNotification(job NotificationJob) error {
	_, err := p.DB.Exec("INSERT INTO notifications (task_id, message, created_at) VALUES (?, ?, ?)",
		job.TaskID, job.Message, time.Now().Format(time.RFC3339))
	return err
}

func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.DB.Query(`
		SELECT n.id, n.task_id, n.message, n.created_at
		FROM notifications n
		JOIN tasks t ON t.id = n.task_id
		WHERE t.user_id = ?
		ORDER BY n.created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, "Error querying notifications", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notifications []entity.Notification
	for rows.Next() {
		var n entity.Notification
		if err := rows.Scan(&n.ID, &n.TaskID, &n.Message, &n.CreatedAt); err != nil {
			http.Error(w, "Error reading notifications", http.StatusInternalServerError)
			return
		}
		notifications = append(notifications, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}
