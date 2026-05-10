# 全面审计报告 · nexcore-s-ui v1.7.10

> 生成日期:2026-05-10
> 范围:Go 后端(14.5K LOC)+ Vue/EP 前端(19.8K LOC)+ 构建/部署/数据库
> 方法:并行 5 个专项子代理(安全 / 后端 / 前端 / 构建 / 数据库)→ 主代理交叉验证 + 重新定级

---

## 🔴 CRITICAL(立刻修)

### 1. 密码 / Token 明文存数据库
- `database/model/model.go` User.Password / Tokens.Token 都是 `string` 列,**未哈希**
- `service/user.go:63` 登录用 `WHERE username = ? and password = ?` 直接比对明文
- `service/user.go:151` `Token: common.Random(32)` 生成后**原样落库**,无 sha256/bcrypt
- 影响:DB 文件一旦被泄(误备份 / 误打包 / 攻击拿到 SQLite),所有凭证瞬间失守,token 无法防御
- 修复:登录走 bcrypt(cost ≥ 10);token 落库前 `sha256()`,客户端拿到的明文 token 只在生成那一刻返回一次

### 2. 默认管理员凭证 admin / admin 硬编码
- `database/db.go:initUser()` — 用户表为空时建 `Username:"admin", Password:"admin"`
- 如果用户绕过 `install.sh`(直接 `go build && ./sui`)或 `install.sh` 的 `sui admin -username/-password` 步骤失败,公网就是 admin/admin
- 修复:首次启动如果没有人调过 `sui admin`,**拒绝启动**并打印一次性引导(或随机生成密码并写日志/stdout 一次)

### 3. Session Cookie 缺关键安全标志
- `api/session.go:21-24`:`Secure: false`,`HttpOnly` / `SameSite` 都没设
- `gorilla/sessions` 的默认 `HttpOnly` 是 false(不是 true),即 cookie 可被前端 JS 读到 → XSS 偷 session
- 修复:`HttpOnly: true`、`SameSite: http.SameSiteLaxMode`、HTTPS 时 `Secure: true`(可根据 `webCertFile` 是否非空动态判断)

### 4. 登录无任何防爆破
- `service/user.go:Login` 没有失败计数 / IP 节流 / 延迟
- 修复:每 IP 失败 ≥ 5 次,加 5s/15s/60s 阶梯延迟,并写审计日志

### 5. `util/genLink.go` 多处类型断言无 ok 检查 → 可被坏数据 panic
- 例如 `oTls["reality"].(map[string]interface{})`、`addr["server"].(string)`、`alpnList[i] = v.(string)` 等
- 客户端 / 入站若有部分协议不带 reality 或 alpn 中混入非字符串,生成链接会 panic 整个 HTTP 进程(gin 默认 recover,但请求直接 500)
- 修复:全部改成 `if x, ok := v.(T); ok {...}` 形式

---

## 🟠 HIGH

| # | 问题 | 位置 | 一句话修复 |
|---|---|---|---|
| H1 | SQLite `PRAGMA foreign_keys` 没开 | `database/db.go:58` DSN | DSN 追加 `&_foreign_keys=on` |
| H2 | `clients.inbounds` json 列被 `json_each()` 当数组扫,空/坏 JSON 静默返 0 行,导致孤儿 client 误删 / 入站删除残留 | `service/client.go:343,351,399…` 多处 `json.Unmarshal` 不 check err | 加 err 处理,坏数据走告警分支不走"什么都没发生" |
| H3 | `service/config.go` 重启 sing-box 的 goroutine 无 ctx 取消,`lastStartFailTime` 读写无锁 race | `service/config.go:259,271,123,145` | sync.WaitGroup + 把 `lastStartFailTime` 走 atomic |
| H4 | Cron 任务 Stop 不等飞行中事务结束 | `cronjob/cronJob.go:40` | `cron.Stop()` 返回的 ctx.Done() 上 `Wait` |
| H5 | settings 表 key 没 UNIQUE 索引,saveSetting 是 read-then-write 竞态 | `database/model/model.go:7` + `service/setting.go:169` | 加 UNIQUE(key) + 改用 UPSERT |
| H6 | systemd unit 零加固,以 root 运行,无 `ProtectSystem` / `NoNewPrivileges` 等 | `nexcore-s-ui.service` + `install.sh:379-396` 兜底 unit | 加上述两个 + `PrivateTmp=true`,sing-box 的 `CAP_NET_ADMIN` 用 `AmbientCapabilities` 单独给 |
| H7 | `Logs.vue:28` `v-html="line"` 渲染后端日志 | `frontend/src/layouts/modals/Logs.vue:28` | 改 `{{ line }}` + CSS `white-space: pre-wrap`(日志里只有少量 ANSI 颜色,可走前端解析后用具名 class) |
| H8 | `entrypoint.sh` 写死 `/app/db/s-ui.db` 老路径 | 顶层 `entrypoint.sh` | docker 都已删 → 这个文件也该删,不然将来谁拷它会出鬼故障 |

---

## 🟡 MED(质量 / 健壮性)

- Session `MaxAge` 默认 `0`(永不过期)— `service/setting.go:54` → 至少给 7 天默认
- 订阅 `?host=` 入参不校验 — `api/v1/subscription.go:131` → 白名单或至少正则
- `stats(resource, tag, date_time)` 无复合索引,`settings(key)` 无索引 — 量起来后慢
- `service/client.go:611` `ResetDays ≤ 0` 时 `NextReset` 会被算成过去时间,陷入即时重置循环 — 加保护
- `prepareTls`、`buildLinkRemarkCtx` 静默吞 unmarshal err
- `database/backup.go` 未 `BEGIN IMMEDIATE` 拍快照,WAL 期间撕裂可能性
- `Settings.vue` 大量硬编码中文字符串(`"节点名称"`、`"分享链接域名来源"` 等),没走 `$t(...)` — 即便只两种语言,英文用户也看不到对应翻译
- `service/inbounds.go:239` `UpdateOutJsons` 循环更新无统一 tx,中途失败半破坏

---

## 🟢 LOW(打磨)

- `Login.vue:159` 等多处 `setTimeout(..., 350)` 凑动画时序
- `AppBar.vue` 还有原生 `<button>`,跟其它地方 `<el-button>` 不齐
- 表单必填项缺 `*` 标记
- `database/db.go:38` `os.MkdirAll(dir, 01740)` → 改 `0o700` 更干净
- `nexcore-s-ui.sh` 菜单里 `bash <(curl ... install.sh)` 的 pipe 语义,文档要说清

---

## ⚠️ 子代理判定修正记录

| Agent 报的 | 主代理修正 |
|---|---|
| 前端 Agent:**i18n 只剩 2 语言 = CRITICAL 违反 CLAUDE.md** | 实际:`locales/index.ts` 顶部注释 `"只剩两种语言,直接全量同步加载"` 表明是**主动收窄**。结论:不是回退,而是 **CLAUDE.md 第 8 条已与现状脱节**,要么改 CLAUDE.md 把 6 语言 → 2 语言,要么补回 4 语言。当前不算 bug,定级 INFO。 |
| 安全 Agent:**db dir 0o740 = CRITICAL** | `01740` 是 sticky + 740,owner 完全控制,group 只读,other 无访问。**HIGH 都不到**,放 LOW。 |
| 安全 Agent:**CSRF 是 CRITICAL** | 当前 panel 一般部署在 path 加随机后缀(`install.sh` 的 `random_slug`)+ 短期内非高价值目标,CSRF 实战门槛较高;但配合上面"无 SameSite"会放大风险。先把 SameSite 加上,CSRF 放 HIGH。 |
| 后端 Agent:多处 `json.Unmarshal` 不 check err = CRITICAL | 大部分位置有上下文 fallback,真正会导致**误删 client / 误清孤儿**的就是 H2 那条。其余降 MED。 |

---

## ✅ 验证干净的地方(可跳过)

- Token 验证用 `subtle.ConstantTimeCompare`(`api/v1/middleware.go:95-114`)
- `install.sh` SHA256 校验 + HTTPS 拉包 + 安装后落地权限
- `update.sh` 不动 DB 目录,systemd unit 备份
- 前端 `package.json` 已无 vuetify / mdi / notivue / moment 残留(CLAUDE.md 第 6 条达成)
- `docker` 残留代码已彻底清理(只剩 entrypoint.sh 这个孤儿,见 H8)
- `vue-tsc` / vite build 通过,bundle 主入口 213KB(gzip 79KB)
- 链接生成 `LinkAddrSource = panel|tls` 双分支逻辑正确
- 所有迁移脚本 `AutoMigrate` 不 drop 数据
- 版本号 1.7.10 在 `config/version` / 二进制 / CLI `-v` 三处一致

---

## 建议修复顺序(投入产出比)

**周内必做**(攻击面直接闭合):
1. 哈希密码(bcrypt)+ 哈希 token 落库 + 旧明文一次性迁移脚本(CRITICAL #1)
2. 拒启动 admin/admin(CRITICAL #2)
3. session cookie HttpOnly + SameSite + 动态 Secure(CRITICAL #3)
4. 登录失败节流(CRITICAL #4)
5. `genLink.go` 类型断言全加 ok(CRITICAL #5)

**两周内**:H1–H4(数据一致性 / 进程稳定性) + H7 v-html

**有空时**:H5–H8 + 全部 MED + LOW
