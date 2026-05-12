package response

import (
	"encoding/json"
	"net/http"
)

const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusPending = "pending"
)

// Response is the standard envelope returned by every endpoint.
type Response struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Status  string `json:"status"`
	Success bool   `json:"success"`
}

func Ok(data any, message string) Response {
	return Response{
		Data:    data,
		Message: message,
		Status:  StatusSuccess,
		Success: true,
	}
}

func Fail(message string) Response {
	return Response{
		Data:    nil,
		Message: message,
		Status:  StatusError,
		Success: false,
	}
}

func Pending(data any, message string) Response {
	return Response{
		Data:    data,
		Message: message,
		Status:  StatusPending,
		Success: true,
	}
}

// WriteJSON writes a Response to the http.ResponseWriter with the given status code.
func WriteJSON(w http.ResponseWriter, status_code int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status_code)
	_ = json.NewEncoder(w).Encode(resp)
}
