package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ender507/llm-gateway/internal/errs"
	"github.com/ender507/llm-gateway/internal/llm"
	"github.com/ender507/llm-gateway/internal/metrics"
	"github.com/ender507/llm-gateway/utils"
)

type ChatCompletionsRequest struct {
	Stream bool   `json:"stream"`
	Model  string `json:"model"`
}

func ChatCompletionsHandler(c *gin.Context) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	ctx := c.Request.Context()

	bodyBytes, reqBody, err := readChatRequestBody(c)
	if err != nil {
		err := errs.BadRequest("read request body failed", err)
		log.Errorw("read request body failed", "trace_id", traceID, "err", err.Error())
		errorResponse(c, "-", err)
		return
	}

	timeout := utils.HandleRequestTimeout
	if reqBody.Stream {
		timeout = utils.HandleRequestStreamTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	backendManager := llm.GetBackendManager()
	backendList := backendManager.GetBackendsByModel(reqBody.Model)
	if len(backendList) == 0 {
		err := errs.UpstreamServiceUnavailable("no backend available", fmt.Errorf("available backend num is 0"))
		log.Errorw("no backend available", "trace_id", traceID, "model", reqBody.Model)
		errorResponse(c, reqBody.Model, err)
		return
	}
	availableBackendList := make([]*llm.Backend, 0)
	for _, backend := range backendList {
		if backend.CB().Allow() {
			availableBackendList = append(availableBackendList, backend)
		}
	}
	if len(availableBackendList) == 0 {
		err := errs.UpstreamServiceUnavailable("no available backend", fmt.Errorf("all backends are circuit breaker open"))
		errorResponse(c, reqBody.Model, err)
		return
	}
	sessionID := c.GetHeader("X-Session-Id")
	selected, reused := backendManager.GetBackendBySession(sessionID, availableBackendList)
	acquired := selected.TryAcquire(ctx, utils.QueueTimeout)
	if !acquired {
		err := errs.UpstreamServiceUnavailable("backend busy", fmt.Errorf("backend %s: acquire timeout after %s", selected.Endpoint, utils.QueueTimeout))
		metrics.IncQueueTimeout(selected.Endpoint)
		log.Errorw("backend busy", "trace_id", traceID, "err", err.Error())
		errorResponse(c, reqBody.Model, err)
		return
	}
	defer selected.Release()
	if !selected.CB().BeginCall() {
		err := errs.UpstreamServiceUnavailable("backend probe quota full", fmt.Errorf("backend %s", selected.Endpoint))
		errorResponse(c, reqBody.Model, err)
		return
	}
	defer selected.CB().ReleaseProbe()

	log.Infow("backend selected", "trace_id", traceID, "model", reqBody.Model, "session_id", sessionID, "endpoint", selected.Endpoint, "reused_session", reused)

	url := selected.Endpoint + "/v1/chat/completions"
	start := time.Now()
	var resp *http.Response
	var callErr *errs.GatewayError
	if reqBody.Stream {
		// 流式：单次调用，不重试
		resp, callErr = callOllama(ctx, url, bodyBytes)
	} else {
		// 非流式：开启指数抖动重试
		resp, callErr = callOllamaWithRetry(ctx, reqBody.Model, url, bodyBytes, traceID, selected)
	}
	reportBackendResult(selected, callErr)
	if callErr != nil {
		utils.GetLogger().Errorw("call ollama upstream failed", "trace_id", traceID, "model", reqBody.Model, "backend", url, "err_msg", callErr.Message, "retriable", callErr.Retriable)
		errorResponse(c, reqBody.Model, callErr)
		return
	}
	defer resp.Body.Close()

	if reqBody.Stream {
		handleStreamChat(c, start, resp, reqBody.Model)
	} else {
		handleNonStreamChat(c, start, resp, reqBody.Model)
	}

}

func readChatRequestBody(c *gin.Context) (body []byte, reqBody ChatCompletionsRequest, err error) {
	body, err = io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, reqBody, err
	}
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		return nil, reqBody, err
	}
	return body, reqBody, nil
}

// handleStreamChat 流式SSE转发
func handleStreamChat(c *gin.Context, start time.Time, upstreamResp *http.Response, modelName string) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		err := errs.InternalServer("response writer does not support flusher", fmt.Errorf("c.Write is not a http.Flusher"))
		log.Errorw("response writer does not support flusher", "trace_id", traceID, "err", err.Error())
		errorResponse(c, modelName, err)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(upstreamResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	firstToken := true
	for scanner.Scan() {
		line := scanner.Text()
		_, _ = fmt.Fprintf(c.Writer, "%s\n", line) // Scan 默认以 \n 为分隔符，因此需要再补上去掉的 \n
		flusher.Flush()

		if firstToken && line != "" && line != "data: [DONE]" {
			ttft := time.Since(start)
			metrics.RecordTTFT(modelName, ttft.Seconds())
			log.Infow("ttft recorded", "trace_id", traceID, "ttft_ms", ttft.Milliseconds(), "model_name", modelName)
			firstToken = false
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case <-c.Request.Context().Done():
			log.Infow("client disconnected, stream ended", "trace_id", traceID)
		default:
			log.Errorw("read stream from upstream failed", "trace_id", traceID, "err", err.Error())
		}
	}

	log.Infow("chat stream completed", "trace_id", traceID, "cost_ms", time.Since(start).Milliseconds(), "model_name", modelName)
	metrics.RecordRequest(modelName, "200")
}

// handleNonStreamChat 非流式，一次性返回
func handleNonStreamChat(c *gin.Context, start time.Time, upstreamResp *http.Response, modelName string) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	respBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		err := errs.InternalServer("read non-stream ollama response failed", err)
		log.Errorw("read non-stream ollama response failed", "trace_id", traceID, "err", err.Error())
		errorResponse(c, modelName, err)
		return
	}
	log.Infow("chat non-stream completed", "trace_id", traceID, "cost_ms", time.Since(start).Milliseconds(), "model_name", modelName)
	c.Data(http.StatusOK, "application/json", respBody)
	metrics.RecordRequest(modelName, "200")
}
