package main

import (
	"log"
	"net/http"
	"task-tracker/api"
	"task-tracker/storage/sqlite"
)

func main() {
	sqlite.InitDB()

	r := api.NewRouter()
	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
