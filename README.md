# 📝 Task Management API (Go - net/http)

Đây là một RESTful API giúp bạn quản lý các công việc (task) như tạo mới, xem danh sách, cập nhật, xóa và lọc theo trạng thái.

## 🚀 Khởi chạy server

```bash
go run main.go
```

Server sẽ chạy tại `http://localhost:8080`

## 📚 Danh sách Endpoint

### 1. **Tạo task mới**

- **POST** `/tasks`
- **Request Body (JSON):**
```json
{
  "description": "Viết báo cáo môn học"
}
```
- **Response:**
```json
{
  "message": "Task added successfully"
}
```

### 2. **Lấy danh sách tất cả task**

- **GET** `/tasks`
- **Response:**
```json
[
  {
    "id": 1,
    "description": "Viết báo cáo môn học",
    "status": "todo",
    "createdAt": "2025-05-16 09:33:46",
    "updatedAt": "2025-05-16 09:33:46"
  }
]
```

### 3. **Lọc task theo trạng thái**

- **GET** `/tasks?status=todo`
- Hỗ trợ các trạng thái: `todo`, `in-progress`, `done`
- **Response:**
```json
[
  {
    "id": 1,
    "description": "Task A",
    "status": "todo",
    "createdAt": "2025-05-16 09:33:46",
    "updatedAt": "2025-05-16 09:33:46"
  }
]
```

### 4. **Xem chi tiết task theo ID**

- **GET** `/tasks/{id}`
- **Response:**
```json
{
  "id": 2,
  "description": "Task B",
  "status": "in-progress",
  "createdAt": "2025-05-16 09:33:50",
  "updatedAt": "2025-05-16 10:00:00"
}
```

### 5. **Cập nhật task**

- **PUT** `/tasks/{id}`
- **Request Body (JSON):**
```json
{
  "description": "Cập nhật nội dung task",
  "status": "done"
}
```
- **Response:**
```json
{
  "message": "Task updated successfully"
}
```

### 6. **Xóa task**

- **DELETE** `/tasks/{id}`
- **Response:**
```json
{
  "message": "Task deleted successfully"
}
```

## ⚠️ Mã lỗi HTTP

| Mã lỗi | Ý nghĩa                         |
|--------|---------------------------------|
| 400    | Request sai định dạng dữ liệu   |
| 404    | Không tìm thấy task             |
| 500    | Lỗi máy chủ khi xử lý           |

## 🔒 Concurrency

API đã xử lý an toàn concurrent read/write vào file bằng `sync.RWMutex`.

## 📂 Lưu trữ dữ liệu

Tất cả dữ liệu task được lưu trong file `data/tasks.json` dưới dạng mảng JSON.

## 🧪 Test với Postman

1. Mở Postman.
2. Gửi các request như mô tả ở trên.
3. Đảm bảo server Go đang chạy trước khi test.