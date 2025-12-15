package utils

import (
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	InfoLog  *log.Logger
	ErrorLog *log.Logger
)

func InitLogger() {
	// Create logs folder if not exists
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}

	InfoLog = log.New(&lumberjack.Logger{
		Filename:   "logs/info.log",
		MaxSize:    5, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}, "INFO: ", log.Ldate|log.Ltime)

	ErrorLog = log.New(&lumberjack.Logger{
		Filename:   "logs/error.log",
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     60,
		Compress:   true,
	}, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
