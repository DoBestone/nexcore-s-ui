package database

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database/model"
	logr "github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/util/common"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// initUser 首次启动建首位管理员。
//
// 安全模型(AUDIT.md C2):**不再硬编码 admin/admin**。
//   - 库为空 → 生成 16 字符高熵随机密码,bcrypt 落库,**明文一次性 stdout**(运维必读),
//     用户名固定 "admin"(可在 sui CLI / 面板里改)。
//   - 已有用户 → 跳过。
//
// 这样即使用户跳过 install.sh 的 `sui admin -username/-password` 步骤直接 `go build && ./sui`,
// 也不会让公网起一台 admin/admin 的面板;启动日志强制运维看到密码至少一次。
func initUser() error {
	var count int64
	err := db.Model(&model.User{}).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	plain := common.Random(16)
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), 12)
	if err != nil {
		return err
	}
	user := &model.User{
		Username: "admin",
		Password: string(hashed),
	}
	if err := db.Create(user).Error; err != nil {
		return err
	}
	// 顶头高亮:启动日志一次性打印初始凭据。运维错过这次就只能 `sui admin -username admin -password XXX` 重置。
	logr.Info("================================================================")
	logr.Info("  首次启动 — 管理员初始凭据(本次启动只显示一次,务必抄走!)")
	logr.Info("    username : admin")
	logr.Info("    password : ", plain)
	logr.Info("  改密码:登录面板「设置 → 修改密码」 或 CLI `sui admin -username admin -password 新密码`")
	logr.Info("================================================================")
	return nil
}

func OpenDB(dbPath string) error {
	dir := path.Dir(dbPath)
	// AUDIT.md LOW:0o700 = owner rwx,其他无访问 — DB 只 sui 进程读,不需要
	// group/other 任何位。原 01740 含 sticky bit + group r,过松。
	err := os.MkdirAll(dir, 0o700)
	if err != nil {
		return err
	}

	var gormLogger logger.Interface

	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	c := &gorm.Config{
		Logger: gormLogger,
	}
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}
	// AUDIT.md H1:启用 SQLite foreign_key 强制(默认是 off,DDL 里写的
	// `FOREIGN KEY ... REFERENCES tls(id)` 一直没真的生效)。
	// 现在删 TLS 时若有 inbound 引用,DB 层兜一层错,不再依赖 service 层
	// 显式 "tls in use" 检查的鲁棒性。
	dsn := dbPath + sep + "_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"
	db, err = gorm.Open(sqlite.Open(dsn), c)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if config.IsDebug() {
		db = db.Debug()
	}
	return nil
}

func InitDB(dbPath string) error {
	err := OpenDB(dbPath)
	if err != nil {
		return err
	}

	// Default Outbounds
	if !db.Migrator().HasTable(&model.Outbound{}) {
		db.Migrator().CreateTable(&model.Outbound{})
		defaultOutbound := []model.Outbound{
			{Type: "direct", Tag: "direct", Options: json.RawMessage(`{}`)},
		}
		db.Create(&defaultOutbound)
	}

	err = db.AutoMigrate(
		&model.Setting{},
		&model.Tls{},
		&model.Inbound{},
		&model.Outbound{},
		&model.Endpoint{},
		&model.User{},
		&model.Tokens{},
		&model.Stats{},
		&model.Client{},
		&model.Changes{},
		&model.ApiLog{},
		&model.BlockRule{},
	)
	if err != nil {
		return err
	}
	// 老数据库新加的 inbounds.enable 列对历史行可能落到 NULL;
	// 显式回填一次,确保升级后所有现存入站默认是启用状态。
	db.Exec("UPDATE inbounds SET enable = 1 WHERE enable IS NULL")
	err = initUser()
	if err != nil {
		return err
	}

	return nil
}

func GetDB() *gorm.DB {
	return db
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
