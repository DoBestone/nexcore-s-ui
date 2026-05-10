package service

import (
	"encoding/json"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/util/common"

	"gorm.io/gorm"
)

type TlsService struct {
	InboundService
}

// stripACMEKeyType 删掉 server.acme.key_type 字段。sing-box 1.13.5+ 的
// option.InboundACMEOptions 不再有 KeyType,reload 时见到该字段会因
// strict-unmarshal 报 "unknown field key_type" 让面板保存失败。
// 旧版默认 ec256(P256/ECDSA),sing-box 现在走 certmagic 默认值(也是 P256),
// 行为等价,删掉无副作用。
func stripACMEKeyType(serverRaw json.RawMessage) json.RawMessage {
	if len(serverRaw) == 0 {
		return serverRaw
	}
	var srv map[string]interface{}
	if err := json.Unmarshal(serverRaw, &srv); err != nil {
		return serverRaw
	}
	acme, ok := srv["acme"].(map[string]interface{})
	if !ok {
		return serverRaw
	}
	if _, has := acme["key_type"]; !has {
		return serverRaw
	}
	delete(acme, "key_type")
	srv["acme"] = acme
	out, err := json.Marshal(srv)
	if err != nil {
		return serverRaw
	}
	return out
}

func (s *TlsService) GetAll() ([]model.Tls, error) {
	db := database.GetDB()
	tlsConfig := []model.Tls{}
	err := db.Model(model.Tls{}).Scan(&tlsConfig).Error
	if err != nil {
		return nil, err
	}

	return tlsConfig, nil
}

func (s *TlsService) Save(tx *gorm.DB, action string, data json.RawMessage, hostname string) error {
	var err error

	switch action {
	case "new", "edit":
		var tls model.Tls
		err = json.Unmarshal(data, &tls)
		if err != nil {
			return err
		}
		// sing-box 1.13.5+ 删除了 acme.key_type — 写库前 strip 掉,
		// 否则 reload sing-box 会报 unknown field "key_type"。
		// 老配置(v1.7.4 时代签发的)/ 用户粘贴的 schema 都可能带这个字段。
		tls.Server = stripACMEKeyType(tls.Server)
		err = tx.Save(&tls).Error
		if err != nil {
			return err
		}
		if action == "edit" {
			var inbounds []model.Inbound
			err = tx.Model(model.Inbound{}).Preload("Tls").Where("tls_id = ?", tls.Id).Find(&inbounds).Error
			if err != nil {
				return err
			}
			if len(inbounds) > 0 {
				err = s.ClientService.UpdateLinksByInboundChange(tx, &inbounds, hostname, "")
				if err != nil {
					return err
				}
				var inboundIds []uint
				for _, inbound := range inbounds {
					inboundIds = append(inboundIds, inbound.Id)
				}
				err = s.InboundService.UpdateOutJsons(tx, inboundIds, hostname)
				if err != nil {
					return common.NewError("unable to update out_json of inbounds: ", err.Error())
				}
				err = s.InboundService.RestartInbounds(tx, inboundIds)
				if err != nil {
					return err
				}
			}
		}
	case "del":
		var id uint
		err = json.Unmarshal(data, &id)
		if err != nil {
			return err
		}
		var inboundCount int64
		err = tx.Model(model.Inbound{}).Where("tls_id = ?", id).Count(&inboundCount).Error
		if err != nil {
			return err
		}
		if inboundCount > 0 {
			return common.NewError("tls in use")
		}
		err = tx.Where("id = ?", id).Delete(model.Tls{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}
