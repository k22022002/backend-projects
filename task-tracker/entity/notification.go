package entity

type Notification struct {
	ID        int    `json:"id"`
	TaskID    int    `json:"task_id"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}
