package main

import (
	"encoding/json"
	"net/http"
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

type JsonResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func constructResponse(w http.ResponseWriter, httpCode int, message, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	response := JsonResponse{
		Message: message,
		Error:   err,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		appCtx.Logger.Error("Failed to encode JSON response: ", err)
	}
}

func constructResponseWithData(w http.ResponseWriter, httpCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		appCtx.Logger.Error("Failed to encode JSON response: ", err)
	}
}
