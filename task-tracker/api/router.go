package api

import (
	"net/http"

	"task-tracker/middleware"
	handler "task-tracker/system"

	"github.com/gorilla/mux"
)

func NewRouter() http.Handler {
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/register", handler.Register).Methods("POST")
	r.HandleFunc("/login", handler.Login).Methods("POST")

	// Protected routes
	s := r.PathPrefix("/tasks").Subrouter()
	s.Use(middleware.JWTMiddleware)
	s.HandleFunc("", handler.GetAllTasks).Methods("GET")
	s.HandleFunc("/filter", handler.FilterTasksByStatus).Methods("GET")
	s.HandleFunc("/{id:[0-9]+}", handler.GetTask).Methods("GET")
	s.HandleFunc("", handler.CreateTask).Methods("POST")
	s.HandleFunc("/{id:[0-9]+}", handler.UpdateTask).Methods("PUT")
	s.HandleFunc("/{id:[0-9]+}", handler.DeleteTask).Methods("DELETE")

	return r
}
