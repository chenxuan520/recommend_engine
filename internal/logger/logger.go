package logger

import (
	"log"
	"os"
)

var (
	debugMode = false
	infoLog   = log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lmsgprefix)
	debugLog  = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lmsgprefix)
	errorLog  = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmsgprefix)
)

// SetDebug 设置是否开启调试模式
func SetDebug(debug bool) {
	debugMode = debug
}

// Info 打印信息日志
func Info(format string, v ...interface{}) {
	infoLog.Printf(format, v...)
}

// Debug 打印调试日志
func Debug(format string, v ...interface{}) {
	if debugMode {
		debugLog.Printf(format, v...)
	}
}

// Error 打印错误日志
func Error(format string, v ...interface{}) {
	errorLog.Printf(format, v...)
}

// Fatal 打印错误日志并退出
func Fatal(format string, v ...interface{}) {
	errorLog.Fatalf(format, v...)
}
