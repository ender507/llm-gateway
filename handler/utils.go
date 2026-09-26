package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ender507/llm-gateway/internal/metrics"
)

type HandlerErrorType string

const (
	upstreamError       HandlerErrorType = "upstream_error"
	internalError       HandlerErrorType = "internal_error"
	invalidRequestError HandlerErrorType = "invalid_request_error"
)

func errorResponse(c *gin.Context, modelName string, httpCode int, msg string, errType HandlerErrorType) {
	metrics.RecordRequest(modelName, strconv.Itoa(httpCode))
	c.JSON(httpCode, gin.H{
		"error": gin.H{"message": msg, "type": errType, "code": nil},
	})
}
