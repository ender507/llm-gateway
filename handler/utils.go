package handler

import (
	"github.com/gin-gonic/gin"
)

type HandlerErrorType string

const (
	upstreamError       HandlerErrorType = "upstream_error"
	internalError       HandlerErrorType = "internal_error"
	invalidRequestError HandlerErrorType = "invalid_request_error"
)

func errorResponse(c *gin.Context, httpCode int, msg string, errType HandlerErrorType) {
	c.JSON(httpCode, gin.H{
		"error": gin.H{"message": msg, "type": errType, "code": -1},
	})
}
