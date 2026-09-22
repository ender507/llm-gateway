package main

import (
	"fmt"
	"time"

	"github.com/ender507/llm-gateway/handler"
	"github.com/ender507/llm-gateway/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	// 初始化日志实例
	err := utils.InitLogger()
	if err != nil {
		panic(fmt.Errorf("failed to init logger: %w", err))
	}
	logger := utils.GetLogger()

	// 注册服务路由与中间件
	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(traceMiddleware(), accessEndLogMiddleware())

	v1Group := r.Group("/v1")
	v1Group.GET("/models", handler.ListModelsHandler)
	v1Group.POST("/chat/completions", handler.ChatCompletionsHandler)

	logger.Infow("service starting", "listen_addr", ":8080")
	err = r.Run(":8080")
	if err != nil {
		logger.Fatalw("server start failed", "err", err)
	}
}

// traceMiddleware 为每个请求独立分配 traceID
func traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.NewString()
		c.Set(utils.TraceID, traceID)
		c.Header("X-Trace-ID", traceID)
		c.Next()
	}
}

// accessEndLogMiddleware 在请求结束后统计请求基本数据并输出到日志
func accessEndLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// 执行后续handler
		c.Next()
		// c.Next()返回后，请求处理完毕，统计耗时
		traceID, ok := c.Get(utils.TraceID)
		if !ok {
			traceID = "unknown"
		}
		costMs := time.Since(start).Milliseconds()
		utils.GetLogger().Infow("http access result",
			utils.TraceID, traceID,
			"method", c.Request.Method,
			"route_path", c.FullPath(),
			"status_code", c.Writer.Status(),
			"cost_ms", costMs,
		)
	}
}
