# nexcore-s-ui

**NexCore Sing-Box 控制面板 — 无人值守部署、全量 REST API、Cloudflare 自动签证书**

[![Go Report Card](https://goreportcard.com/badge/github.com/DoBestone/nexcore-s-ui)](https://goreportcard.com/report/github.com/DoBestone/nexcore-s-ui)
[![Release](https://img.shields.io/github/v/release/DoBestone/nexcore-s-ui?logo=github)](https://github.com/DoBestone/nexcore-s-ui/releases)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

NexCore 产品线下的 sing-box 控制面板。前端 Vue 3 + Element Plus,后端 Go,单二进制部署。一键 `install.sh` 自动生成随机管理员凭据,**TLS 证书 Cloudflare DNS-01 全自动**,REST API 完整覆盖面板能力且与 [`nexcore-x-ui`](https://github.com/DoBestone/nexcore-x-ui) 主控互通。

> **声明:** 仅供个人学习与技术研究,请勿用于非法用途。

---

## v1.0.0 亮点

- **无人值守部署** —— `install.sh` / `update.sh` 一键完成;首装自动生成随机管理员凭据并打印,无需任何交互。
- **全量 REST API** —— 面板上能做的所有操作都有 API 入口,支持 Bearer Token 鉴权 + 调用日志审计。
- **Cloudflare 自动 TLS** —— 凭一个 CF API Token,自动加 DNS A 记录(随机 / 自定义前缀)+ 自动走 ACME DNS-01 签发并续签证书,sing-box 启动时即可用。
- **`nexcore-x-ui` 兼容层** —— `/api/v1/*` 完全镜像 [`nexcore-x-ui`](https://github.com/DoBestone/nexcore-x-ui) 的 REST 形态(同路径布局 / 同鉴权头 / 同响应壳 / 同状态码),为 x-ui 写的主控对接代码可直接指向本节点,**无需改一行**。
- **路由模板** —— 一键应用屏蔽广告 / 恶意 / 钓鱼 / 中国大陆直连 / 私有 IP 直连等常用规则集。
- **前端 API 控制台** —— Token 管理 / 端点速查 / 调用日志 三 tab 集中查看。

完整变更见 [v1.0.0 release notes](https://github.com/DoBestone/nexcore-s-ui/releases/tag/v1.0.0)。

---

## 功能速查

| 功能                                          |       状态       |
| --------------------------------------------- | :--------------: |
| 多协议(VLESS / VMess / Trojan / Hysteria 等) |        ✅        |
| 多语言(en / fa / vi / 中 / 繁 / ру)         |        ✅        |
| 多客户端 / 多入站                              |        ✅        |
| 高级路由规则 + **一键模板**                    |        ✅        |
| 在线状态、流量、系统监控                        |        ✅        |
| 订阅链接(原生 / json / clash + 元信息)       |        ✅        |
| 浅色 / 深色主题                                |        ✅        |
| **REST API + Bearer Token + 审计日志**        |        ✅        |
| **`nexcore-x-ui` 兼容 REST**(/api/v1)       |        ✅        |
| **Cloudflare 自动签发 TLS**                   |        ✅        |

## 支持平台

| 平台 | 架构 | 状态 |
|----------|--------------|---------|
| Linux    | amd64, arm64, armv7, armv6, armv5, 386, s390x | ✅ |
| Windows  | amd64, 386, arm64 | ✅ |
| macOS    | amd64, arm64 | 🚧 实验性 |

## 默认安装信息

- 面板端口:`3095`(刻意与上游 `s-ui` 的 2095 错开,可同机共存)
- 面板路径:`/app/`
- 订阅端口:`3096`(刻意与上游 `s-ui` 的 2096 错开)
- 订阅路径:`/sub/`
- 管理员凭据:首装由 `install.sh` **自动随机生成并打印**(立即记录;之后只能用 `nexcore-s-ui` 菜单重置)

### 与上游 `alireza0/s-ui` 的隔离

`nexcore-s-ui` 在路径 / 服务名 / 数据库 / 端口 / 命令名上与上游 `s-ui` **完全独立**,可在同一台机器同时安装、互不干扰。

| 维度 | 上游 `s-ui` | `nexcore-s-ui` |
|---|---|---|
| 安装目录 | `/usr/local/s-ui/` | `/usr/local/nexcore-s-ui/` |
| 数据库 | `/usr/local/s-ui/db/s-ui.db` | `/usr/local/nexcore-s-ui/db/nexcore-s-ui.db` |
| systemd 服务 | `s-ui.service` | `nexcore-s-ui.service` |
| 管理命令 | `/usr/bin/s-ui` | `/usr/bin/nexcore-s-ui` |
| 默认面板端口 | 2095 | **3095** |
| 默认订阅端口 | 2096 | **3096** |
| 浏览器 cookie | `s-ui` | `nexcore-s-ui` |

---

## 一键部署(Linux)

`install.sh` 与 `update.sh` 都从 **本仓库 release**(`DoBestone/nexcore-s-ui`)拉对应版本的 tarball,带 SHA256 校验,无需任何交互。

### 全新安装(最新版)

```sh
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/install.sh)
```

### 安装指定版本

```sh
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/install.sh) v1.0.0
```

### 已装过 → 强制重装(覆盖二进制,保留 `db/`)

```sh
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/install.sh) --force
```

### 升级到最新版

```sh
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/update.sh)
# 装完面板后等价命令:
nexcore-s-ui update
```

`update.sh` 只替换 sui 二进制 + `bin/` + service unit,**`/usr/local/nexcore-s-ui/db/` 永远不动**(数据库完整保留)。

### 升级到指定版本

```sh
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/update.sh) v1.0.0
```

### 部署完成提示

`install.sh` 跑完会直接在终端打印:

```
═════════════════════════════════════════════
  nexcore-s-ui 已部署
═════════════════════════════════════════════
Current panel settings:
        Panel port:      3095
        Panel path:      /app/
        ...
访问地址:
http://<你的 IP>:3095/app/
首装明文凭据 (★ 立即记录,后续只能 nexcore-s-ui 菜单重置):
  用户名: admin_a3f9k2
  密 码:  9KqL4mPzN2vR
```

---

## REST API

完整文档:**面板 → API 管理 → API 文档** tab,内置基础 URL + curl 示例 + 全部端点速查。

### 三套 API 共存

| 前缀 | 鉴权 | 用途 |
|---|---|---|
| `/api/*` | session cookie | 面板自身 UI 调用 |
| `/apiv2/*` | Bearer Token / X-API-Token / Token 头 | 通用脚本对接,响应壳与 v1 风格(`{success, msg, obj}`) |
| `/api/v1/*` | Bearer Token / X-API-Token | **`nexcore-x-ui` 兼容**,`{data}` / `{error,code,message}` 壳 |

### 快速上手

```sh
# 1) 在面板 → API 管理 → 新建 Token (复制保存,仅显示一次)
TOKEN="<your-token>"
HOST="http://your-server:3095/app"

# 2) 健康检查(无需鉴权)
curl $HOST/api/v1/health
# → {"data":{"impl":"nexcore-s-ui","status":"ok","time":1778279413710}}

# 3) 校验 token
curl -H "Authorization: Bearer $TOKEN" $HOST/api/v1/me
# → {"data":{"username":"admin","tokenDesc":"my-token"}}

# 4) 服务器状态
curl -H "Authorization: Bearer $TOKEN" $HOST/api/v1/server/status

# 5) 列出入站
curl -H "Authorization: Bearer $TOKEN" $HOST/api/v1/inbounds

# 6) 改面板设置
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -X PATCH -d '{"subPort":2099}' $HOST/api/v1/settings
```

### `/api/v1` 端点全览(40 项)

详见面板内"API 文档"tab。核心分组:

- **健康/身份**: `GET /health`, `GET /me`, `GET /server/status`
- **sing-box**: `GET /xray/{status,logs,config}`、`POST /xray/restart`(命名空间 `xray` 用于兼容 x-ui 主控;`/singbox/*` 同义)
- **入站/出站/端点/服务**: 全 REST CRUD(`GET/POST/PUT/DELETE`)
- **客户端**: `GET /clients`, `GET /clients/:identifier/traffic`, `POST /clients/:identifier/reset-traffic`
- **在线/流量**: `GET /online-ips[/:tag]`, `GET /onlines`, `GET /traffic[/live]`
- **审计**: `GET /access-logs`, `DELETE /access-logs`
- **设置/Token/系统**: `GET/PATCH /settings`, `GET/POST/DELETE /tokens`, `POST /system/restart-panel`
- **s-ui 独有**(命名空间 `/sui/*`): Cloudflare zones/dns/tls 三件套、原生 sing-box 完整配置下载、订阅 URI

### 与 `nexcore-x-ui` 主控对接

主控对接代码无需修改:

```diff
- HOST=https://x-node.example.com/api/v1
+ HOST=https://s-node.example.com/app/api/v1
  # 同样的 Authorization: Bearer <token>
  # 同样的 {data} / {error,code,message} 响应壳
  # 同样的 HTTP 状态码语义
```

字段差异(`inbounds[].settings` 等 sing-box 与 xray schema 不同)在主控渲染时按节点 `impl` 字段(`/health` 返回)自适应即可。

---

## Cloudflare 自动签发 TLS

**面板 → TLS 设置 → Cloudflare 一键签发**,3 步完成:

1. **Token + 邮箱** —— 粘贴 Cloudflare API Token(权限 `Zone:DNS:Edit + Zone:Read`,Global Key 也可),填 ACME 注册邮箱。
2. **DNS** —— 选根域、选前缀策略(随机 / 自定义 / 根域)、填公网 IP(可一键自动获取)、决定是否走 CF 反代。
3. **签发** —— 给 TLS 配置取个名,提交。

幕后:面板调 CF API 加 A 记录 → 写入一条 `model.Tls`(server JSON 内嵌 `acme + dns01_challenge.cloudflare`)→ sing-box 启动时由内置 `with_acme` ACME 客户端自动走 DNS-01 挑战签发证书,后续自动续签。

**Token 不持久化** —— 只下发给 sing-box 内嵌使用,面板侧不存。

API 调用版本(任选 v1 或 v2):

```sh
# 列出 token 可见的 zone
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -X POST -d '{"token":"<cf_token>"}' \
     $HOST/api/v1/sui/cloudflare/zones

# 自动加 A 记录(随机前缀)
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -X POST -d '{"token":"<cf_token>","zoneId":"<id>","random":true,"prefix":"node1","ip":"1.2.3.4"}' \
     $HOST/api/v1/sui/cloudflare/dns/upsert-a

# 生成内嵌 ACME 配置的 TLS 记录
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -X POST -d '{"name":"cf-auto","fqdn":"node1-x.example.com","email":"a@b.c","token":"<cf_token>"}' \
     $HOST/api/v1/sui/cloudflare/tls/issue
```

---

## 从源码构建

### 前置依赖

- **Go** 1.25+
- **Node.js** 20+ / npm
- C 编译器(CGO 需要,如 `gcc` / `clang`)

### 一键构建

```sh
git clone https://github.com/DoBestone/nexcore-s-ui.git
cd nexcore-s-ui
./build.sh
```

`build.sh` 会:

1. `cd frontend && npm i && npm run build` —— 输出 `frontend/dist/`
2. `cp -R frontend/dist/* web/html/` —— 镜像静态资源到 Go embed 目录
3. `go build -ldflags '-w -s -checklinkname=0 ...' -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale" -o sui main.go`

最终产物:`./sui`(Linux / macOS arm64 约 75 MB)。

### 运行

```sh
./sui
```

默认监听 `3095`,浏览器打开 `http://127.0.0.1:3095/app/`。开发期临时账号通过 CLI 创建:

```sh
./sui admin -username admin -password <随机串>
./sui admin -show
```

生产环境推荐走 `install.sh`,首装会自动随机生成并打印到终端。

---

## 仅开发前端

```sh
cd frontend
npm i
npm run dev      # vite dev server, 默认 :3000,代理 /app/api → :3095
```

打开 `http://127.0.0.1:3000/`(后端需另开 `./sui`)。

### 前端技术栈

- **框架**:Vue 3.5 + TypeScript
- **组件库**:Element Plus(按需注册,组件 CSS 自动按需注入)
- **构建**:Vite 8 + unplugin-vue-components(ElementPlusResolver)+ unplugin-auto-import
- **状态**:Pinia
- **路由**:Vue Router 4
- **i18n**:vue-i18n(异步按需加载语言包)
- **图标**:`@element-plus/icons-vue`(线框风格,16 / 18 / 20 三档)
- **图表**:chart.js + vue-chartjs(仅 Stats 模态加载)
- **二维码**:qrcode.vue(仅 QrCode / WgQrCode 模态加载)

### 构建产物路径契约

```
frontend/dist/  ──cp──>  web/html/  ──go:embed *──>  Go binary
```

`web/web.go` 通过 `embed.FS` 把 `web/html/index.html` 与 `web/html/assets/*` 打包进二进制,资源文件名用 8 字节随机 hash 防 CDN 缓存。**不要修改输出路径**,否则 Go embed 找不到资源。

---

## 卸载

```sh
sudo -i
systemctl disable nexcore-s-ui --now
rm -f /etc/systemd/system/nexcore-s-ui.service
systemctl daemon-reload
rm -fr /usr/local/nexcore-s-ui
rm -f /usr/bin/nexcore-s-ui
```

> 注:`/usr/local/nexcore-s-ui/db/` 是数据库目录(订阅、客户端、入站、TLS 等),`rm -fr /usr/local/nexcore-s-ui` 会一并清掉。如需保留请先单独备份。
>
> 上游 `s-ui`(若同机存在)在 `/usr/local/s-ui/`,本仓库的卸载命令**完全不会触碰**它。

---

## 协议支持

- 通用:Mixed / SOCKS / HTTP / HTTPS / Direct / Redirect / TProxy
- V2Ray 系:VLESS / VMess / Trojan / Shadowsocks
- 其它:ShadowTLS / Hysteria / Hysteria2 / Naive / TUIC / AnyTLS / WireGuard / Tailscale / Warp / Tor / SSH
- 完整 XTLS 支持
- Reality / ECH / **ACME(支持 Cloudflare DNS-01 自动签发)**

---

## 环境变量

| 变量             | 类型                                              | 默认       |
| ---------------- | :-----------------------------------------------: | :--------- |
| `SUI_LOG_LEVEL`  | `"debug"` \| `"info"` \| `"warn"` \| `"error"`    | `"info"`   |
| `SUI_DEBUG`      | `boolean`                                         | `false`    |
| `SUI_BIN_FOLDER` | `string`                                          | `"bin"`    |
| `SUI_DB_FOLDER`  | `string`                                          | `"db"`     |
| `SINGBOX_API`    | `string`                                          | -          |

---

## 文档

- [`UI_MIGRATION_PLAN.md`](./UI_MIGRATION_PLAN.md) —— 前端迁移计划、阶段记录、风险登记、性能对比
- [`CONTRIBUTING.md`](./CONTRIBUTING.md) —— 开发规范、分支管理、测试与 PR 流程
- [`CLAUDE.md`](./CLAUDE.md) —— Claude 工作交接(本仓库专用执行规则)

---

## 相关项目

- [`nexcore-x-ui`](https://github.com/DoBestone/nexcore-x-ui) —— NexCore 产品线下的 xray 控制面板;`/api/v1` 接口与本仓库互通,可用同一份主控代码同时管两套节点。

---

## 致谢

- [SagerNet/sing-box](https://github.com/SagerNet/sing-box) —— 协议栈
- [Element Plus](https://element-plus.org/) —— 组件库
- [Vue 3](https://vuejs.org/) —— 前端框架
- [Cloudflare API](https://developers.cloudflare.com/api/) —— DNS / ACME 自动化
- [alireza0/s-ui](https://github.com/alireza0/s-ui) —— 早期版本基于此 fork;前端已全量重写,后端做了较多改造

---

## 许可

GPL v3。
