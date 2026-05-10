package service

import (
	"encoding/json"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/util/common"

	"gorm.io/gorm"
)

// BlockRuleService 管理 model.BlockRule 表 — 跟 ConfigService.Save 的
// "obj=block-rules" 分支配对,act = new / edit / del,data 见各 case 注释。
//
// 写入后不直接重启 sing-box(由 ConfigService.Save 末尾统一异步触发,详见
// service/config.go 的 obj switch)。
type BlockRuleService struct{}

func (s *BlockRuleService) GetAll() ([]model.BlockRule, error) {
	var rules []model.BlockRule
	err := database.GetDB().Order("id ASC").Find(&rules).Error
	if rules == nil {
		rules = []model.BlockRule{}
	}
	return rules, err
}

// Save 兼容 sui 内部 save 风格:
//   act=new   data = BlockRule(无 id)
//   act=edit  data = BlockRule(必带 id)
//   act=del   data = []uint(id 列表)
func (s *BlockRuleService) Save(tx *gorm.DB, act string, data json.RawMessage) error {
	switch act {
	case "new":
		var r model.BlockRule
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		r.Id = 0
		return tx.Create(&r).Error
	case "edit":
		var r model.BlockRule
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		if r.Id == 0 {
			return common.NewError("block-rules edit: id required")
		}
		return tx.Save(&r).Error
	case "del":
		var ids []uint
		if err := json.Unmarshal(data, &ids); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		return tx.Where("id IN ?", ids).Delete(&model.BlockRule{}).Error
	}
	return common.NewError("block-rules unknown act: ", act)
}
