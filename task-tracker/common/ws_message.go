package common

type WSMessage struct {
	Event       string `json:"event"`
	TaskID      int    `json:"task_id,omitempty"`
	Description string `json:"description,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}
