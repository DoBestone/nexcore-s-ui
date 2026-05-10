package v1

// 屏蔽规则 CRUD —— x-ui 兼容面 + 主控全局屏蔽规则下发对接。
//
// 路由(register 在 v1.go):
//   GET    /block-rules           # 列表
//   POST   /block-rules           # 新增
//   PUT    /block-rules/:id       # 全量更新
//   DELETE /block-rules/:id       # 删除
//   GET    /block-rules/presets   # 预置规则集(给主控/前端"应用预置"用)
//
// 跟节点「路由列表」是两个独立模块:
//   - 路由列表:用户在面板 UI 编辑的 sing-box route.rules,持久化在 setting.config
//   - 屏蔽规则:本表 block_rules 存,生成最终 sing-box config 时由 ConfigService
//     自动注入到 route.rules 数组开头(action=reject,优先级高于路由列表)
// 用户在路由列表页 UI 看不到本表规则,删本表行也不会动路由列表。
//
// 底层路径:写操作统一走 configSvc.Save("block-rules", act, data, ...)
// 跟 sui 内部面板调用同一份逻辑(service/config.go:Save 的 obj switch),
// reload sing-box / 写 model.Changes 审计、tx 一致性都复用。

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"

	"github.com/gin-gonic/gin"
)

// blockRuleAllowedTypes 白名单 — 跟 service/config.go 的 blockRuleToRouteRule
// 翻译表保持同步,新增类型需要两边一起改。
var blockRuleAllowedTypes = map[string]bool{
	"domain":   true,
	"ip":       true,
	"geosite":  true,
	"geoip":    true,
	"port":     true,
	"protocol": true,
	"source":   true,
}

func (a *Controller) listBlockRules(c *gin.Context) {
	rules, err := a.configSvc.BlockRuleService.GetAll()
	if err != nil {
		Internal(c, "db_error", err)
		return
	}
	OK(c, rules)
}

func (a *Controller) createBlockRule(c *gin.Context) {
	var r model.BlockRule
	if err := c.ShouldBindJSON(&r); err != nil {
		BadRequest(c, "invalid_body", err.Error())
		return
	}
	r.Id = 0
	if err := validateBlockRule(&r); err != nil {
		BadRequest(c, "invalid_rule", err.Error())
		return
	}
	data, _ := json.Marshal(r)
	if _, err := a.configSvc.Save("block-rules", "new", data, "", c.GetString("api_token_user"), getPanelHost(c)); err != nil {
		Internal(c, mapSaveErr(err, "save_failed"), err)
		return
	}
	// 主控期望 POST 返回带 id 的对象。Save 内部 Create 会回填 r.Id 但 data
	// 已经 marshal 出去了,这里读最新一条按 type+value+remark 反查 id —
	// 对单次新增(无并发)正确;并发新增由 model.Changes 兜底审计。
	var saved model.BlockRule
	if err := database.GetDB().Where("type = ? AND value = ? AND remark = ?", r.Type, r.Value, r.Remark).Order("id DESC").First(&saved).Error; err == nil {
		OK(c, saved)
		return
	}
	OK(c, r) // 兜底:返本次入参,id=0
}

func (a *Controller) updateBlockRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		BadRequest(c, "invalid_id", "id must be positive integer")
		return
	}
	var r model.BlockRule
	if err := c.ShouldBindJSON(&r); err != nil {
		BadRequest(c, "invalid_body", err.Error())
		return
	}
	r.Id = uint(id)
	if err := validateBlockRule(&r); err != nil {
		BadRequest(c, "invalid_rule", err.Error())
		return
	}
	data, _ := json.Marshal(r)
	if _, err := a.configSvc.Save("block-rules", "edit", data, "", c.GetString("api_token_user"), getPanelHost(c)); err != nil {
		Internal(c, mapSaveErr(err, "save_failed"), err)
		return
	}
	OK(c, r)
}

func (a *Controller) deleteBlockRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		BadRequest(c, "invalid_id", "id must be positive integer")
		return
	}
	data, _ := json.Marshal([]uint{uint(id)})
	if _, err := a.configSvc.Save("block-rules", "del", data, "", c.GetString("api_token_user"), getPanelHost(c)); err != nil {
		Internal(c, mapSaveErr(err, "save_failed"), err)
		return
	}
	NoContent(c)
}

// listBlockRulePresets 预置规则集:主控/前端"应用预置"按钮的数据源。
//
// 复用 sing-box 生态的 geosite-* 数据集,把常见"屏蔽广告/追踪器/私网直连"打包
// 成一组可一键导入的规则。主控用 .key 做幂等键,前端用 .rules[] 批量 POST。
func (a *Controller) listBlockRulePresets(c *gin.Context) {
	OK(c, []gin.H{
		{
			"key":         "ads",
			"name":        "屏蔽广告(geosite-category-ads-all)",
			"description": "命中所有广告 + 反欺诈域名,推荐启用",
			"rules": []model.BlockRule{
				{Type: "geosite", Value: "category-ads-all", Remark: "屏蔽广告", Enable: true},
			},
		},
		{
			"key":         "tracker",
			"name":        "屏蔽追踪器",
			"description": "命中常见 analytics / tracker 域名",
			"rules": []model.BlockRule{
				{Type: "geosite", Value: "category-public-tracker", Remark: "屏蔽追踪器", Enable: true},
			},
		},
		{
			"key":         "porn",
			"name":        "屏蔽成人内容(geosite-category-porn)",
			"description": "命中成人内容域名;部分场景(家庭/学校网络)需要",
			"rules": []model.BlockRule{
				{Type: "geosite", Value: "category-porn", Remark: "屏蔽成人内容", Enable: true},
			},
		},
	})
}

// validateBlockRule:type 必须在白名单,value 至少一个非空 token。
func validateBlockRule(r *model.BlockRule) error {
	if !blockRuleAllowedTypes[r.Type] {
		return fmt.Errorf("type 必须是 domain|ip|geosite|geoip|port|protocol|source 之一,got: %q", r.Type)
	}
	if !hasNonEmptyToken(r.Value) {
		return fmt.Errorf("value 不能为空")
	}
	if r.Type == "port" {
		// 提前验证 port 全是合法整数,避免 reload 时 sing-box 起不来
		for _, v := range strings.Split(r.Value, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, err := strconv.Atoi(v); err != nil {
				return fmt.Errorf("port 值必须是整数,got: %q", v)
			}
		}
	}
	return nil
}

func hasNonEmptyToken(s string) bool {
	for _, p := range strings.Split(s, ",") {
		if strings.TrimSpace(p) != "" {
			return true
		}
	}
	return false
}
