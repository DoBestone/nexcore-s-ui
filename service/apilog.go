package service

import (
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
)

type ApiLogService struct{}

// Add 记录一条 API 调用。失败时静默 — 日志写不进去也不能拖崩业务调用。
func (s *ApiLogService) Add(entry *model.ApiLog) {
	db := database.GetDB()
	if db == nil {
		return
	}
	if entry.DateTime == 0 {
		entry.DateTime = time.Now().Unix()
	}
	_ = db.Create(entry).Error
}

// List 分页查询。method / path / username 任一非空时按精确(path 用 like)过滤;
// since/until 是 unix 秒,0 表示不过滤。返回 (logs, total)。
func (s *ApiLogService) List(method, path, username string, since, until int64, limit, offset int) ([]model.ApiLog, int64, error) {
	db := database.GetDB()
	q := db.Model(&model.ApiLog{})
	if method != "" {
		q = q.Where("method = ?", method)
	}
	if path != "" {
		q = q.Where("path LIKE ?", "%"+path+"%")
	}
	if username != "" {
		q = q.Where("username = ?", username)
	}
	if since > 0 {
		q = q.Where("date_time >= ?", since)
	}
	if until > 0 {
		q = q.Where("date_time <= ?", until)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	var logs []model.ApiLog
	err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// Clear 清空所有日志(管理员手动维护用)。
func (s *ApiLogService) Clear() error {
	db := database.GetDB()
	return db.Exec("DELETE FROM api_logs").Error
}

// PruneOlderThan 删除 unix 秒之前的日志,后台 cron 用。
func (s *ApiLogService) PruneOlderThan(ts int64) error {
	db := database.GetDB()
	return db.Where("date_time < ?", ts).Delete(&model.ApiLog{}).Error
}
