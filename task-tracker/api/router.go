package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"task-tracker/cache"
	_ "task-tracker/docs" // Import the generated docs package
	"task-tracker/middleware"
	"task-tracker/storage/sqlite"
	"task-tracker/system"
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
	rateLimiter := middleware.NewRateLimiter(cache.RedisClient, 100, time.Hour)
	s.Use(rateLimiter.Middleware)
	s.Use(middleware.JWTMiddleware)
	s.HandleFunc("", h.GetAllTasks).Methods("GET")
	s.HandleFunc("/filter", h.FilterTasksByStatus).Methods("GET")
	s.HandleFunc("/{id:[0-9]+}", h.GetTask).Methods("GET")
	s.HandleFunc("", h.CreateTask).Methods("POST")
	s.HandleFunc("/{id:[0-9]+}", h.UpdateTask).Methods("PUT")
	s.HandleFunc("/{id:[0-9]+}", h.DeleteTask).Methods("DELETE")
	s.HandleFunc("/notifications", h.GetNotifications).Methods("GET")
	r.Handle("/rate-limit", middleware.JWTMiddleware(system.RateLimitStatusHandler(cache.RedisClient))).Methods("GET")
	v2 := r.PathPrefix("/v2").Subrouter()
	v2.Use(middleware.JWTMiddleware)
	v2.HandleFunc("/tasks", h.CreateTaskV2).Methods("POST")
	v2.HandleFunc("/tasks", h.GetAllTasksV2).Methods("GET")
	return r
}
