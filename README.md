# backend-projects
taskcli/
├── main.go
├── entity/
│   └── entity.go         // Định nghĩa EntityID
├── component/
│   ├── description.go    // Component mô tả
│   ├── status.go         // Component trạng thái
│   └── time.go           // Component thời gian
├── system/
│   ├── add.go            // Thêm nhiệm vụ
│   ├── update.go         // Cập nhật nhiệm vụ
│   ├── delete.go         // Xoá nhiệm vụ
│   ├── mark.go           // Đổi trạng thái
│   └── list.go           // Liệt kê nhiệm vụ
├── storage/
│   └── store.go          // Đọc/ghi JSON
🚀 Cách chạy
Cài đặt Go nếu chưa có: https://go.dev/dl

Clone hoặc tải về mã nguồn.

Chạy ứng dụng bằng dòng lệnh:go run main.go <command>
| Lệnh                     | Mô tả                                                         |
| ------------------------ | ------------------------------------------------------------- |
| `add <desc>`             | Thêm task mới với mô tả                                       |
| `list`                   | Hiển thị tất cả các task                                      |
| `list <status>`          | Hiển thị task theo trạng thái (`todo`, `in-progress`, `done`) |
| `update <id> <new_desc>` | Cập nhật mô tả task                                           |
| `delete <id>`            | Xoá task                                                      |
| `mark <id> <status>`     | Đổi trạng thái task                                           |


🧪 Ví dụ sử dụng

go run main.go add "Viết tài liệu dự án"
go run main.go list
go run main.go mark 1 done
go run main.go update 1 "Hoàn tất tài liệu"
go run main.go delete 1

📝 File lưu trữ

Dữ liệu được lưu trữ tự động vào file tasks.json trong cùng thư mục.

📌 Kiến trúc CES

Component: Dữ liệu như Description, Status, Time.

Entity: ID duy nhất (int).

System: Logic thao tác với component (thêm, xoá, sửa…).
