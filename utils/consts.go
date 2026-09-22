package utils

const (
	OllamaDomain = "http://127.0.0.1:11434"

	TraceID = "trace_id"
)

type HandlerErrorType string

const (
	UpstreamError       HandlerErrorType = "upstream_error"
	InternalError       HandlerErrorType = "internal_error"
	InvalidRequestError HandlerErrorType = "invalid_request_error"
)
