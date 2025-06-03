package sqlite

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() *sql.DB {
	var err error
	DB, err = sql.Open("sqlite", "data.db")
	if err != nil {
		log.Fatal(err)
	}

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

	_, err = DB.Exec(createUsersTable)
	if err != nil {
		log.Fatal(err)
	}
	_, err = DB.Exec(createTasksTable)
	if err != nil {
		log.Fatal(err)
	}
	return DB
}
