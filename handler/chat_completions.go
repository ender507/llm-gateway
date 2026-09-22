package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ender507/llm-gateway/utils"
	"github.com/gin-gonic/gin"
)

type ChatCompletionsRequest struct {
	Stream bool   `json:"stream"`
	Model  string `json:"model"`
}

func ChatCompletionsHandler(c *gin.Context) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	ctx := c.Request.Context()
	url := utils.OllamaDomain + "/v1/chat/completions"

	bodyBytes, reqBody, err := readChatRequestBody(c)
	if err != nil {
		log.Errorw("read request body failed", "trace_id", traceID, "err", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": "read request body failed", "type": utils.InvalidRequestError},
		})
		return
	}

	ollamaReq, err := buildOllamaChatRequest(ctx, url, bodyBytes)
	if err != nil {
		log.Errorw("build ollama request failed", "trace_id", traceID, "err", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": "build ollama request failed", "type": utils.InternalError},
		})
		return
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(ollamaReq)
	if err != nil {
		log.Errorw("call ollama chat failed", "trace_id", traceID, "model", reqBody.Model, "err", err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"message": "upstream ollama unreachable", "type": utils.UpstreamError},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorw("read upstream error body failed", "trace_id", traceID, "model", reqBody.Model, "status", resp.StatusCode, "err", err.Error())
			c.JSON(http.StatusBadGateway, gin.H{
				"error": gin.H{"message": "upstream service status not 200", "type": utils.UpstreamError},
			})
			return
		}
		log.Errorw("ollama return non-200 value", "trace_id", traceID, "model", reqBody.Model, "raw_body", string(errBody), "status", resp.StatusCode)
		c.JSON(resp.StatusCode, gin.H{
			"error": gin.H{"message": string(errBody), "type": utils.UpstreamError},
		})
		return
	}

	if reqBody.Stream {
		handleStreamChat(c, start, resp)
	} else {
		handleNonStreamChat(c, start, resp)
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

func buildOllamaChatRequest(ctx context.Context, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// handleStreamChat 流式SSE转发
func handleStreamChat(c *gin.Context, start time.Time, upstreamResp *http.Response) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Errorw("response writer does not support flusher", "trace_id", traceID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": "response writer does not support flusher", "type": utils.InvalidRequestError},
		})
		return
	}

	scanner := bufio.NewScanner(upstreamResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	firstToken := true
	for scanner.Scan() {
		line := scanner.Text()
		_, _ = fmt.Fprintf(c.Writer, "%s\n", line) // Scan 默认以 \n 为分隔符，因此需要再补上去掉的 \n
		flusher.Flush()

		if firstToken && line != "" && line != "data: [DONE]" {
			ttft := time.Since(start)
			log.Infow("ttft recorded", "trace_id", traceID, "ttft_ms", ttft.Milliseconds())
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

	log.Infow("chat stream completed", "trace_id", traceID, "cost_ms", time.Since(start).Milliseconds())
}

// handleNonStreamChat 非流式，一次性返回
func handleNonStreamChat(c *gin.Context, start time.Time, upstreamResp *http.Response) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	respBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		log.Errorw("read non-stream ollama response failed", "trace_id", traceID, "err", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": fmt.Sprintf("read non-stream ollama response failed: %v", err), "type": utils.InternalError},
		})
		return
	}
	log.Infow("chat non-stream completed", "trace_id", traceID, "cost_ms", time.Since(start).Milliseconds())
	c.Data(http.StatusOK, "application/json", respBody)
}
