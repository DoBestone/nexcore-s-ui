package v1

import (
	"crypto/subtle"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/service"

	"github.com/gin-gonic/gin"
)

// 与 nexcore-x-ui 同款鉴权传输 — Bearer 优先,X-API-Token 兜底。
// query string `?api_token=` 永不接受(进 access log + 浏览器历史 = 泄漏)。
const (
	bearerPrefix = "Bearer "
	headerToken  = "X-API-Token"
)

func ExtractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); h != "" {
		if strings.HasPrefix(h, bearerPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(h, bearerPrefix))
		}
	}
	if h := c.GetHeader(headerToken); h != "" {
		return strings.TrimSpace(h)
	}
	// nexcore-s-ui 上游 v2 用 Token: <t> 头 — 兼容,这样老脚本不会因为
	// 切到 v1 端点就失效。
	if h := c.GetHeader("Token"); h != "" {
		return strings.TrimSpace(h)
	}
	return ""
}

// tokenMemo 是 v1 鉴权的内存副本,跟 ApiV2Handler.tokens 等价但独立维护 —
// 让 /api/v1 与 /apiv2 解耦,各自的 reload 节奏独立(避免一个 panic 拖另一个)。
type tokenMemo struct {
	mu     sync.RWMutex
	loaded time.Time
	items  []memoItem
}

type memoItem struct {
	Token    string
	Username string
	Desc     string
	Expiry   int64
}

var memo = &tokenMemo{}

// Reload 拉一次 model.Tokens 全量,丢内存里。鉴权热路径只读,不查 db。
//
// 调用方:web/web.go 启动时 + 每次 admin 加/删 token 后(主动)。
// fail-soft:db 故障时保留前次成功载入的副本。
func Reload() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	now := time.Now().Unix()
	var rows []model.Tokens
	err := db.Model(&model.Tokens{}).Preload("User").
		Where("expiry == 0 OR expiry > ?", now).
		Find(&rows).Error
	if err != nil {
		return err
	}
	items := make([]memoItem, 0, len(rows))
	for _, r := range rows {
		username := ""
		if r.User != nil {
			username = r.User.Username
		}
		items = append(items, memoItem{
			Token:    r.Token,
			Username: username,
			Desc:     r.Desc,
			Expiry:   r.Expiry,
		})
	}
	memo.mu.Lock()
	memo.items = items
	memo.loaded = time.Now()
	memo.mu.Unlock()
	return nil
}

// resolveToken constant-time 比对内存中所有 token,命中即返回身份。
// constant-time 不是为了"防 token 字典攻击" — 在这种 32-char 随机串里没
// 意义 — 而是 cargo cult 自 x-ui 的做法,保持代码风格一致。
func resolveToken(raw string) (memoItem, bool) {
	if raw == "" {
		return memoItem{}, false
	}
	memo.mu.RLock()
	defer memo.mu.RUnlock()
	now := time.Now().Unix()
	for _, it := range memo.items {
		if it.Expiry > 0 && it.Expiry < now {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(it.Token), []byte(raw)) == 1 {
			return it, true
		}
	}
	return memoItem{}, false
}

// AuthMiddleware 失败:401 + missing_api_token / invalid_api_token,与 x-ui
// 同 code,主控不用区分对端。
//
// 命中后注入 ctx:
//   api_token        = raw value
//   api_token_user   = username (s-ui 的"创建该 token 的人")
//   api_token_desc   = token 备注
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := ExtractToken(c)
		if raw == "" {
			Unauthorized(c, "missing_api_token")
			return
		}
		it, ok := resolveToken(raw)
		if !ok {
			Unauthorized(c, "invalid_api_token")
			return
		}
		c.Set("api_token", raw)
		c.Set("api_token_user", it.Username)
		c.Set("api_token_desc", it.Desc)
		c.Next()
	}
}

// AccessLogMiddleware 把 /api/v1/* 的每次调用落进 model.ApiLog,与 /apiv2
// 共用同一张表。便于审计统一。
func AccessLogMiddleware(logSvc *service.ApiLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		entry := &model.ApiLog{
			DateTime:  started.Unix(),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			LatencyMs: time.Since(started).Milliseconds(),
			RemoteIp:  remoteIP(c),
			Username:  c.GetString("api_token_user"),
			TokenDesc: c.GetString("api_token_desc"),
		}
		if len(c.Errors) > 0 {
			entry.Err = c.Errors[0].Error()
		}
		logSvc.Add(entry)
	}
}

func remoteIP(c *gin.Context) string {
	if v := c.GetHeader("X-Forwarded-For"); v != "" {
		ips := strings.Split(v, ",")
		return strings.TrimSpace(ips[0])
	}
	addr := c.Request.RemoteAddr
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return ip
}
