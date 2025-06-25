package common

type NotificationJob struct {
	TaskID  int
	Message string
}
type contextKey string

const ContextUserIDKey contextKey = "user_id"
