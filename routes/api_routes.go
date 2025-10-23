package routes

import (
	"farshore.ai/fast-comfy-api/handler"
	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(r *gin.Engine, h *handler.APIHandler) {
	api := r.Group("/api")
	{
		api.POST("/generate_sync", h.GenerateSyncHandler)

		// ✅ 管理接口
		api.GET("/list", h.ListAPIsHandler)
		api.POST("/start/:token", h.StartAPIHandler)
		api.POST("/stop/:token", h.StopAPIHandler)
	}
}
