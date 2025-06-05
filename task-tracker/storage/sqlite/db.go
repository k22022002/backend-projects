package sqlite

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "data.db")
	if err != nil {
		log.Fatal(err)
	}
	if err := InitSchema(db); err != nil {
		log.Fatal(err)
	}
	return db, nil
}

var DB *sql.DB

// InitSchema nhận db để tạo bảng
func InitSchema(db *sql.DB) error {
	createUsersTable := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);`

	createTasksTable := `CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		created_at TEXT,
		updated_at TEXT,
		user_id INTEGER,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	if _, err := db.Exec(createUsersTable); err != nil {
		return err
	}
	if _, err := db.Exec(createTasksTable); err != nil {
		return err
	}
	return nil
}
