package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func DecodeJSON[T any](r *http.Request) (T, error) {
	var result T
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	err := d.Decode(&result)
	if err != nil {
		return result, fmt.Errorf("error unmarshalling body: %w", err)
	}

	return result, nil
}

func GetURLParam(r *http.Request, key string) (string, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return "", fmt.Errorf("missing or empty URL parameter: %s", key)
	}
	return value, nil
}

func GetUUIDParam(r *http.Request, key string) (uuid.UUID, error) {
	value, err := GetURLParam(r, key)
	if err != nil {
		return uuid.UUID{}, err
	}

	parsedUUID, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid UUID format for parameter %s: %w", key, err)
	}

	return parsedUUID, nil
}
