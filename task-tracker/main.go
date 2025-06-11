package main

import (
	"log"
	"net/http"

	"task-tracker/api"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	r := api.NewRouter()
	log.Println("Server is running at http://localhost:8080")
	http.ListenAndServe(":8080", r)
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
