package component

import "time"

type TimeComponent struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var Times = map[int]TimeComponent{}

// Thiết lập CreatedAt & UpdatedAt khi tạo mới
func SetTime(id int, created, updated time.Time) {
	Times[id] = TimeComponent{
		CreatedAt: created,
		UpdatedAt: updated,
	}
}

// Cập nhật UpdatedAt khi chỉnh sửa
func UpdateTime(id int) {
	if t, ok := Times[id]; ok {
		t.UpdatedAt = time.Now()
		Times[id] = t
	} else {
		now := time.Now()
		Times[id] = TimeComponent{
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
}

func GetTime(id int) TimeComponent {
	if t, ok := Times[id]; ok {
		return t
	}
	now := time.Now()
	return TimeComponent{
		CreatedAt: now,
		UpdatedAt: now,
	}
}
