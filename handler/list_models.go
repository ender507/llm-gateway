package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ender507/llm-gateway/internal/errs"
	"github.com/ender507/llm-gateway/internal/llm"
	"github.com/ender507/llm-gateway/utils"
	"github.com/gin-gonic/gin"
)

type ListModelsResp struct {
	Object string           `json:"object"`
	Data   []ListModelsItem `json:"data"`
}

type ListModelsItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

func ListModelsHandler(c *gin.Context) {
	traceID, _ := c.Get(utils.TraceID)
	log := utils.GetLogger()
	ctx := c.Request.Context()
	ctx, cancel := context.WithTimeout(ctx, utils.HandleRequestTimeout)
	defer cancel()

	modelBackendMap := llm.GetBackendManager().ModelBackendMap()
	if len(modelBackendMap) == 0 {
		err := errs.UpstreamServiceUnavailable("no available backend", fmt.Errorf("available backend num is 0"))
		log.Errorw(err.Error(), "trace_id", traceID)
		errorResponse(c, "-", err)
		return
	}
	resp := ListModelsResp{
		Object: "list",
	}
	for modelName, backend := range modelBackendMap {
		if len(backend) == 0 {
			continue
		}
		resp.Data = append(resp.Data, ListModelsItem{
			ID:      modelName,
			Object:  "model",
			OwnedBy: "ollama",
		})
	}
	log.Infow("list models success", "trace_id", traceID, "model_count", len(resp.Data))
	c.JSON(http.StatusOK, resp)
}
