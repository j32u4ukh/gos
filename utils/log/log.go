package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var logger *Logger

type Logger struct {
	lg    *log.Logger
	file  *os.File
	level LogLevel
	skip  int
}

func SetLogger(logName string, folder string, level LogLevel) error {
	lg, file, err := GetLogger(logName, folder)
	if err != nil {
		return errors.Wrap(err, "Failed to init logger.")
	}
	logger = &Logger{
		lg:    lg,
		file:  file,
		level: level,
		skip:  2,
	}
	return nil
}

func GetLogger(logName string, folder string) (*log.Logger, *os.File, error) {
	fileName := fmt.Sprintf("%s.log", logName)
	flag := os.O_WRONLY | os.O_CREATE | os.O_APPEND
	logPath := filepath.Join(folder, fileName)
	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating folder:", err)
		return nil, nil, errors.Wrapf(err, "Failed to make directory: %s", folder)
	}
	f, err := os.OpenFile(logPath, flag, 0666)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Failed to open: %s", logName)
	}
	logger := log.New(io.MultiWriter(os.Stdout, f), "", log.Ldate|log.Ltime)
	// ロガーを設定して、UTC 時間を使用
	logger.SetFlags(logger.Flags() | log.LUTC)
	return logger, f, nil
}

func SetLevel(level LogLevel) {
	logger.level = level
}

func SetSkip(skip int) {
	logger.skip = skip
}

func Debug(format string, a ...any) {
	if logger.level <= DEBUG {
		msg := "[Debug] " + fmt.Sprintf(format, a...)
		msg = formatMsg(msg)
		logger.lg.Println(msg)
	}
}

func Info(format string, a ...any) {
	if logger.level <= INFO {
		msg := "[Info] " + fmt.Sprintf(format, a...)
		msg = formatMsg(msg)
		logger.lg.Println(msg)
	}
}

func Warn(format string, a ...any) {
	if logger.level <= WARN {
		msg := "[Warn] " + fmt.Sprintf(format, a...)
		msg = formatMsg(msg)
		logger.lg.Println(msg)
	}
}

func Error(format string, a ...any) {
	msg := "[Error] " + fmt.Sprintf(format, a...)
	msg = formatMsg(msg)
	logger.lg.Println(msg)
}

func formatMsg(msg string) string {
	_, file, line, _ := runtime.Caller(logger.skip)
	return fmt.Sprintf("%s | %s:%d", msg, file, line)
}

func Close() error {
	err := logger.file.Close()
	if err != nil {
		return errors.Wrap(err, "Failed to close *os.File.")
	}
	return nil
}
