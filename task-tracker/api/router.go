package api

import (
	"task-tracker/system"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/tasks", system.CreateTask).Methods("POST")
	r.HandleFunc("/tasks", system.GetAllTasks).Methods("GET")
	r.HandleFunc("/tasks/{id:[0-9]+}", system.GetTask).Methods("GET")
	r.HandleFunc("/tasks/{id:[0-9]+}", system.UpdateTask).Methods("PUT")
	r.HandleFunc("/tasks/{id:[0-9]+}", system.DeleteTask).Methods("DELETE")
	return r
}
