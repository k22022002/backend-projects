package api

import (
	"log"
	"net/http"

	"task-tracker/middleware"
	"task-tracker/storage/sqlite"
	handler "task-tracker/system"

	"github.com/gorilla/mux"
)

func NewRouter() http.Handler {
	r := mux.NewRouter()
	db, err := sqlite.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	h := handler.NewHandler(db)
	// Public routes
	r.HandleFunc("/register", h.Register).Methods("POST")
	r.HandleFunc("/login", h.Login).Methods("POST")

	// Protected routes
	s := r.PathPrefix("/tasks").Subrouter()
	s.Use(middleware.JWTMiddleware)
	s.HandleFunc("", h.GetAllTasks).Methods("GET")
	s.HandleFunc("/filter", h.FilterTasksByStatus).Methods("GET")
	s.HandleFunc("/{id:[0-9]+}", h.GetTask).Methods("GET")
	s.HandleFunc("", h.CreateTask).Methods("POST")
	s.HandleFunc("/{id:[0-9]+}", h.UpdateTask).Methods("PUT")
	s.HandleFunc("/{id:[0-9]+}", h.DeleteTask).Methods("DELETE")

	return r
}
