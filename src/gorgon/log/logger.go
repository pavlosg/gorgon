package log

import (
	"fmt"
	std_log "log"
	"runtime"
	"strconv"
)

type Level string

const (
	INFO    Level = "INFO"
	WARNING Level = "WARNING"
	ERROR   Level = "ERROR"
)

type Logger func(level Level, message string)

var _logger Logger = func(level Level, message string) {
	std_log.Println("gorgon:", level, message)
}

func GetLogger() Logger {
	return _logger
}

func SetLogger(logger Logger) {
	_logger = logger
}

func Log(level Level, format string, args ...interface{}) {
	log(level, format, args...)
}

func Info(format string, args ...interface{}) {
	log(INFO, format, args...)
}

func Warning(format string, args ...interface{}) {
	log(WARNING, format, args...)
}

func Error(format string, args ...interface{}) {
	log(ERROR, format, args...)
}

func log(level Level, format string, args ...interface{}) {
	if _logger == nil {
		return
	}
	buffer := []byte(fmt.Sprintf(format, args...))
	if _, file, line, ok := runtime.Caller(2); ok {
		buffer = append(buffer, " @"...)
		buffer = append(buffer, file...)
		buffer = append(buffer, ':')
		buffer = append(buffer, strconv.Itoa(line)...)
	}
	_logger(level, string(buffer))
}
