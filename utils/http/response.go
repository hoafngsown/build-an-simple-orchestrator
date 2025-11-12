package httputil

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, code int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("error encoding JSON response: %v\n", err)
			return err
		}
	}

	return nil
}

func WriteError(w http.ResponseWriter, code int, message string) {
	log.Printf("%s\n", message)

	response := ErrorResponse{
		HTTPStatusCode: code,
		Message:        message,
	}

	// Use WriteJSON to ensure consistent formatting
	// Ignore error here as we're already in error handling path
	_ = WriteJSON(w, code, response)
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
