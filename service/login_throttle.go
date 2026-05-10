package service

import (
	"sync"
	"time"
)

// loginThrottle 是 per-IP 登录失败节流器。
//
// AUDIT.md C4:之前登录无任何防爆破,attacker 拿到面板就能无限并发跑
// 字典攻击。这里加最小代价的内存计数 + 阶梯延迟:
//
//	失败 < 5 次:不延迟
//	5 ≤ 失败 < 10:延迟 5s
//	10 ≤ 失败 < 20:延迟 15s
//	失败 ≥ 20:延迟 60s
//
// 不引依赖、不持久化(进程重启清零) — 简单稳健。多机部署的真正防爆破
// 应该在反向代理层做(nginx limit_req / Cloudflare WAF),这里是兜底。
//
// IP 维度:同 IP 暴破时序攻击不同账户都计入(防止 attacker 用同 IP 跑
// username 字典)。成功登录会清零该 IP 的计数。
//
// TTL:每条记录 30 分钟无活动自动 GC,避免长期积累内存。
type loginThrottle struct {
	mu      sync.Mutex
	entries map[string]*throttleEntry
}

type throttleEntry struct {
	failures int
	lastSeen time.Time
}

const throttleTTL = 30 * time.Minute

var globalLoginThrottle = &loginThrottle{
	entries: make(map[string]*throttleEntry),
}

// CheckAndDelay 阻塞调用方相应时长(模拟"等不及") — 失败次数越多 sleep 越久。
// 用 sleep 是因为它对 attacker 自动化脚本足够"贵",且不需要把网络层挂起。
// 调用前(login flow 入口)调一次,内部按当前失败数延迟。
func (t *loginThrottle) CheckAndDelay(ip string) {
	if ip == "" {
		return
	}
	t.mu.Lock()
	t.gcLocked()
	e := t.entries[ip]
	failures := 0
	if e != nil {
		failures = e.failures
	}
	t.mu.Unlock()

	delay := delayForFailures(failures)
	if delay > 0 {
		time.Sleep(delay)
	}
}

// MarkFailure 登录失败时调,失败计数 +1。
func (t *loginThrottle) MarkFailure(ip string) {
	if ip == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gcLocked()
	e := t.entries[ip]
	if e == nil {
		e = &throttleEntry{}
		t.entries[ip] = e
	}
	e.failures++
	e.lastSeen = time.Now()
}

// MarkSuccess 登录成功时调,清零该 IP 的失败计数(避免合法用户长期累积罚单)。
func (t *loginThrottle) MarkSuccess(ip string) {
	if ip == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, ip)
}

// FailureCount 返回当前失败次数(给外部审计 / log 用)。
func (t *loginThrottle) FailureCount(ip string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if e := t.entries[ip]; e != nil {
		return e.failures
	}
	return 0
}

func (t *loginThrottle) gcLocked() {
	now := time.Now()
	for ip, e := range t.entries {
		if now.Sub(e.lastSeen) > throttleTTL {
			delete(t.entries, ip)
		}
	}
}

func delayForFailures(n int) time.Duration {
	switch {
	case n >= 20:
		return 60 * time.Second
	case n >= 10:
		return 15 * time.Second
	case n >= 5:
		return 5 * time.Second
	default:
		return 0
	}
}

// LoginThrottle 全局单例,api 层 / Login 内部用。
func LoginThrottle() *loginThrottle {
	return globalLoginThrottle
}
