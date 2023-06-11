package utils

import (
	"fmt"

	"github.com/j32u4ukh/glog/v2"
)

var logger *glog.Logger

// func init() {
// 	InitLogger(glog.DebugLevel, "", glog.BasicOption(glog.WarnLevel, true, false, true))
// }

func SetLogger(lg *glog.Logger) {
	logger = lg
}

func Debug(message string, a ...any) {
	if logger != nil {
		logger.Debug(message, a...)
	} else {
		fmt.Printf("[Debug] %s\n", fmt.Sprintf(message, a...))
	}
}

func Info(message string, a ...any) {
	if logger != nil {
		logger.Info(message, a...)
	} else {
		fmt.Printf("[Info] %s\n", fmt.Sprintf(message, a...))
	}
}

func Warn(message string, a ...any) {
	if logger != nil {
		logger.Warn(message, a...)
	} else {
		fmt.Printf("[Warn] %s\n", fmt.Sprintf(message, a...))
	}
}

func Error(message string, a ...any) {
	if logger != nil {
		logger.Debug(message, a...)
	} else {
		fmt.Printf("[Error] %s\n", fmt.Sprintf(message, a...))
	}
}
