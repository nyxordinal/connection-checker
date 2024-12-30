package main

import (
	"strconv"

	"github.com/sirupsen/logrus"
)

func CreateLogger(level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00",
	})
	logger.SetLevel(level)
	return logger
}

func atoi(str string) int {
	result, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return result
}
