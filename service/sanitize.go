package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

// SanitizeRawConfig 兼容老格式 / 别家工具(xray、v2ray-core、x-ui 等)
// 粘贴过来的 JSON,自动修正 sing-box 1.13.x strict-unmarshal 会拒收的字段:
//
//   1. port 类字段从 string → number(server_port / listen_port / 等都是 uint16)
//   2. acme.key_type 删除(sing-box 1.13.5+ 移除该字段,见 stripACMEKeyType)
//
// 入口:所有 Service.Save 在写库 / reload 前先跑一遍。
//
// 失败兜底:任何步骤报错都返回原 raw,不破坏用户输入(后续 Save 还是会撞错,
// 但至少不会因为 sanitizer bug 把好数据搞丢)。
func SanitizeRawConfig(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	coerceSchemaRecursive(v)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

// portFields sing-box 里语义为单个 uint16 的字段名。
// 不包含路由规则的 "port"(那是数组,可以含 string 也可含 int range),
// 路由 port 数组里如果是 ["80","443"] 这种,sing-box 不会报 unmarshal 错(union 类型)。
var portFields = map[string]bool{
	"server_port":           true,
	"listen_port":           true,
	"alternative_http_port": true,
	"alternative_tls_port":  true,
	"fallback_port":         true,
	"public_port":           true,
}

// keysToRemove sing-box 1.13.x 删除的字段 — 见到就移除。
// 加新字段时跟 sing-box upstream changelog 同步。
var keysToRemove = map[string]bool{
	"key_type": true, // ACME 字段,1.13.5+ 移除(走 certmagic 默认 P256/ECDSA)
}

// StripDownstreamFields 删除所有以 "_" 开头的字段。
//
// sui 前端在 route.rules 里加 `_nb_binding` 等 metadata 标记由内部逻辑生成的规则,
// 旧版 sing-box 容忍未知字段所以一直能跑;sing-box 1.13.x strict-unmarshal
// 见到任何下划线前缀字段直接报 "unknown field"、reload 失败。
//
// 这些字段只能在 sui 数据层(setting.config / DB)流转,不能下发给 sing-box。
// 调用点:ConfigService.GetConfig 的 marshal 后(下发给 sing-box 前)。
//
// setting 表里仍保留这些字段,前端通过 /api/load → SettingService.GetConfig
// 拿到的 raw config 包含 _nb_binding,业务逻辑(查找/删除 binding 规则)不受影响。
func StripDownstreamFields(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	if !stripUnderscoreRecursive(v) {
		return raw
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return out
}

// stripUnderscoreRecursive 返回是否真的删了任何字段(供 caller 决定要不要重 marshal)
func stripUnderscoreRecursive(v any) bool {
	changed := false
	switch tv := v.(type) {
	case map[string]any:
		for k, val := range tv {
			if strings.HasPrefix(k, "_") {
				delete(tv, k)
				changed = true
				continue
			}
			if stripUnderscoreRecursive(val) {
				changed = true
			}
		}
	case []any:
		for _, item := range tv {
			if stripUnderscoreRecursive(item) {
				changed = true
			}
		}
	}
	return changed
}

func coerceSchemaRecursive(v any) {
	switch tv := v.(type) {
	case map[string]any:
		for k, val := range tv {
			if keysToRemove[k] {
				delete(tv, k)
				continue
			}
			if portFields[k] {
				if s, ok := val.(string); ok {
					if n, err := strconv.Atoi(s); err == nil {
						tv[k] = float64(n)
						continue
					}
				}
			}
			coerceSchemaRecursive(val)
		}
	case []any:
		for _, item := range tv {
			coerceSchemaRecursive(item)
		}
	}
}
