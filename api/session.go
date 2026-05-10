package api

import (
	"encoding/gob"
	"net/http"
	"strings"

	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginUser = "LOGIN_USER"
	// 默认 7 天 — AUDIT.md MED 修:之前 sessionMaxAge=0 = 永不过期,
	// 偷到 cookie 永久持有。给个无侵入默认值,管理员可在「设置」覆盖。
	defaultSessionMaxAgeSeconds = 7 * 24 * 60 * 60
)

func init() {
	gob.Register(model.User{})
}

// secureCookieOptions 拼装 cookie 安全选项。
//
// AUDIT.md C3:之前 Secure:false + 默认 HttpOnly:false + 无 SameSite —
// XSS 直接 document.cookie 偷 session,跨站 form 提交可带 cookie CSRF。
//
//   - HttpOnly:浏览器 JS 拿不到 cookie,XSS 没法直接外传 session
//   - SameSiteLax:跨站 GET 仍带(允许书签/外链回来),跨站 POST/PUT 不带 → CSRF 闭合
//   - Secure:仅在面板真的走 HTTPS 时才置 true(panel 同时支持 http /https,强行 Secure 会让 http 模式 cookie 永远不发出)。
//     探测口径:settings.webCertFile 非空 OR Request.TLS != nil。
func secureCookieOptions(c *gin.Context, maxAge int) sessions.Options {
	if maxAge <= 0 {
		maxAge = defaultSessionMaxAgeSeconds
	}
	o := sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if c.Request.TLS != nil {
		o.Secure = true
	} else if cert, _ := (&service.SettingService{}).GetCertFile(); strings.TrimSpace(cert) != "" {
		o.Secure = true
	}
	return o
}

func SetLoginUser(c *gin.Context, userName string, maxAge int) error {
	// 旧入参 maxAge 来自 settings.sessionMaxAge,单位"分钟" — 保持兼容,
	// 内部转秒。0 / 负数走 secureCookieOptions 的 7 天默认。
	maxAgeSec := 0
	if maxAge > 0 {
		maxAgeSec = maxAge * 60
	}
	options := secureCookieOptions(c, maxAgeSec)

	s := sessions.Default(c)
	s.Set(loginUser, userName)
	s.Options(options)

	return s.Save()
}

func SetMaxAge(c *gin.Context) error {
	s := sessions.Default(c)
	s.Options(secureCookieOptions(c, 0))
	return s.Save()
}

func GetLoginUser(c *gin.Context) string {
	s := sessions.Default(c)
	obj := s.Get(loginUser)
	if obj == nil {
		return ""
	}
	objStr, ok := obj.(string)
	if !ok {
		return ""
	}
	return objStr
}

func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != ""
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	o := secureCookieOptions(c, 0)
	o.MaxAge = -1 // -1 = 立刻删除 cookie
	s.Options(o)
	s.Save()
}
