package main

import (
	"log"
	"net/http"

	"task-tracker/api"
	"task-tracker/storage/sqlite"
)

func main() {
	db := sqlite.InitDB()
	defer db.Close()

	r := api.NewRouter()
	log.Println("Server is running at http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
