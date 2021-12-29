package logger

import (
	"fmt"
	"log"

	"github.com/goft-cloud/go-xxl-job-client/v2/option"
)

// Info log
func Info(msg string) {
	log.Println("[XXL-INFO]", msg)
}

// Infof log
func Infof(tpl string, args ...interface{}) {
	log.Println("[XXL-INFO]", fmt.Sprintf(tpl, args...))
}

// Error log
func Error(msg string) {
	log.Println("[XXL-ERROR]", msg)
}

// Errorf log
func Errorf(tpl string, args ...interface{}) {
	log.Println("[XXL-ERROR]", fmt.Sprintf(tpl, args...))
}

// Debug log
func Debug(msg string) {
	if option.IsDebugMode() {
		log.Println("[XXL-DEBUG]", msg)
	}
}

// Debugf log
func Debugf(tpl string, args ...interface{}) {
	if option.IsDebugMode() {
		log.Println("[XXL-DEBUG]", fmt.Sprintf(tpl, args...))
	}
}
