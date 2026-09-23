package utils

import "time"

const (
	OllamaDomain = "http://127.0.0.1:11434"

	HandleRequestTimeout       = 10 * time.Second
	BackendHealthCheckDuration = 3 * time.Second

	TraceID = "trace_id"
)
