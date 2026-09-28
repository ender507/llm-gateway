package errs

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func HandleUpstreamError(statusCode int, body []byte) *GatewayError {
	upstreamMsg := extractUpstreamMessage(body)
	retriable := statusCode == http.StatusTooManyRequests || statusCode >= 500
	msg := fmt.Sprintf("upstream error: %s", upstreamMsg)
	cause := fmt.Errorf("upstream status %d, body: %s", statusCode, string(body))
	return New(statusCode, msg, retriable, cause)
}

// extractUpstreamMessage 尝试从 Ollama JSON 错误体里解析出 error 字段
func extractUpstreamMessage(body []byte) string {
	var parsed struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return parsed.Error
	}
	return "unknown upstream error"
}

func BadRequest(msg string, cause error) *GatewayError {
	return New(http.StatusBadRequest, msg, false, cause)
}

func NotFound(msg string, cause error) *GatewayError {
	return New(http.StatusNotFound, msg, false, cause)
}

func UpstreamServiceUnavailable(msg string, cause error) *GatewayError {
	return New(http.StatusServiceUnavailable, msg, true, cause) // 上游不可用，可重试
}

func UpstreamBadGateway(msg string, cause error) *GatewayError {
	return New(http.StatusBadGateway, msg, true, cause) // 上游连接失败，可重试
}

func InternalServer(msg string, cause error) *GatewayError {
	return New(http.StatusInternalServerError, msg, false, cause)
}
