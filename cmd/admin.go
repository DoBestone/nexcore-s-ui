package cmd

import (
	"fmt"
	"strings"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util/common"
)

// resetAdmin 把 first user 重置成 admin + 16 字符随机密码,**stdout 一次性
// 打印明文** — install.sh / 运维 menu "Reset admin credentials to default"
// 都依赖这个明文输出(install.sh 拿来当 FRESH_PASS 显示给用户)。
//
// AUDIT.md C2 之前是硬编码 admin/admin,改成随机密码。
func resetAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	plain := common.Random(16)
	userService := service.UserService{}
	if err := userService.UpdateFirstUser("admin", plain); err != nil {
		fmt.Println("reset admin credentials failed:", err)
		return
	}
	// stdout 一次性可见,机器可解析 — install.sh 用 grep 拿
	fmt.Println("reset admin credentials success")
	fmt.Println("\tUsername:\tadmin")
	fmt.Println("\tPassword:\t" + plain)
}

func updateAdmin(username string, password string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	if username != "" || password != "" {
		userService := service.UserService{}
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("reset admin credentials failed:", err)
		} else {
			fmt.Println("reset admin credentials success")
		}
	}
}

// showAdmin 显示 first user 的用户名 + 密码状态。
//
// AUDIT.md C1 后 password 列存 bcrypt(`$2a$...$60chars`),**不能解出明文**。
// 检测到 hash 时改打印"已 bcrypt 哈希,无法显示明文 — 要重置请跑 sui admin
// -username admin -password 新密码 或 sui admin -reset"。
//
// 历史(v1.7.12 之前)password 是明文,这命令直接 print 列值 — 用户、
// install.sh 都依赖这个行为读密码。现在 print bcrypt hash 是个迷惑信息
// (用户拿 hash 去登录会失败),所以必须显式说明。
func showAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}
	userService := service.UserService{}
	userModel, err := userService.GetFirstUser()
	if err != nil {
		fmt.Println("get current user info failed,error info:", err)
		return
	}
	if userModel.Username == "" {
		fmt.Println("current admin username is empty")
		return
	}
	fmt.Println("First admin credentials:")
	fmt.Println("\tUsername:\t" + userModel.Username)
	if isBcryptStored(userModel.Password) {
		fmt.Println("\tPassword:\t<bcrypt-hashed,无法显示明文>")
		fmt.Println("\t                要重置:sui admin -reset(随机密码)或 sui admin -username admin -password <新密码>")
	} else if userModel.Password == "" {
		fmt.Println("\tPassword:\t<empty>")
	} else {
		// 老 DB 升级前的明文遗留(理论上 service.UpgradePlaintextPasswords
		// 启动时已升 bcrypt,这分支留给未跑过升级或手工写 DB 的边角场景)
		fmt.Println("\tPassword:\t" + userModel.Password + "  (legacy plaintext — 启 panel 会自动升级 bcrypt)")
	}
}

// isBcryptStored 判别 password 列是否已 bcrypt 化。bcrypt hash 永远 60 字符,
// 以 $2a$/$2b$/$2y$ 起头 — 跟 service.isBcryptHash 保持一致(那是 service
// 包私有,这里复制一份避免 cmd 反向 import service 私有判别)。
func isBcryptStored(s string) bool {
	if len(s) != 60 {
		return false
	}
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}
