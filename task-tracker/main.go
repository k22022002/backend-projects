package main

import (
	"log"
	"net/http"
	"task-tracker/api"
	"task-tracker/cache"
	"task-tracker/storage/sqlite"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// @title Task Tracker API
// @version 1.0
// @description REST API for task tracking with JWT authentication
// @termsOfService http://swagger.io/terms/

// @contact.name G G Dev Team
// @contact.email contact@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	sqlite.MigrateDB("file:data.db?cache=shared&mode=rwc")
	// ✅ Khởi tạo Redis trước khi chạy HTTP server
	if err := cache.InitRedis("redis:6379"); err != nil {
		log.Fatalf("Không thể kết nối Redis: %v", err)
	}

	// ✅ Khởi tạo router
	r := api.NewRouter()

	log.Println("🚀 Server is running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Lỗi khi chạy server: %v", err)
	}
}

func InitLogger() *zap.Logger {
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
	})

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.InfoLevel,
	)

	return zap.New(core)
}
