package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 响应外壳与 nexcore-x-ui 完全一致 — 这样主控的 SDK / 业务系统不用区分对端
// 是 x-ui 还是 s-ui,直接复用同一套 client。
//
// 成功:
//   { "data": <object|array|null> }
//
// 失败:
//   { "error": true, "code": "...", "message": "...", "details": {...} }
//
// 列表分页时附带 meta:
//   { "data": [...], "meta": { "total": N, "page": M, "size": K } }

type ErrorResp struct {
	Error   bool   `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func OKMeta(c *gin.Context, data any, meta gin.H) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": meta})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Fail(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorResp{
		Error:   true,
		Code:    code,
		Message: message,
		Details: details,
	})
}

func BadRequest(c *gin.Context, code, message string) {
	Fail(c, http.StatusBadRequest, code, message, nil)
}

func NotFound(c *gin.Context, code, message string) {
	Fail(c, http.StatusNotFound, code, message, nil)
}

func Unauthorized(c *gin.Context, code string) {
	Fail(c, http.StatusUnauthorized, code, "authentication required", nil)
}

func Forbidden(c *gin.Context, code, message string) {
	Fail(c, http.StatusForbidden, code, message, nil)
}

func Internal(c *gin.Context, code string, err error) {
	msg := "internal error"
	if err != nil {
		msg = err.Error()
	}
	Fail(c, http.StatusInternalServerError, code, msg, nil)
}
