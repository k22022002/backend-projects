package component

type Status string

const (
	Todo       Status = "todo"
	InProgress Status = "in-progress"
	Done       Status = "done"
)

var Statuses = map[int]Status{}

func SetStatus(id int, status Status) {
	Statuses[id] = status
}

func GetStatus(id int) Status {
	return Statuses[id]
}
