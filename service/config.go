package service

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/alireza0/s-ui/core"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/util/common"
)

var (
	LastUpdate          int64
	corePtr             *core.Core
	startCoreMu         sync.Mutex
	startCoreInProgress bool
	lastStartFailTime   time.Time
	startCooldown       = 15 * time.Second
)

type ConfigService struct {
	ClientService
	TlsService
	SettingService
	InboundService
	OutboundService
	EndpointService
}

// ensureBuiltinOutbounds 保证最终 config 至少包含 tag=direct 出站。
// rule-set 默认 `download_detour: "direct"`、DNS 把 geoip-cn / 私网回 direct,
// 一旦用户把 direct 删了或空 DB 状态,sing-box 启动直接报
// "download detour not found: direct"。缺则在末尾补一个最简 direct。
//
// 不自动注 block —— 新版 sing-box 已废弃 block outbound 类型,改用路由 action=reject。
func ensureBuiltinOutbounds(outbounds []json.RawMessage) []json.RawMessage {
	for _, o := range outbounds {
		var meta struct{ Tag string }
		if err := json.Unmarshal(o, &meta); err == nil && meta.Tag == "direct" {
			return outbounds
		}
	}
	return append(outbounds, json.RawMessage(`{"type":"direct","tag":"direct"}`))
}

type SingBoxConfig struct {
	Log          json.RawMessage   `json:"log"`
	Dns          json.RawMessage   `json:"dns"`
	Ntp          json.RawMessage   `json:"ntp"`
	Inbounds     []json.RawMessage `json:"inbounds"`
	Outbounds    []json.RawMessage `json:"outbounds"`
	Endpoints    []json.RawMessage `json:"endpoints"`
	Route        json.RawMessage   `json:"route"`
	Experimental json.RawMessage   `json:"experimental"`
}

func NewConfigService(core *core.Core) *ConfigService {
	corePtr = core
	return &ConfigService{}
}

func (s *ConfigService) GetConfig(data string) (*[]byte, error) {
	var err error
	if len(data) == 0 {
		data, err = s.SettingService.GetConfig()
		if err != nil {
			return nil, err
		}
	}
	singboxConfig := SingBoxConfig{}
	err = json.Unmarshal([]byte(data), &singboxConfig)
	if err != nil {
		return nil, err
	}

	singboxConfig.Inbounds, err = s.InboundService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	singboxConfig.Outbounds, err = s.OutboundService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	// 兜底:必须存在 `direct` 出站。
	// rule-set 默认 `download_detour: "direct"`,DNS 默认规则也会把私网 / geoip-cn
	// 路由到 direct;一旦用户把 direct 出站删了或从未建过(空 DB 状态),
	// sing-box 启动会立刻挂在 "download detour not found: direct"。
	// 这里在生成最终 config 时强制保证 direct/block 两个出站存在,缺则补。
	singboxConfig.Outbounds = ensureBuiltinOutbounds(singboxConfig.Outbounds)
	singboxConfig.Endpoints, err = s.EndpointService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	rawConfig, err := json.MarshalIndent(singboxConfig, "", "  ")
	if err != nil {
		return nil, err
	}
	return &rawConfig, nil
}

func (s *ConfigService) StartCore() error {
	if corePtr.IsRunning() {
		return nil
	}
	startCoreMu.Lock()
	if startCoreInProgress {
		startCoreMu.Unlock()
		return nil
	}
	if time.Since(lastStartFailTime) < startCooldown {
		// startCooldown / time.Second 仍是 time.Duration 类型,String() 输出
		// 它的纳秒数(如 "15ns")。强转 int64 才得到字面量秒数。
		logger.Info("start core cooldown ", int64(startCooldown/time.Second), " seconds")
		startCoreMu.Unlock()
		return nil
	}
	startCoreInProgress = true
	startCoreMu.Unlock()
	defer func() {
		startCoreMu.Lock()
		startCoreInProgress = false
		startCoreMu.Unlock()
	}()

	logger.Info("starting core")
	rawConfig, err := s.GetConfig("")
	if err != nil {
		return err
	}
	err = corePtr.Start(*rawConfig)
	if err != nil {
		startCoreMu.Lock()
		lastStartFailTime = time.Now()
		startCoreMu.Unlock()
		logger.Error("start sing-box err:", err.Error())
		return err
	}
	logger.Info("sing-box started")
	return nil
}

func (s *ConfigService) RestartCore() error {
	err := s.StopCore()
	if err != nil {
		return err
	}
	return s.StartCore()
}

func (s *ConfigService) restartCoreWithConfig(config json.RawMessage) error {
	startCoreMu.Lock()
	if startCoreInProgress {
		startCoreMu.Unlock()
		return nil
	}
	startCoreInProgress = true
	startCoreMu.Unlock()
	defer func() {
		startCoreMu.Lock()
		startCoreInProgress = false
		startCoreMu.Unlock()
	}()

	if corePtr.IsRunning() {
		if err := corePtr.Stop(); err != nil {
			logger.Error("restart sing-box err (stop):", err.Error())
			return err
		}
	}
	rawConfig, err := s.GetConfig(string(config))
	if err != nil {
		logger.Error("restart sing-box err (get config):", err.Error())
		return err
	}
	if err := corePtr.Start(*rawConfig); err != nil {
		logger.Error("restart sing-box err (start):", err.Error())
		return err
	}
	logger.Info("sing-box restarted with new config")
	return nil
}

func (s *ConfigService) StopCore() error {
	err := corePtr.Stop()
	if err != nil {
		return err
	}
	logger.Info("sing-box stopped")
	return nil
}

func (s *ConfigService) CheckOutbound(tag string, link string) core.CheckOutboundResult {
	if tag == "" {
		return core.CheckOutboundResult{Error: "missing query parameter: tag"}
	}
	if corePtr == nil || !corePtr.IsRunning() {
		return core.CheckOutboundResult{Error: "core not running"}
	}
	return core.CheckOutbound(corePtr.GetCtx(), tag, link)
}

func (s *ConfigService) Save(obj string, act string, data json.RawMessage, initUsers string, loginUser string, hostname string) ([]string, error) {
	var err error
	var objs []string = []string{obj}

	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
			// Try to start core if it is not running
			if !corePtr.IsRunning() {
				s.StartCore()
			}
		} else {
			tx.Rollback()
		}
	}()

	switch obj {
	case "clients":
		var inboundIds []uint
		inboundIds, err = s.ClientService.Save(tx, act, data, hostname)
		if err == nil && len(inboundIds) > 0 {
			objs = append(objs, "inbounds")
			err = s.InboundService.RestartInbounds(tx, inboundIds)
			if err != nil {
				return nil, common.NewErrorf("failed to update users for inbounds: %v", err)
			}
		}
	case "tls":
		err = s.TlsService.Save(tx, act, data, hostname)
		objs = append(objs, "clients", "inbounds")
	case "inbounds":
		err = s.InboundService.Save(tx, act, data, initUsers, hostname)
		objs = append(objs, "clients")
	case "outbounds":
		err = s.OutboundService.Save(tx, act, data)
	case "endpoints":
		err = s.EndpointService.Save(tx, act, data)
	case "config":
		err = s.SettingService.SaveConfig(tx, data)
		if err != nil {
			return nil, err
		}
		configData := make(json.RawMessage, len(data))
		copy(configData, data)
		go func() { _ = s.restartCoreWithConfig(configData) }()
	case "settings":
		err = s.SettingService.Save(tx, data)
	default:
		return nil, common.NewError("unknown object: ", obj)
	}
	if err != nil {
		return nil, err
	}

	dt := time.Now().Unix()
	err = tx.Create(&model.Changes{
		DateTime: dt,
		Actor:    loginUser,
		Key:      obj,
		Action:   act,
		Obj:      data,
	}).Error
	if err != nil {
		return nil, err
	}

	LastUpdate = time.Now().Unix()

	return objs, nil
}

func (s *ConfigService) CheckChanges(lu string) (bool, error) {
	if lu == "" {
		return true, nil
	}
	if LastUpdate == 0 {
		db := database.GetDB()
		var count int64
		err := db.Model(model.Changes{}).Where("date_time > " + lu).Count(&count).Error
		if err == nil {
			LastUpdate = time.Now().Unix()
		}
		return count > 0, err
	} else {
		intLu, err := strconv.ParseInt(lu, 10, 64)
		return LastUpdate > intLu, err
	}
}

func (s *ConfigService) GetChanges(actor string, chngKey string, count string) []model.Changes {
	c, _ := strconv.Atoi(count)
	whereString := "`id`>0"
	if len(actor) > 0 {
		whereString += " and `actor`='" + actor + "'"
	}
	if len(chngKey) > 0 {
		whereString += " and `key`='" + chngKey + "'"
	}
	db := database.GetDB()
	var chngs []model.Changes
	err := db.Model(model.Changes{}).Where(whereString).Order("`id` desc").Limit(c).Scan(&chngs).Error
	if err != nil {
		logger.Warning(err)
	}
	return chngs
}
