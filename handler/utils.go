package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ender507/llm-gateway/internal/errs"
	"github.com/ender507/llm-gateway/internal/metrics"
)

type HandlerErrorType string

const (
	upstreamError       HandlerErrorType = "upstream_error"
	internalError       HandlerErrorType = "internal_error"
	invalidRequestError HandlerErrorType = "invalid_request_error"
)

func errorResponse(c *gin.Context, modelName string, err *errs.GatewayError) {
	metrics.RecordRequest(modelName, strconv.Itoa(err.Code))
	c.JSON(err.Code, gin.H{
		"error": gin.H{
			"message": err.Message,
			"type":    errTypeOf(err.Code),
			"code":    nil,
		},
	})
}

// OpenAI API 规范里的 error.type
var codeErrorTypeMap = map[int]string{
	http.StatusBadRequest:          "invalid_request_error",
	http.StatusNotFound:            "invalid_request_error",
	http.StatusTooManyRequests:     "rate_limit_error",
	http.StatusInternalServerError: "internal_error",
	http.StatusBadGateway:          "upstream_error",
	http.StatusServiceUnavailable:  "upstream_error",
}

func errTypeOf(code int) string {
	if t, ok := codeErrorTypeMap[code]; ok {
		return t
	}
	return "internal_error"
}
