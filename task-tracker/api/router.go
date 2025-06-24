package api

import (
	"context"
	"log"
	"net/http"

	_ "task-tracker/docs" // Import the generated docs package
	"task-tracker/middleware"
	"task-tracker/storage/sqlite"
	handler "task-tracker/system"
	"task-tracker/ws"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter() http.Handler {
	//  Khởi tạo DB và pool
	db, err := sqlite.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	//  Worker pool xử lý noti
	pool := handler.NewNotificationWorkerPool(db, 5)
	pool.Start(ctx)

	//  Khởi tạo handler chính
	h := handler.NewHandler(db, pool)

	//  Khởi tạo hub và chạy nó
	ws.WsHub = ws.NewHub()
	go ws.WsHub.Run()

	//  Router setup
	r := mux.NewRouter()

	//  Public routes
	r.HandleFunc("/ws", ws.HandleWS)
	r.HandleFunc("/register", h.Register).Methods("POST")
	r.HandleFunc("/login", h.Login).Methods("POST")
	r.HandleFunc("/health", h.Health).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	//  Protected routes
	s := r.PathPrefix("/tasks").Subrouter()
	s.Use(middleware.JWTMiddleware)
	s.HandleFunc("", h.GetAllTasks).Methods("GET")
	s.HandleFunc("/filter", h.FilterTasksByStatus).Methods("GET")
	s.HandleFunc("/{id:[0-9]+}", h.GetTask).Methods("GET")
	s.HandleFunc("", h.CreateTask).Methods("POST")
	s.HandleFunc("/{id:[0-9]+}", h.UpdateTask).Methods("PUT")
	s.HandleFunc("/{id:[0-9]+}", h.DeleteTask).Methods("DELETE")
	s.HandleFunc("/notifications", h.GetNotifications).Methods("GET")

	return r
}
