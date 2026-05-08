# nexcore-s-ui

**Sing-Box 高级 Web 控制面板 · NexCore 二次开发版**

[![Go Report Card](https://goreportcard.com/badge/github.com/DoBestone/nexcore-s-ui)](https://goreportcard.com/report/github.com/DoBestone/nexcore-s-ui)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

> 基于 [alireza0/s-ui](https://github.com/alireza0/s-ui) v1.4.1 二次开发。前端从 Vuetify 4 全量重写为 Element Plus,严格遵循 [NexCore 统一 UI 设计规范](https://github.com/DoBestone/nexcore-s-ui/blob/main/UI_MIGRATION_PLAN.md);后端基于上游,精简了 Docker 链路与冗余日志容器检测。

> **声明:** 仅供个人学习与技术研究,请勿用于非法用途。

---

## 与上游 alireza0/s-ui 的差异

| 维度 | 上游 | 本仓库 |
|---|---|---|
| 前端框架 | Vuetify 4 + Material Design | **Element Plus + NexCore 设计系统** |
| 字体 | Roboto / 默认 system | Manrope display + system body + JetBrains Mono(`tabular-nums`) |
| 国际化 | 6 种全量加载 | 6 种(en / fa / vi / zhHans / zhHant / ru),**异步按需加载** |
| 通知 | Notivue | ElMessage / ElNotification |
| 时间库 | moment.js | dayjs |
| Docker | 提供 Dockerfile / compose / CI | **已移除** |
| 前端集成 | git submodule | **直接合并到本仓库** |
| 初始 JS / CSS | ~1280 KB / ~370 KB | **213 KB / 28 KB**(gzip ~86 KB) |
| Modal 加载 | 全量打包 | **按需 lazy import** |

---

## 功能速查

| 功能                                    |       状态       |
| --------------------------------------- | :--------------: |
| 多协议(VLESS / VMess / Trojan / Hysteria 等)|        ✅        |
| 多语言(en / fa / vi / 中 / 繁 / ру)    |        ✅        |
| 多客户端 / 多入站                        |        ✅        |
| 高级路由规则界面                         |        ✅        |
| 在线状态、流量、系统监控                  |        ✅        |
| 订阅链接(原生 / json / clash + 元信息) |        ✅        |
| 浅色 / 深色主题                          |        ✅        |
| API 接口                                 |        ✅        |

## 支持平台

| 平台 | 架构 | 状态 |
|----------|--------------|---------|
| Linux    | amd64, arm64, armv7, armv6, armv5, 386, s390x | ✅ |
| Windows  | amd64, 386, arm64 | ✅ |
| macOS    | amd64, arm64 | 🚧 实验性 |

## 默认安装信息

- 面板端口:2095
- 面板路径:`/app/`
- 订阅端口:2096
- 订阅路径:`/sub/`
- 默认账号 / 密码:`admin` / `admin`(**生产前请立即修改**)

---

## 一键脚本(Linux / macOS)

> 上游 `alireza0/s-ui` 的官方脚本仍然可用(它从 `alireza0/s-ui` releases 拉取二进制)。如果你希望使用本仓库的产物,需要先 fork-ready 本仓库并自行打 release。

上游脚本(快速可用):
```sh
bash <(curl -Ls https://raw.githubusercontent.com/alireza0/s-ui/master/install.sh)
```

---

## 从源码构建(本仓库)

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
3. `go build -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale" -o sui main.go`

最终产物:`./sui`(macOS arm64 约 53 MB,Linux 类似)。

### 运行

```sh
./sui
```

默认监听 `2095`,浏览器打开 `http://127.0.0.1:2095/app/`,登录 `admin` / `admin`。

---

## 仅开发前端

```sh
cd frontend
npm i
npm run dev      # vite dev server, 默认 :3000,代理 /app/api → :2095
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
systemctl disable s-ui --now
rm -f /etc/systemd/system/sing-box.service
systemctl daemon-reload
rm -fr /usr/local/s-ui
rm /usr/bin/s-ui
```

---

## 协议支持

- 通用:Mixed / SOCKS / HTTP / HTTPS / Direct / Redirect / TProxy
- V2Ray 系:VLESS / VMess / Trojan / Shadowsocks
- 其它:ShadowTLS / Hysteria / Hysteria2 / Naive / TUIC / AnyTLS / WireGuard / Tailscale / Warp / Tor / SSH
- 完整 XTLS 支持
- Reality / ECH / ACME 集中证书管理

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

## SSL 证书(Certbot 简版)

```bash
snap install core; snap refresh core
snap install --classic certbot
ln -s /snap/bin/certbot /usr/bin/certbot
certbot certonly --standalone --register-unsafely-without-email \
                 --non-interactive --agree-tos -d <你的域名>
```

---

## 文档

- [`UI_MIGRATION_PLAN.md`](./UI_MIGRATION_PLAN.md) —— 迁移计划、阶段记录、风险登记、三轮审计修复明细 + 性能对比
- [`CONTRIBUTING.md`](./CONTRIBUTING.md) —— 开发规范、分支管理、测试与 PR 流程
- [`CLAUDE.md`](./CLAUDE.md) —— Claude 工作交接(本仓库专用执行规则)

---

## 致谢

- [alireza0/s-ui](https://github.com/alireza0/s-ui) —— 上游原作者,Sing-Box 控制面板的核心实现
- [SagerNet/sing-box](https://github.com/SagerNet/sing-box) —— 协议栈
- [Element Plus](https://element-plus.org/) —— 组件库
- [Vue 3](https://vuejs.org/) —— 前端框架

---

## 许可

GPL v3 —— 与上游一致。
