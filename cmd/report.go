package cmd

// `sui report` —— install.sh 末尾把"装完的凭据"POST 给 webhook 用的子命令。
// 跟 nexcore-x-ui 的 report.go 同款契约,主控对接代码不区分两端。
//
// 安全模型:
//   - HMAC-SHA256(body, REPORT_KEY) → X-NexCore-Signature 头
//   - HTTPS 强制,-allow-http 才允许明文
//   - 单次 POST,10s 超时,不重试 — 失败由 install.sh warn 提示
//   - REPORT_KEY 必须经 ENV 传入(走 CLI 会进 /proc/<pid>/cmdline → ps 可见)
//
// 用法:
//   REPORT_KEY=secret \
//   NEXCORE_REPORT_PASSWORD="$pwd" \
//   NEXCORE_REPORT_API_TOKEN="$tok" \
//     sui report -url https://provider.example.com/nexcore/callback [-allow-http]

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/service"
)

// runReport 由 ParseCmd 在 `case "report"` 分支调用。reportURL / allowHTTP
// 走 CLI 解析(URL 不算秘密);REPORT_KEY 等敏感值走 ENV。
func runReport(reportURL string, allowHTTP bool) int {
	key := os.Getenv("REPORT_KEY")
	if reportURL == "" {
		fmt.Fprintln(os.Stderr, "report: -url is required")
		return 2
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "report: REPORT_KEY env is required (HMAC signing key)")
		return 2
	}

	if strings.HasPrefix(strings.ToLower(reportURL), "http://") && !allowHTTP {
		fmt.Fprintln(os.Stderr, "report: refusing to POST credentials over plain http; use https or pass -allow-http for testing")
		return 2
	}
	if !strings.HasPrefix(strings.ToLower(reportURL), "http://") && !strings.HasPrefix(strings.ToLower(reportURL), "https://") {
		fmt.Fprintln(os.Stderr, "report: -url must begin with http:// or https://")
		return 2
	}

	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Fprintln(os.Stderr, "report: db init failed:", err)
		return 1
	}
	settingSvc := service.SettingService{}
	userSvc := service.UserService{}

	// 拼 payload — 主控按 X-NexCore-Signature 验完签后即可消费
	port, _ := settingSvc.GetPort()
	pathStr, _ := settingSvc.GetWebPath()
	domain, _ := settingSvc.GetWebDomain()
	user, _ := userSvc.GetFirstUser()
	username := ""
	if user != nil {
		username = user.Username
	}

	scheme := "http"
	if cert, _ := settingSvc.GetCertFile(); cert != "" {
		scheme = "https"
	}
	host := domain
	if host == "" {
		host = "127.0.0.1"
	}
	panelURL := fmt.Sprintf("%s://%s:%d%s", scheme, host, port, pathStr)

	payload := map[string]interface{}{
		"timestamp":    time.Now().Unix(),
		"panel_type":   "sui",
		"panel_name":   config.GetName(),
		"panelVersion": config.GetVersion(),
		"panel": map[string]interface{}{
			"url":      panelURL,
			"port":     port,
			"webPath":  pathStr,
			"domain":   domain,
			"scheme":   scheme,
			"username": username,
			"password": os.Getenv("NEXCORE_REPORT_PASSWORD"),
		},
		"api": map[string]interface{}{
			"token": os.Getenv("NEXCORE_REPORT_API_TOKEN"),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "report: marshal payload failed:", err)
		return 1
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("POST", reportURL, bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, "report: build request failed:", err)
		return 1
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NexCore-Signature", sig)
	req.Header.Set("User-Agent", "nexcore-s-ui/"+config.GetVersion())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "report: POST failed:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "report: webhook returned %d\n", resp.StatusCode)
		return 1
	}
	fmt.Println("ok")
	return 0
}
