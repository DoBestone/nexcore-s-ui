package model

// BlockRule 兼容 nexcore-x-ui /block-rules 体系的快捷屏蔽规则。
//
// 跟节点「路由列表」(用户自己编辑的 sing-box route.rules)是两个独立模块:
//   - 路由列表:面板 UI 编辑 → 持久化到 setting.config 的 route.rules
//   - 屏蔽规则:本表存储,生成最终 sing-box config 时由 ConfigService 自动注入
//     到 route.rules 数组开头(action=reject,优先级高于路由列表)
//
// 两者在 sing-box 内核里都是 route.rules,但生命周期、UI 入口、命名空间完全分开,
// 用户在路由列表页看不到这里的规则,删本表行不会影响路由列表。
//
// 命名空间约定:remark 以 "[NexCore]" 前缀的行由主控批量管理,UI 标记为只读;
// 其它 remark 是节点本地手加,UI 可编辑。
type BlockRule struct {
	Id         uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Type       string `json:"type" gorm:"size:16;index"`        // domain | ip | geosite | geoip | port | protocol | source
	Value      string `json:"value" gorm:"size:512"`            // 多值用逗号分隔
	Remark     string `json:"remark" gorm:"size:255"`
	InboundTag string `json:"inboundTag" gorm:"size:128;index"` // 空 = 全局
	Enable     bool   `json:"enable" gorm:"default:true"`
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt  int64  `json:"-" gorm:"autoUpdateTime:milli"`
}

func (BlockRule) TableName() string { return "block_rules" }
