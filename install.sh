#!/bin/bash
# nexcore-s-ui · install (fresh install only — for upgrades use update.sh
# or `s-ui update`).
#
# 用法:
#   bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/install.sh)
#   bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/install.sh) v1.4.1
#   bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/nexcore-s-ui/main/install.sh) --force
#
# 想从上游 alireza0/s-ui release 拉(本仓库未自打 release 时):
#   GH_OWNER=alireza0 GH_REPO=s-ui bash <(curl -Ls .../install.sh)
#
# 可覆盖默认值的环境变量:
#   GH_OWNER  GH_REPO  INSTALL_DIR  PKG_PREFIX  CMD_NAME  SERVICE_NAME
#
# 与 nexcore-x-ui 思路一致:首装走全自动随机凭据,装完立即 echo 出
# 用户名/密码/面板 URI;这是 v1.0 之前老脚本一堆 read -p 交互的取代品。

set -eo pipefail

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
blue='\033[0;34m'
cyan='\033[0;36m'
plain='\033[0m'

CMD_NAME="${CMD_NAME:-s-ui}"
SERVICE_NAME="${SERVICE_NAME:-${CMD_NAME}}"
GH_OWNER="${GH_OWNER:-DoBestone}"
GH_REPO="${GH_REPO:-nexcore-s-ui}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/${CMD_NAME}}"
PKG_PREFIX="${PKG_PREFIX:-s-ui}"   # tarball 解压顶层目录名,与 release 包结构一致
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
DB_PATH="${INSTALL_DIR}/db/${CMD_NAME}.db"

FORCE=false
TARGET_VERSION=""

for arg in "$@"; do
    case "$arg" in
        --force|-f)  FORCE=true ;;
        v*|V*|[0-9]*) TARGET_VERSION="$arg" ;;
    esac
done

# ---------- output helpers ----------

step() { echo -e "${blue}▸${plain} $*"; }
ok()   { echo -e "${green}✓${plain} $*"; }
warn() { echo -e "${yellow}!${plain} $*" >&2; }
err()  { echo -e "${red}✗${plain} $*" >&2; }
die()  { err "$*"; exit 1; }

# ---------- preflight ----------

[[ $EUID -ne 0 ]] && die "必须以 root 身份运行此脚本"

if ! command -v systemctl >/dev/null 2>&1; then
    die "本机没有 systemd,无法安装。仅支持带 systemd 的发行版"
fi

case $(uname -m) in
    x86_64|x64|amd64)            ARCH=amd64 ;;
    i*86|x86)                    ARCH=386 ;;
    aarch64|arm64|armv8*|armv8)  ARCH=arm64 ;;
    armv7l|armv7*|armv7|arm)     ARCH=armv7 ;;
    armv6*|armv6)                ARCH=armv6 ;;
    armv5*|armv5)                ARCH=armv5 ;;
    s390x)                       ARCH=s390x ;;
    *) die "未支持的 CPU 架构:$(uname -m)" ;;
esac

# detect existing install — 装过了就让用 update.sh,除非 --force
existing_install=false
if [[ -f "${SERVICE_FILE}" ]] || [[ -x "${INSTALL_DIR}/sui" ]]; then
    existing_install=true
fi
if ${existing_install} && ! ${FORCE}; then
    err "检测到 ${CMD_NAME} 已安装(存在 ${SERVICE_FILE} 或 ${INSTALL_DIR}/sui)。"
    echo
    echo -e "  ${cyan}日常升级请用:${plain}"
    echo -e "    bash <(curl -Ls https://raw.githubusercontent.com/${GH_OWNER}/${GH_REPO}/main/update.sh)"
    echo -e "    或:  ${CMD_NAME} update"
    echo
    echo -e "  ${cyan}强制重装(覆盖二进制,保留 ${INSTALL_DIR}/db/):${plain}"
    echo -e "    bash <(curl -Ls https://raw.githubusercontent.com/${GH_OWNER}/${GH_REPO}/main/install.sh) --force"
    exit 1
fi

# DATA_DIR (db/) 父目录可写
if [[ ! -w "$(dirname "${INSTALL_DIR}")" ]]; then
    die "无法写入 $(dirname "${INSTALL_DIR}") — 检查 root 权限"
fi

free_tmp_kb=$(df -k /tmp 2>/dev/null | awk 'NR==2{print $4}')
if [[ -n "${free_tmp_kb}" && "${free_tmp_kb}" -lt 102400 ]]; then
    warn "/tmp 可用空间 < 100MB($((free_tmp_kb / 1024))MB),下载/解压可能失败"
fi

mem_kb=$(awk '/MemTotal/{print $2}' /proc/meminfo 2>/dev/null || echo 0)
if [[ "${mem_kb}" -gt 0 && "${mem_kb}" -lt 524288 ]]; then
    warn "系统内存 < 512MB($((mem_kb / 1024))MB),sing-box + 面板 + sqlite 同时运行可能 OOM"
fi

echo -e "${green}架构:${plain} ${ARCH}"

# ---------- deps ----------

ensure_pkg() {
    local pkgs="$*"
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -y >/dev/null 2>&1 || true
        DEBIAN_FRONTEND=noninteractive apt-get install -y ${pkgs} >/dev/null 2>&1 || return 1
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y ${pkgs} >/dev/null 2>&1 || return 1
    elif command -v yum >/dev/null 2>&1; then
        yum install -y ${pkgs} >/dev/null 2>&1 || return 1
    elif command -v zypper >/dev/null 2>&1; then
        zypper -q install -y ${pkgs} >/dev/null 2>&1 || return 1
    elif command -v pacman >/dev/null 2>&1; then
        pacman -Sy --noconfirm ${pkgs} >/dev/null 2>&1 || return 1
    elif command -v apk >/dev/null 2>&1; then
        apk add --no-cache ${pkgs} >/dev/null 2>&1 || return 1
    else
        return 1
    fi
}

required_tools=(curl tar awk grep sed)
missing=()
for t in "${required_tools[@]}"; do
    command -v "$t" >/dev/null 2>&1 || missing+=("$t")
done
# coreutils 提供 sha256sum
if ! command -v sha256sum >/dev/null 2>&1; then
    missing+=(coreutils)
fi
# tzdata 不强制但有用
if [[ ${#missing[@]} -gt 0 ]]; then
    step "安装缺失依赖: ${missing[*]}"
    ensure_pkg "${missing[@]}" || warn "自动安装失败 — 可手动 \`apt install ${missing[*]}\` 或等价命令"
fi
for t in curl tar awk grep sed; do
    command -v "$t" >/dev/null 2>&1 || die "依赖 '$t' 不可用,无法继续"
done

# ---------- download + verify ----------

resolve_version() {
    if [[ -n "$1" ]]; then
        echo "$1"
        return
    fi
    local v
    v=$(curl -fsSL "https://api.github.com/repos/${GH_OWNER}/${GH_REPO}/releases/latest" \
        | grep -E '"tag_name":' \
        | sed -E 's/.*"([^"]+)".*/\1/' || true)
    if [[ -z "$v" ]]; then
        die "无法获取最新版本号(GitHub API 限流?或仓库尚无 release — 试 GH_OWNER=alireza0 GH_REPO=s-ui)"
    fi
    echo "$v"
}

download_release() {
    # 调用方用 $(...) 捕获 tarball 路径,所有进度输出必须走 stderr
    local version="$1"
    local pkg="${PKG_PREFIX}-linux-${ARCH}.tar.gz"
    local url="https://github.com/${GH_OWNER}/${GH_REPO}/releases/download/${version}/${pkg}"
    local sum_url="https://github.com/${GH_OWNER}/${GH_REPO}/releases/download/${version}/checksums.txt"
    local dest="/tmp/${PKG_PREFIX}-${version}-${ARCH}.tar.gz"
    local sum_dest="/tmp/${PKG_PREFIX}-${version}-checksums.txt"

    step "下载: ${url}" >&2
    if ! curl -fSL --connect-timeout 10 -o "${dest}" "${url}" >&2; then
        rm -f "${dest}"
        die "下载失败 — release ${version} 可能不存在,或 GitHub 网络不通"
    fi

    # SHA256 best-effort:有 checksums.txt 就强校验,没有就 warn 后继续。
    # 上游 alireza0/s-ui 的 release 没有 checksums.txt;本仓库的 release
    # 应该带上。
    if curl -fsSL --connect-timeout 10 -o "${sum_dest}" "${sum_url}" 2>/dev/null && command -v sha256sum >/dev/null 2>&1; then
        step "校验 SHA256…" >&2
        local expected actual
        expected=$(awk -v want="${pkg}" '$2 == want || $2 == "*"want {print $1; exit}' "${sum_dest}")
        if [[ -z "${expected}" ]]; then
            warn "checksums.txt 中没有 ${pkg} 的条目,跳过校验"
        else
            actual=$(sha256sum "${dest}" | awk '{print $1}')
            if [[ "${expected}" != "${actual}" ]]; then
                rm -f "${dest}" "${sum_dest}"
                die "SHA256 不匹配!  expected=${expected}  actual=${actual}"
            fi
            ok "  SHA256 OK: ${actual}" >&2
        fi
        rm -f "${sum_dest}"
    else
        warn "release 未提供 checksums.txt,跳过 SHA256 校验" >&2
    fi

    echo "${dest}"
}

# ---------- install ----------

handle_legacy_singbox() {
    # 老安装可能还残留 sing-box.service(上游 v1.x 旧拓扑),停掉并清掉
    # 二进制残骸 — 与原 install.sh 的 prepare_services 同语义
    if [[ -f /etc/systemd/system/sing-box.service ]]; then
        step "清理老 sing-box.service…"
        systemctl stop sing-box 2>/dev/null || true
        systemctl disable sing-box 2>/dev/null || true
        rm -f /etc/systemd/system/sing-box.service
        rm -f "${INSTALL_DIR}/bin/sing-box" "${INSTALL_DIR}/bin/runSingbox.sh" "${INSTALL_DIR}/bin/signal" 2>/dev/null || true
    fi
}

install_panel() {
    local version="$1"
    local archive="$2"

    if ${existing_install}; then
        step "停止旧服务"
        systemctl stop "${SERVICE_NAME}" 2>/dev/null || true
    fi

    handle_legacy_singbox

    # 解压到临时目录后再 cp,这样 db/ 不会被 release 包覆盖
    local extract_dir="/tmp/${CMD_NAME}-extract"
    rm -rf "${extract_dir}"
    mkdir -p "${extract_dir}"
    tar -xzf "${archive}" -C "${extract_dir}/"
    if [[ ! -d "${extract_dir}/${PKG_PREFIX}" ]]; then
        rm -rf "${extract_dir}"
        die "压缩包结构异常(缺少 ${PKG_PREFIX}/ 目录)"
    fi

    mkdir -p "${INSTALL_DIR}"

    # 替换二进制 + 脚本 + service unit;永远不动 ${INSTALL_DIR}/db/
    install -m 0755 "${extract_dir}/${PKG_PREFIX}/sui" "${INSTALL_DIR}/sui"
    if [[ -f "${extract_dir}/${PKG_PREFIX}/${CMD_NAME}.sh" ]]; then
        install -m 0755 "${extract_dir}/${PKG_PREFIX}/${CMD_NAME}.sh" "${INSTALL_DIR}/${CMD_NAME}.sh"
        install -m 0755 "${INSTALL_DIR}/${CMD_NAME}.sh" "/usr/bin/${CMD_NAME}"
    fi
    if [[ -d "${extract_dir}/${PKG_PREFIX}/bin" ]]; then
        rm -rf "${INSTALL_DIR}/bin"
        cp -a "${extract_dir}/${PKG_PREFIX}/bin" "${INSTALL_DIR}/bin"
        chmod +x "${INSTALL_DIR}/bin/"* 2>/dev/null || true
    fi
    # GitHub Actions runner 默认 uid 1001;若上游打包未 chown,这里强制纠回 root
    chown -R root:root "${INSTALL_DIR}" 2>/dev/null || true
    mkdir -p "${INSTALL_DIR}/db"
    chmod 700 "${INSTALL_DIR}/db"

    # service file:tarball 里有就装上(常见命名 *.service)
    local svc_src
    svc_src=$(find "${extract_dir}/${PKG_PREFIX}" -maxdepth 1 -name "*.service" | head -n1)
    if [[ -n "${svc_src}" ]]; then
        install -m 0644 "${svc_src}" "${SERVICE_FILE}"
    elif [[ ! -f "${SERVICE_FILE}" ]]; then
        # tarball 没有 service 文件且本机也没有 — 现造一个最小可用版
        cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=${CMD_NAME} Service
After=network.target
Wants=network.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}/
ExecStart=${INSTALL_DIR}/sui
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
        chmod 644 "${SERVICE_FILE}"
    fi

    rm -rf "${extract_dir}" "${archive}"

    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}" >/dev/null 2>&1 || true
    systemctl reset-failed "${SERVICE_NAME}" 2>/dev/null || true
}

# ---------- post-install: migrate, seed admin, start ----------

random_credential() {
    # base64 6 字节 → 8 字符可见,去掉特殊符避免后续 shell 引用问题
    head -c 12 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 12
}

setup_first_run() {
    # migrate 是幂等的,跨"首装/升级"都跑一次
    step "数据库迁移…"
    "${INSTALL_DIR}/sui" migrate || warn "migrate 报错(可能是首装空 db,可忽略)"

    if [[ -f "${DB_PATH}" ]] && "${INSTALL_DIR}/sui" admin -show 2>/dev/null | grep -q '^[[:space:]]*Username:[[:space:]]*[^[:space:]]'; then
        # 已有 admin → 升级语义,不动凭据
        FRESH_USER=""
        FRESH_PASS=""
        return
    fi

    FRESH_USER="admin_$(random_credential | tr 'A-Z' 'a-z' | head -c 6)"
    FRESH_PASS="$(random_credential)"
    step "创建初始管理员凭据(随机生成)"
    if ! "${INSTALL_DIR}/sui" admin -username "${FRESH_USER}" -password "${FRESH_PASS}" >/dev/null 2>&1; then
        warn "写入 admin 凭据失败 — 装完手动 ${cyan}${CMD_NAME}${plain} 进菜单重置"
        FRESH_USER=""
        FRESH_PASS=""
    fi
}

wait_for_active() {
    local max="${1:-30}"
    for i in $(seq 1 "${max}"); do
        if systemctl is-active --quiet "${SERVICE_NAME}"; then
            return 0
        fi
        sleep 1
    done
    return 1
}

show_credentials() {
    echo
    echo -e "${green}═════════════════════════════════════════════${plain}"
    echo -e "${green}  nexcore-s-ui 已部署${plain}"
    echo -e "${green}═════════════════════════════════════════════${plain}"

    "${INSTALL_DIR}/sui" setting -show 2>/dev/null || \
        warn "setting -show 失败 — 试试: ${cyan}journalctl -u ${SERVICE_NAME} -n 80${plain}"

    echo
    echo -e "${green}访问地址:${plain}"
    "${INSTALL_DIR}/sui" uri 2>/dev/null || warn "无法获取 URI — 检查服务状态"

    if [[ -n "${FRESH_USER}" && -n "${FRESH_PASS}" ]]; then
        echo
        echo -e "${yellow}首装明文凭据 (★ 立即记录,后续只能 ${CMD_NAME} 菜单重置):${plain}"
        echo -e "  用户名: ${green}${FRESH_USER}${plain}"
        echo -e "  密 码:  ${green}${FRESH_PASS}${plain}"
    else
        echo
        echo -e "${yellow}保留已有管理员凭据${plain}(升级/重装语义,未生成新密码)"
        echo -e "  查看当前账号: ${cyan}${INSTALL_DIR}/sui admin -show${plain}"
    fi

    echo
    echo -e "${green}管理命令:${plain}"
    echo "  ${CMD_NAME}                       交互菜单"
    echo "  systemctl status ${SERVICE_NAME}  服务状态"
    echo "  journalctl -u ${SERVICE_NAME} -f  实时日志"
    echo
}

# ---------- main ----------

echo -e "${green}nexcore-s-ui · install${plain}"
VERSION="$(resolve_version "${TARGET_VERSION}")"
echo -e "${green}版本:${plain} ${VERSION}"

ARCHIVE="$(download_release "${VERSION}")"
install_panel "${VERSION}" "${ARCHIVE}"

# admin / migrate 必须在 systemctl start 之前 — sui CLI 与 panel 进程
# 共用同一个 sqlite,start 之后再调 admin 会卡 db lock
setup_first_run

step "启动服务"
systemctl restart "${SERVICE_NAME}"

step "等待 systemd 服务激活(最长 30s)…"
if ! wait_for_active 30; then
    err "服务未在 30s 内进入 active 状态。最近日志:"
    journalctl -u "${SERVICE_NAME}" -n 60 --no-pager || true
    die "首次启动失败 — 修复后再 \`systemctl restart ${SERVICE_NAME}\` 或重跑本脚本"
fi
ok "服务已激活"

show_credentials
