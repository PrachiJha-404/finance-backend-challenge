package apierr

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return e.Message
}

func Write(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Message})
}

func BadRequest(message string) *APIError {
	return &APIError{Code: http.StatusBadRequest, Message: message}
}

func Unauthorized(message string) *APIError {
	if message == "" {
		message = "unauthorized"
	}
	return &APIError{Code: http.StatusUnauthorized, Message: message}
}

func Forbidden(message string) *APIError {
	if message == "" {
		message = "forbidden"
	}
	return &APIError{Code: http.StatusForbidden, Message: message}
}

func NotFound(message string) *APIError {
	if message == "" {
		message = "not found"
	}
	return &APIError{Code: http.StatusNotFound, Message: message}
}

func Conflict(message string) *APIError {
	return &APIError{Code: http.StatusConflict, Message: message}
}

func Internal(message string) *APIError {
	if message == "" {
		message = "internal server error"
	}
	return &APIError{Code: http.StatusInternalServerError, Message: message}
}
