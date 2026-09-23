package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ender507/llm-gateway/utils"
	"github.com/gin-gonic/gin"
)

type OllamaTagsResp struct {
	Models []OllamaModelItem `json:"models"`
}

type OllamaModelItem struct {
	Name string `json:"name"`
}

type ListModelsResp struct {
	Object string           `json:"object"`
	Data   []ListModelsItem `json:"data"`
}

type ListModelsItem struct {
	ID     string `json:"id"`
	Object string `json:"object"`
}

func ListModelsHandler(c *gin.Context) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	ctx := c.Request.Context()
	ctx, cancel := context.WithTimeout(ctx, utils.HandleRequestTimeout)
	defer cancel()
	url := utils.OllamaDomain + "/api/tags"

	// 请求 ollama，获取现有模型信息
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Errorw("build ollama tags request failed", "trace_id", traceID, "err", err.Error())
		errorResponse(c, http.StatusInternalServerError, fmt.Sprintf("create request failed: %s", err), internalError)
		return
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Errorw("call ollama /api/tags failed", "trace_id", traceID, "err", err.Error())
		errorResponse(c, http.StatusInternalServerError, fmt.Sprintf("fetch models from ollama failed: %s", err), upstreamError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Errorw("ollama return non-200 status", "trace_id", traceID, "status", resp.StatusCode)
		errorResponse(c, http.StatusInternalServerError, fmt.Sprintf("ollama response code(%v) not 200", resp.StatusCode), upstreamError)
		return
	}
	var ollamaResp OllamaTagsResp
	err = json.NewDecoder(resp.Body).Decode(&ollamaResp)
	if err != nil {
		log.Errorw("decode ollama tags json failed",
			"trace_id", traceID,
			"err", err.Error(),
		)
		errorResponse(c, http.StatusInternalServerError, fmt.Sprintf("parse upstream json failed: %s", err), internalError)
		return
	}

	// 将结果封装成 OpenAI API 规范并返回
	openaiData := make([]ListModelsItem, 0, len(ollamaResp.Models))
	for _, m := range ollamaResp.Models {
		openaiData = append(openaiData, ListModelsItem{
			ID:     m.Name,
			Object: "model",
		})
	}

	result := ListModelsResp{
		Object: "list",
		Data:   openaiData,
	}

	log.Infow("list models success",
		"trace_id", traceID,
		"model_count", len(openaiData),
	)
	c.JSON(http.StatusOK, result)
}
