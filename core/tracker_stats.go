package core

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/alireza0/s-ui/database/model"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/atomic"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/network"
)

type Counter struct {
	read  *atomic.Int64
	write *atomic.Int64
	// cumRead/cumWrite 是 read/write 的"永不归零镜像":每次 GetStats() Swap 时
	// 把吞掉的 delta 也加进 cum,SnapshotCounters() 返 cum + 当前未 swap 的 read/write
	// 总和。给"实时速率"端点用 — 1.5s 高频拉单调递增,前端算 diff/dt 永远为正。
	cumRead  *atomic.Int64
	cumWrite *atomic.Int64
}

type StatsTracker struct {
	access    sync.Mutex
	inbounds  map[string]Counter
	outbounds map[string]Counter
	users     map[string]Counter
	// 在线 IP tracking:tag → source IP → last-seen unix sec。每条新连接刷
	// 时间戳,SnapshotOnlineIPs 按时间窗口(默认 60s)统计当前活跃 IP 数,
	// 顺手清理过期 IP 免内存涨。机场用来:看入站当前并发用户数(IP 维度)、
	// 检测客户端账号被多少 IP 同时使用(共享账号告警)。
	inboundIPs map[string]map[string]int64
	userIPs    map[string]map[string]int64
}

func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		inbounds:   make(map[string]Counter),
		outbounds:  make(map[string]Counter),
		users:      make(map[string]Counter),
		inboundIPs: make(map[string]map[string]int64),
		userIPs:    make(map[string]map[string]int64),
	}
}

// SnapshotCounters 返当前所有 inbound/outbound/user 的 read+write counter Load
// 值(不 Swap,不清零)。给"实时速率"端点用 — 前端 1.5s 拉一次,自己算 diff/dt。
//
// 跟 GetStats() 的区别:GetStats Swap 出来后值归零,只在 SaveStats cron(~10s)
// 调用,把 delta 写 DB。SnapshotCounters 不动 counter,可以高频拉而不会跟 cron
// 抢"周期 delta"。
//
// 返回:resource ("inbound"|"outbound"|"user") → tag → {up: read, down: write}
func (c *StatsTracker) SnapshotCounters() map[string]map[string]map[string]int64 {
	c.access.Lock()
	defer c.access.Unlock()
	// 总量 = 已落 cum 的部分 + 当前 pending(下次 GetStats 会 swap 走)
	total := func(ctr Counter) (up, down int64) {
		up = ctr.cumRead.Load() + ctr.read.Load()
		down = ctr.cumWrite.Load() + ctr.write.Load()
		return
	}
	out := map[string]map[string]map[string]int64{
		"inbound":  {},
		"outbound": {},
		"user":     {},
	}
	for tag, ctr := range c.inbounds {
		up, down := total(ctr)
		out["inbound"][tag] = map[string]int64{"up": up, "down": down}
	}
	for tag, ctr := range c.outbounds {
		up, down := total(ctr)
		out["outbound"][tag] = map[string]int64{"up": up, "down": down}
	}
	for tag, ctr := range c.users {
		up, down := total(ctr)
		out["user"][tag] = map[string]int64{"up": up, "down": down}
	}
	return out
}

func (c *StatsTracker) Reset() {
	c.access.Lock()
	defer c.access.Unlock()
	c.inbounds = make(map[string]Counter)
	c.outbounds = make(map[string]Counter)
	c.users = make(map[string]Counter)
	c.inboundIPs = make(map[string]map[string]int64)
	c.userIPs = make(map[string]map[string]int64)
}

// recordIP 写入连接的 source IP。已持锁的 caller 调用。
func (c *StatsTracker) recordIPLocked(inbound, user, ip string) {
	if ip == "" {
		return
	}
	now := time.Now().Unix()
	if inbound != "" {
		m, ok := c.inboundIPs[inbound]
		if !ok {
			m = make(map[string]int64)
			c.inboundIPs[inbound] = m
		}
		m[ip] = now
	}
	if user != "" {
		m, ok := c.userIPs[user]
		if !ok {
			m = make(map[string]int64)
			c.userIPs[user] = m
		}
		m[ip] = now
	}
}

// SnapshotOnlineIPs 返回 (inbound tag→活跃 IP 数, user name→活跃 IP 数)。
// "活跃" = lastSeen ≥ now-windowSec。顺手把过期 IP 删掉,免长期运行积累。
func (c *StatsTracker) SnapshotOnlineIPs(windowSec int64) (map[string]int, map[string]int) {
	c.access.Lock()
	defer c.access.Unlock()
	cutoff := time.Now().Unix() - windowSec
	prune := func(src map[string]map[string]int64) map[string]int {
		out := make(map[string]int)
		for tag, ips := range src {
			for ip, last := range ips {
				if last < cutoff {
					delete(ips, ip)
				}
			}
			if len(ips) == 0 {
				delete(src, tag)
				continue
			}
			out[tag] = len(ips)
		}
		return out
	}
	return prune(c.inboundIPs), prune(c.userIPs)
}

// QueryOnlineIPs 返回单个 tag(inbound 或 user)的活跃 IP 列表。
// resource 取 "inbound" 或 "user"。空返回空 slice,不返回 nil。
// 已经在 SaveStats 周期 prune 过过期项,这里读 + slice copy 不做修改。
func (c *StatsTracker) QueryOnlineIPs(resource, tag string, windowSec int64) []string {
	c.access.Lock()
	defer c.access.Unlock()
	var src map[string]map[string]int64
	switch resource {
	case "inbound":
		src = c.inboundIPs
	case "user":
		src = c.userIPs
	default:
		return []string{}
	}
	bucket, ok := src[tag]
	if !ok {
		return []string{}
	}
	cutoff := time.Now().Unix() - windowSec
	out := make([]string, 0, len(bucket))
	for ip, last := range bucket {
		if last >= cutoff {
			out = append(out, ip)
		}
	}
	return out
}

func (c *StatsTracker) getReadCounters(inbound string, outbound string, user string, sourceIP string) ([]*atomic.Int64, []*atomic.Int64) {
	var readCounter []*atomic.Int64
	var writeCounter []*atomic.Int64
	c.access.Lock()
	defer c.access.Unlock()

	if inbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.inbounds, inbound).read)
		writeCounter = append(writeCounter, c.inbounds[inbound].write)
	}
	if outbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.outbounds, outbound).read)
		writeCounter = append(writeCounter, c.outbounds[outbound].write)
	}
	if user != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.users, user).read)
		writeCounter = append(writeCounter, c.users[user].write)
	}
	// 同事务里记录 source IP,跟 counter 注册一起持锁
	c.recordIPLocked(inbound, user, sourceIP)
	return readCounter, writeCounter
}

func (c *StatsTracker) loadOrCreateCounter(obj *map[string]Counter, name string) Counter {
	counter, loaded := (*obj)[name]
	if loaded {
		return counter
	}
	counter = Counter{read: &atomic.Int64{}, write: &atomic.Int64{}, cumRead: &atomic.Int64{}, cumWrite: &atomic.Int64{}}
	(*obj)[name] = counter
	return counter
}

func (c *StatsTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	readCounter, writeCounter := c.getReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User, sourceIPStr(metadata))
	return bufio.NewInt64CounterConn(conn, readCounter, writeCounter)
}

func (c *StatsTracker) RoutedPacketConnection(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) network.PacketConn {
	readCounter, writeCounter := c.getReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User, sourceIPStr(metadata))
	return bufio.NewInt64CounterPacketConn(conn, readCounter, nil, writeCounter, nil)
}

// sourceIPStr 提取 metadata.Source 的 IP 字符串。Socksaddr 是 host-or-IP 联合体,
// IsIP() 才能拿到 netip.Addr;否则可能是 domain 形态(WS over CDN 客户端写
// 域名给入站时),那种没意义就跳过。
func sourceIPStr(m adapter.InboundContext) string {
	if !m.Source.IsIP() {
		return ""
	}
	addr := m.Source.Addr
	if !addr.IsValid() {
		return ""
	}
	return addr.String()
}

func (c *StatsTracker) GetStats() *[]model.Stats {
	c.access.Lock()
	defer c.access.Unlock()

	dt := time.Now().Unix()

	// swapAndAccumulate 把 read/write Swap 出来,同时把 delta 加到 cum 镜像
	// (供 SnapshotCounters 算永不归零的总量)
	swapAndAccumulate := func(counter Counter) (up, down int64) {
		down = counter.write.Swap(0)
		up = counter.read.Swap(0)
		if down > 0 {
			counter.cumWrite.Add(down)
		}
		if up > 0 {
			counter.cumRead.Add(up)
		}
		return
	}

	s := []model.Stats{}
	for inbound, counter := range c.inbounds {
		up, down := swapAndAccumulate(counter)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for outbound, counter := range c.outbounds {
		up, down := swapAndAccumulate(counter)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for user, counter := range c.users {
		up, down := swapAndAccumulate(counter)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: true,
				Traffic:   up,
			})
		}
	}
	return &s
}
