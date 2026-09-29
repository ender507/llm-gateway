package handler

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ender507/llm-gateway/internal/errs"
	"github.com/ender507/llm-gateway/internal/metrics"
	"github.com/ender507/llm-gateway/utils"
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

func callOllama(ctx context.Context, url string, bodyBytes []byte) (*http.Response, *errs.GatewayError) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errs.BadRequest("build ollama request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errs.UpstreamBadGateway("failed to request ollama", err)
	}

	if resp.StatusCode == http.StatusOK {
		return resp, nil
	}

	errBody, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, errs.UpstreamBadGateway("read upstream error body failed", readErr)
	}
	ge := errs.HandleUpstreamError(resp.StatusCode, errBody)
	return nil, ge
}

func callOllamaWithRetry(ctx context.Context, modelName, url string, bodyBytes []byte, traceID any) (*http.Response, *errs.GatewayError) {
	backoff := utils.InitialBackoff
	log := utils.GetLogger()

	for attempt := 0; attempt < utils.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			// 客户端提前断开，直接退出，不重试
			return nil, errs.UpstreamBadGateway("client context canceled", err)
		}

		resp, reqErr := callOllama(ctx, url, bodyBytes)
		if reqErr == nil {
			if attempt > 0 { // 首次成功不打点
				metrics.IncGatewayUpstreamRetryTotal(modelName, url, "success")
			}
			return resp, nil
		}
		if !reqErr.Retriable || attempt >= utils.MaxRetries-1 {
			// 不可重试 / 最后一次尝试，直接返回错误
			if attempt > 0 {
				metrics.IncGatewayUpstreamRetryTotal(modelName, url, "failed")
			}
			return nil, reqErr
		}
		metrics.IncGatewayUpstreamRetryTotal(modelName, url, "triggered")
		log.Infow("retry upstream request", "trace_id", traceID, "attempt", attempt+1, "backoff_ms", backoff.Milliseconds(), "err_msg", reqErr.Message)
		if !waitRetry(ctx, backoff) {
			return nil, errs.UpstreamBadGateway("retry wait canceled by context", ctx.Err())
		}
		backoff *= 2
		if backoff > utils.MaxBackoff {
			backoff = utils.MaxBackoff
		}
	}
	return nil, errs.InternalServer("unreachable retry loop", nil)
}

func waitRetry(ctx context.Context, backoff time.Duration) bool {
	jitter := time.Duration(0.8 + 0.4*rand.Float64())
	wait := time.Duration(float64(backoff) * float64(jitter))
	if wait > utils.MaxBackoff {
		wait = utils.MaxBackoff
	}
	select {
	case <-time.After(wait):
		return true
	case <-ctx.Done():
		return false
	}
}
