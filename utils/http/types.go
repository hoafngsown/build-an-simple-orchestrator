package httputil

type ErrorResponse struct {
	HTTPStatusCode int    `json:"http_status_code"`
	Message        string `json:"message"`
}
