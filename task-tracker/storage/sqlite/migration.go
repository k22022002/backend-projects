package sqlite

import (
	"database/sql"
	"log"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func MigrateDB(dsn string) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf(" Không thể mở database: %v", err)
	}
	defer db.Close()

	goose.SetLogger(log.Default())

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf(" Migration thất bại: %v", err)
	}

	log.Println(" Migration đã chạy thành công.")
}
