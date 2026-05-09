#!/bin/sh
set -e

cd frontend
npm i
npm run build

cd ..
echo "Backend"

mkdir -p web/html
rm -fr web/html/*
cp -R frontend/dist/* web/html/

# 平台分发:
#   - macOS:用 build.sh 原作者的 with_musl + naive_outbound,走 macOS 链接器
#   - Linux:Ubuntu/Debian glibc 用 release.yml 的精简 tag 集
#     去掉 with_musl(glibc 不需要静态链接)
#     去掉 with_naive_outbound(链接预编译的 libcronet.a,旧版 ld 不认 .crel.text)
#     去掉 -Wl,-no_warn_duplicate_libraries(macOS-only)
#   gvisor / tailscale / acme 必带 — 否则 warp / tailscale 出站 / panel SSL 失效
case "$(uname -s)" in
  Darwin)
    BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale"
    LDFLAGS='-w -s -checklinkname=0 -extldflags "-Wl,-no_warn_duplicate_libraries"'
    ;;
  *)
    BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,badlinkname,tfogo_checklinkname0,with_tailscale"
    LDFLAGS='-w -s -checklinkname=0'
    ;;
esac

go build -ldflags "$LDFLAGS" -tags "$BUILD_TAGS" -o sui main.go
