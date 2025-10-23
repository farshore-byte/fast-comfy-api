package handler

import (
	"fmt"
	"net/http"
	"time"

	"farshore.ai/fast-comfy-api/core"
	"farshore.ai/fast-comfy-api/model"
	"github.com/gin-gonic/gin"
)

// =======================
// 📦 通用响应结构
// =======================
type Response struct {
	Code int         `json:"code"` // 0 成功，-1 失败
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Success(data interface{}) Response {
	return Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
}

func Fail(msg string) Response {
	return Response{
		Code: -1,
		Msg:  msg,
	}
}

// =======================
// 💡 APIHandler 主体
// =======================
type APIHandler struct {
	APIManager *core.APIManager
}

// 创建实例
func NewAPIHandler(resourceDir string, s3Config model.S3Config, checkInterval time.Duration, enabled bool) *APIHandler {
	return &APIHandler{
		APIManager: core.NewAPIManager(resourceDir, s3Config, checkInterval, enabled),
	}
}

// =======================
// 🚀 生成任务接口
// =======================
func (h *APIHandler) GenerateSyncHandler(c *gin.Context) {
	var req struct {
		Token string                 `json:"token"`
		Vars  map[string]interface{} `json:"vars"`
	}

	// 参数解析
	if err := c.ShouldBindJSON(&req); err != nil {
		h.JSON(c, http.StatusBadRequest, Fail("invalid request body"))
		return
	}

	// 校验 token
	if req.Token == "" {
		h.JSON(c, http.StatusBadRequest, Fail("missing token"))
		return
	}

	// 调用核心逻辑
	urls, err := h.APIManager.GenerateSync(req.Token, req.Vars)
	if err != nil {
		h.JSON(c, http.StatusInternalServerError, Fail(err.Error()))
		return
	}

	// 成功响应
	h.JSON(c, http.StatusOK, Success(urls))
}

// ====================
// 📋 列出 API 接口
// ======================
func (h *APIHandler) ListAPIsHandler(c *gin.Context) {
	list := h.APIManager.ListAPIs()
	c.JSON(http.StatusOK, Success(list))
}

// =========================
// ⏱️ 启动 指定 API 服务
// ========================

func (h *APIHandler) StartAPIHandler(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, Fail("missing token"))
		return
	}
	if err := h.APIManager.StartAPI(token); err != nil {
		c.JSON(http.StatusBadRequest, Fail(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(fmt.Sprintf("API [%s] started", token)))
}

// =================
// ⏹️ 停止 指定 API 服务
// ==================
func (h *APIHandler) StopAPIHandler(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, Fail("missing token"))
		return
	}
	if err := h.APIManager.StopAPI(token); err != nil {
		c.JSON(http.StatusBadRequest, Fail(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(fmt.Sprintf("API [%s] stopped", token)))
}

// =======================
// 🧩 封装统一响应输出
// =======================
func (h *APIHandler) JSON(c *gin.Context, status int, resp Response) {
	c.JSON(status, resp)
}
