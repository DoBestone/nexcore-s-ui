package cmd

// `sui token` —— install.sh / 运维直接生成 / 列出 / 删除 admin scope API token,
// 不依赖 panel HTTP(免去 login + DomainValidator 的麻烦)。
//
// 用法:
//   sui token -add  -desc "installer-bootstrap"   # 生成新 token,stdout 打印
//   sui token -list                               # 列出所有 token (id desc 前缀)
//   sui token -del  -id 3                         # 删除指定 id

import (
	"fmt"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/service"
)

// addToken 给 first user 绑一个新 token,returns 完整 token 字符串(明文,只此一次)。
// expiry: 0 = 永不过期。
func addToken(desc string, expiry int64) {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("db init failed:", err)
		return
	}
	userSvc := service.UserService{}
	user, err := userSvc.GetFirstUser()
	if err != nil || user == nil || user.Username == "" {
		fmt.Println("first user not found — run `sui admin -username ... -password ...` first")
		return
	}
	tok, err := userSvc.AddToken(user.Username, expiry, desc)
	if err != nil {
		fmt.Println("add token failed:", err)
		return
	}
	// 只打印 token 字符串到 stdout(不带任何前缀),给 install.sh
	// `TOKEN=$(sui token -add -desc x)` 直接吃。
	fmt.Println(tok)
}

func listTokens() {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("db init failed:", err)
		return
	}
	userSvc := service.UserService{}
	user, err := userSvc.GetFirstUser()
	if err != nil || user == nil {
		fmt.Println("first user not found")
		return
	}
	tokens, err := userSvc.GetUserTokens(user.Username)
	if err != nil {
		fmt.Println("list tokens failed:", err)
		return
	}
	fmt.Printf("%-4s  %-20s  %-12s  %s\n", "ID", "DESC", "TOKEN-PREFIX", "EXPIRY")
	for _, t := range *tokens {
		prefix := t.Token
		if len(prefix) > 12 {
			prefix = prefix[:12] + "…"
		}
		fmt.Printf("%-4d  %-20s  %-12s  %d\n", t.Id, t.Desc, prefix, t.Expiry)
	}
}

func deleteToken(id string) {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("db init failed:", err)
		return
	}
	userSvc := service.UserService{}
	if err := userSvc.DeleteToken(id); err != nil {
		fmt.Println("delete token failed:", err)
		return
	}
	fmt.Println("ok")
}
