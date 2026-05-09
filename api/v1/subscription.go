package v1

// 分享链接端点 — 替代被删的 sub.go。只返 link + qrcode,不做订阅(用户明确
// 不要订阅逻辑,主控自己拼分发)。数据源:client.Links(Save 时由
// ClientService.updateLinksWithFixedInbounds 算好的 [{remark, type, uri}])。
//
// 路由(register 在 v1.go):
//   GET /inbounds/:id/links             — 单入站所有客户端的 link 列表
//   GET /inbounds/:id/links/by-email    — 同上但按 client.name 索引,带 qrcode
//   GET /inbounds/:id/clients/:email/share — 单 client 的 link + qrcode
//
// host 参数:?host=domain.com 强制覆盖 link 里的服务器字段(主控跨域名分发场景)

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
)

type linkEntry struct {
	Remark string `json:"remark"`
	Type   string `json:"type"`
	URI    string `json:"uri"`
}

// gatherInboundLinks 找该 inbound 的所有 enable client 的链接。
// hostOverride 不空时把 link 里的 server 字段强制替换。
func (a *Controller) gatherInboundLinks(inboundID uint, hostOverride string) ([]gin.H, *model.Inbound, error) {
	db := database.GetDB()
	var ib model.Inbound
	if err := db.Where("id = ?", inboundID).First(&ib).Error; err != nil {
		return nil, nil, err
	}
	var clients []model.Client
	if err := db.Raw(
		"SELECT * FROM clients WHERE enable = ? AND ? IN (SELECT json_each.value FROM json_each(clients.inbounds))",
		true, inboundID,
	).Scan(&clients).Error; err != nil {
		return nil, &ib, err
	}

	out := make([]gin.H, 0, len(clients))
	for _, cli := range clients {
		var links []linkEntry
		if err := json.Unmarshal(cli.Links, &links); err != nil {
			continue
		}
		for _, l := range links {
			// remark = inbound.tag(ClientService.updateLinksWithFixedInbounds 写入)
			if l.Remark != ib.Tag {
				continue
			}
			uri := l.URI
			if hostOverride != "" {
				uri = rewriteLinkHost(uri, hostOverride)
			}
			out = append(out, gin.H{
				"name":   cli.Name,
				"remark": l.Remark,
				"type":   l.Type,
				"link":   uri,
			})
		}
	}
	return out, &ib, nil
}

// rewriteLinkHost 把链接里的 server 字段改成 host。仅替换 add/server,
// sni / utls 等保持原样(它们是 TLS 层语义,不能跟着分发域名走)。
func rewriteLinkHost(uri, host string) string {
	switch {
	case strings.HasPrefix(uri, "vmess://"):
		// vmess 是 base64(json),解出来改 add 再编回去
		payload := strings.TrimPrefix(uri, "vmess://")
		raw, err := base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			raw, err = base64.StdEncoding.DecodeString(payload)
			if err != nil {
				return uri
			}
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return uri
		}
		obj["add"] = host
		nb, _ := json.Marshal(obj)
		return "vmess://" + base64.StdEncoding.EncodeToString(nb)
	case strings.HasPrefix(uri, "vless://"),
		strings.HasPrefix(uri, "trojan://"),
		strings.HasPrefix(uri, "socks5://"),
		strings.HasPrefix(uri, "ss://"),
		strings.HasPrefix(uri, "hysteria2://"),
		strings.HasPrefix(uri, "hy2://"),
		strings.HasPrefix(uri, "tuic://"),
		strings.HasPrefix(uri, "anytls://"),
		strings.HasPrefix(uri, "naive+https://"):
		// userinfo@host:port?params 形式 — 找 @ 之后到 ?/#// 之前
		atIdx := strings.Index(uri, "@")
		if atIdx < 0 {
			return uri
		}
		rest := uri[atIdx+1:]
		stop := len(rest)
		for _, sep := range []string{"?", "#", "/"} {
			if i := strings.Index(rest, sep); i >= 0 && i < stop {
				stop = i
			}
		}
		hostport := rest[:stop]
		colon := strings.LastIndex(hostport, ":")
		if colon < 0 {
			return uri
		}
		port := hostport[colon:]
		return uri[:atIdx+1] + host + port + rest[stop:]
	}
	return uri
}

// resolveHostFromCtx:?host= 优先,否则用 panel 自己的 host(settings.webDomain
// 或 Request.Host fallback)。
func resolveHostFromCtx(c *gin.Context) string {
	if h := c.Query("host"); h != "" {
		return h
	}
	return getPanelHost(c)
}

// makeQR 生成 PNG base64 data URL。256×256 是 v2rayN/Shadowrocket 扫码的常见值。
func makeQR(text string) string {
	if text == "" {
		return ""
	}
	png, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// ---------- handlers ----------

// GET /inbounds/:id/links?host=
func (a *Controller) inboundLinks(c *gin.Context) {
	idStr := c.Param("id")
	n, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "invalid_id", "invalid id: "+idStr)
		return
	}
	host := resolveHostFromCtx(c)
	links, ib, err := a.gatherInboundLinks(uint(n), host)
	if err != nil {
		NotFound(c, "inbound_not_found", "inbound not found: "+idStr)
		return
	}
	OK(c, gin.H{
		"inbound": ib.Tag,
		"host":    host,
		"links":   links,
	})
}

// GET /inbounds/:id/links/by-email?host=
// 返回 {<client.name>: {link, qrcode, type, remark}}
func (a *Controller) inboundLinksByEmail(c *gin.Context) {
	idStr := c.Param("id")
	n, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "invalid_id", "invalid id: "+idStr)
		return
	}
	host := resolveHostFromCtx(c)
	links, _, err := a.gatherInboundLinks(uint(n), host)
	if err != nil {
		NotFound(c, "inbound_not_found", "inbound not found: "+idStr)
		return
	}
	out := gin.H{}
	for _, l := range links {
		name, _ := l["name"].(string)
		uri, _ := l["link"].(string)
		// 同一 client 多协议 link(mixed → socks5+http) — 只取首条
		if _, exists := out[name]; exists {
			continue
		}
		out[name] = gin.H{
			"link":   uri,
			"qrcode": makeQR(uri),
			"type":   l["type"],
			"remark": l["remark"],
		}
	}
	OK(c, out)
}

// GET /inbounds/:id/clients/:email/share?host=
func (a *Controller) inboundClientShare(c *gin.Context) {
	idStr := c.Param("id")
	email := c.Param("email")
	n, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "invalid_id", "invalid id: "+idStr)
		return
	}
	host := resolveHostFromCtx(c)
	links, _, err := a.gatherInboundLinks(uint(n), host)
	if err != nil {
		NotFound(c, "inbound_not_found", "inbound not found: "+idStr)
		return
	}
	for _, l := range links {
		if name, _ := l["name"].(string); name == email {
			uri, _ := l["link"].(string)
			OK(c, gin.H{
				"name":   email,
				"link":   uri,
				"qrcode": makeQR(uri),
				"type":   l["type"],
			})
			return
		}
	}
	NotFound(c, "client_not_found", "email not found in inbound: "+email)
}
