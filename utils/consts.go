package utils

import "time"

const (
	OllamaDomain = "http://127.0.0.1:11434"

	HandleRequestTimeout       = 10 * time.Second
	HandleRequestStreamTimeout = 3 * time.Minute
	BackendHealthCheckDuration = 3 * time.Second

	TraceID = "trace_id"
)
