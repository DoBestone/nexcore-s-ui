package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/util/common"
)

const cfAPIBase = "https://api.cloudflare.com/client/v4"

type CloudflareService struct{}

type CFZone struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type cfResp struct {
	Success bool              `json:"success"`
	Errors  []json.RawMessage `json:"errors"`
	Result  json.RawMessage   `json:"result"`
}

// httpJSON 不是 best-of-class 客户端,但够用 - CF API 老老实实的 JSON,无 cursor/pagination 复杂度。
// token 可以是 Global API Key + email(老式)或 API Token(推荐),这里只支持 Bearer Token。
func (s *CloudflareService) httpJSON(method, path, token string, body interface{}) (*cfResp, error) {
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, cfAPIBase+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var r cfResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, common.NewError("cloudflare API non-JSON (HTTP ", res.StatusCode, "): ", string(raw))
	}
	if !r.Success {
		return &r, common.NewError("cloudflare API failed: ", string(raw))
	}
	return &r, nil
}

// VerifyToken 检查 token 是否合法。CF 的 /user/tokens/verify 是 token 自检入口。
func (s *CloudflareService) VerifyToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return common.NewError("empty cloudflare token")
	}
	_, err := s.httpJSON("GET", "/user/tokens/verify", token, nil)
	return err
}

// ListZones 列出 token 可见的 zone。Global API Token 一般可见全部 zone,
// 普通 scoped token 受 token 权限限制 — 这是用户应当感知的边界。
func (s *CloudflareService) ListZones(token string) ([]CFZone, error) {
	r, err := s.httpJSON("GET", "/zones?per_page=50", token, nil)
	if err != nil {
		return nil, err
	}
	var zones []CFZone
	if err := json.Unmarshal(r.Result, &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

type cfDnsRecord struct {
	Id      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// UpsertARecord 在 zone 下保证存在一条 A 记录指向 ip;若已有同名 A 记录,
// 改为目标 ip(避免重复创建)。返回最终 fqdn 与 record id。
func (s *CloudflareService) UpsertARecord(token, zoneId, name, ip string, proxied bool) (string, string, error) {
	zoneName, err := s.zoneName(token, zoneId)
	if err != nil {
		return "", "", err
	}
	fqdn := name
	if name == "" || name == "@" {
		fqdn = zoneName
		name = "@"
	} else if !strings.HasSuffix(name, zoneName) {
		fqdn = name + "." + zoneName
	}

	listURL := fmt.Sprintf("/zones/%s/dns_records?type=A&name=%s", zoneId, fqdn)
	listResp, err := s.httpJSON("GET", listURL, token, nil)
	if err != nil {
		return "", "", err
	}
	var existing []cfDnsRecord
	if err := json.Unmarshal(listResp.Result, &existing); err != nil {
		return "", "", err
	}

	rec := cfDnsRecord{
		Type:    "A",
		Name:    fqdn,
		Content: ip,
		TTL:     1, // automatic
		Proxied: proxied,
	}

	if len(existing) > 0 {
		recId := existing[0].Id
		updURL := fmt.Sprintf("/zones/%s/dns_records/%s", zoneId, recId)
		_, err := s.httpJSON("PUT", updURL, token, rec)
		if err != nil {
			return "", "", err
		}
		return fqdn, recId, nil
	}
	createURL := fmt.Sprintf("/zones/%s/dns_records", zoneId)
	cresp, err := s.httpJSON("POST", createURL, token, rec)
	if err != nil {
		return "", "", err
	}
	var created cfDnsRecord
	if err := json.Unmarshal(cresp.Result, &created); err != nil {
		return "", "", err
	}
	return fqdn, created.Id, nil
}

func (s *CloudflareService) zoneName(token, zoneId string) (string, error) {
	r, err := s.httpJSON("GET", "/zones/"+zoneId, token, nil)
	if err != nil {
		return "", err
	}
	var z CFZone
	if err := json.Unmarshal(r.Result, &z); err != nil {
		return "", err
	}
	return z.Name, nil
}

// RandomSubdomain 给一个 8 字符 hex 前缀;给运营人员"我懒得想前缀"的快捷出口。
func (s *CloudflareService) RandomSubdomain(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	tail := hex.EncodeToString(b)
	if prefix == "" {
		return "n-" + tail
	}
	return strings.TrimSpace(prefix) + "-" + tail
}

// IssueTLS 不真签证书 — sing-box 自己内置 ACME(with_acme build tag),
// 只要把 acme 块写入 model.Tls.Server,sing-box 启动时会用 dns01_challenge
// 走 Cloudflare 取证书。这里我们只负责落库一条 model.Tls 记录,把 cf
// 的 api_token 嵌进去。
//
// 入参:
//   - name:    TLS 记录在面板里的名字
//   - fqdn:    要签证书的域名(已通过 UpsertARecord 解析到本机 IP)
//   - email:   ACME 注册邮箱
//   - cfToken: dns01 用的 Cloudflare API Token
//   - dataDir: ACME 缓存目录(每个 TLS 一个,免冲突)
func (s *CloudflareService) IssueTLS(name, fqdn, email, cfToken, dataDir string) (uint, error) {
	if dataDir == "" {
		dataDir = "./acme/" + fqdn
	}
	server := map[string]interface{}{
		"enabled":     true,
		"server_name": fqdn,
		"alpn":        []string{"h2", "http/1.1"},
		"acme": map[string]interface{}{
			"domain":              []string{fqdn},
			"data_directory":      dataDir,
			"default_server_name": fqdn,
			"email":               email,
			"provider":            "letsencrypt",
			"dns01_challenge": map[string]interface{}{
				"provider":  "cloudflare",
				"api_token": cfToken,
			},
		},
	}
	clientCfg := map[string]interface{}{
		"enabled":     true,
		"server_name": fqdn,
	}

	srvBytes, err := json.Marshal(server)
	if err != nil {
		return 0, err
	}
	cliBytes, err := json.Marshal(clientCfg)
	if err != nil {
		return 0, err
	}

	if name == "" {
		name = fqdn
	}
	tls := model.Tls{
		Name:   name,
		Server: srvBytes,
		Client: cliBytes,
	}

	db := database.GetDB()
	if err := db.Create(&tls).Error; err != nil {
		return 0, err
	}
	return tls.Id, nil
}
