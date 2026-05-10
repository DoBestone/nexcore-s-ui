package model

import "encoding/json"

type Setting struct {
	Id uint `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	// AUDIT.md H5:Key 加 UNIQUE 索引 — 老 schema 没有索引,saveSetting 是
	// read-then-write 竞态(并发 Save 同 key 可能写入两条同 key 的行)。
	// AutoMigrate 会自动建 UNIQUE INDEX,顺便加速 GetAllSetting 的 WHERE key=?。
	Key   string `json:"key" form:"key" gorm:"uniqueIndex"`
	Value string `json:"value" form:"value"`
}

type Tls struct {
	Id     uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name   string          `json:"name" form:"name"`
	Server json.RawMessage `json:"server" form:"server"`
	Client json.RawMessage `json:"client" form:"client"`
}

type User struct {
	Id         uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	LastLogins string `json:"lastLogin"`
}

type Client struct {
	Id       uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Enable   bool            `json:"enable" form:"enable"`
	Name     string          `json:"name" form:"name"`
	Config   json.RawMessage `json:"config,omitempty" form:"config"`
	Inbounds json.RawMessage `json:"inbounds" form:"inbounds"`
	Links    json.RawMessage `json:"links,omitempty" form:"links"`
	Volume   int64           `json:"volume" form:"volume"`
	Expiry   int64           `json:"expiry" form:"expiry"`
	Down     int64           `json:"down" form:"down"`
	Up       int64           `json:"up" form:"up"`
	Desc     string          `json:"desc" form:"desc"`
	Group    string          `json:"group" form:"group"`

	// Delay start and periodic reset
	DelayStart bool  `json:"delayStart" form:"delayStart" gorm:"default:false;not null"`
	AutoReset  bool  `json:"autoReset" form:"autoReset" gorm:"default:false;not null"`
	ResetDays  int   `json:"resetDays" form:"resetDays" gorm:"default:0;not null"`
	NextReset  int64 `json:"nextReset" form:"nextReset" gorm:"default:0;not null"`
	TotalUp    int64 `json:"totalUp" form:"totalUp" gorm:"default:0;not null"`
	TotalDown  int64 `json:"totalDown" form:"totalDown" gorm:"default:0;not null"`
}

type Stats struct {
	Id uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	// AUDIT.md MED:加复合索引 (resource, tag, date_time) — 流量聚合查询
	// 几乎全部都是 WHERE resource=? AND tag=? AND date_time BETWEEN ?,
	// 老 schema 全表扫,数据量起来后冷启动几秒。索引名固定,GORM 用同名
	// 三列做组合 idx,顺序按选择度(resource 最少 / date_time 最多)。
	DateTime  int64  `json:"dateTime" gorm:"index:idx_stats_lookup,priority:3"`
	Resource  string `json:"resource" gorm:"index:idx_stats_lookup,priority:1"`
	Tag       string `json:"tag" gorm:"index:idx_stats_lookup,priority:2"`
	Direction bool   `json:"direction"`
	Traffic   int64  `json:"traffic"`
}

type Changes struct {
	Id       uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	DateTime int64           `json:"dateTime"`
	Actor    string          `json:"actor"`
	Key      string          `json:"key"`
	Action   string          `json:"action"`
	Obj      json.RawMessage `json:"obj"`
}

type Tokens struct {
	Id     uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Desc   string `json:"desc" form:"desc"`
	Token  string `json:"token" form:"token"`
	Expiry int64  `json:"expiry" form:"expiry"`
	UserId uint   `json:"userId" form:"userId"`
	User   *User  `json:"user" gorm:"foreignKey:UserId;references:Id"`
}

// ApiLog 记录每一次 /apiv2/* 调用 - 给操作员审计 / debug 用。
// 不记录 /api/*(面板自身),不然会被前端轮询(load/onlines)刷爆。
type ApiLog struct {
	Id        uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	DateTime  int64  `json:"dateTime" gorm:"index"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	RemoteIp  string `json:"remoteIp"`
	Username  string `json:"username"`
	TokenDesc string `json:"tokenDesc"`
	Err       string `json:"err"`
}
