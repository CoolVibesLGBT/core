package helpers

import (
	"log"
	"os"
)

var (
	debugMode = true
	logger    = log.New(os.Stdout, "", log.LstdFlags)
)

// SetDebugMode açık/kapalı yapar
func SetDebugMode(enabled bool) {
	debugMode = enabled
}

// Debug fonksiyonu sadece debugMode açıkken yazdırır
func Debug(format string, args ...interface{}) {
	if debugMode {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info fonksiyonu her zaman yazdırır
func Info(format string, args ...interface{}) {
	logger.Printf("[INFO] "+format, args...)
}

// Error fonksiyonu her zaman yazdırır
func Error(format string, args ...interface{}) {
	logger.Printf("[ERROR] "+format, args...)
}

func Println(format string, args ...interface{}) {
	if debugMode {
		logger.Printf("[DEBUG] "+format, args...)
	}
}
