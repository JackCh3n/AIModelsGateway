#!/usr/bin/env bash
# ============================================================
# AI Models Gateway 多平台交叉编译脚本
# 用法: ./scripts/build-all.sh <版本号>
#   版本号如 v1.0.0 或 v2025_0101_1200
# 产物输出到 dist/ 目录
# ============================================================
set -euo pipefail

VERSION="${1:?用法: build-all.sh <版本号>}"
export CGO_ENABLED=0
export TZ=Asia/Shanghai

echo "==> 构建版本: $VERSION"
mkdir -p dist

build() {
  local os="$1" arch="$2"
  local ext="${3:-}"
  local name="aimodels-${os}-${arch}${ext}"
  echo "==> 编译 $os/$arch -> dist/$name"
  GOOS="$os" GOARCH="$arch" go build -ldflags "-s -w -X main.Version=${VERSION}" -o "dist/${name}" .
}

# Windows
build windows amd64 .exe
build windows arm64 .exe
# Linux x86/ARM
build linux amd64
build linux arm64
# Linux 国产芯片
build linux loong64
build linux ppc64le
build linux riscv64
build linux s390x
# macOS
build darwin amd64
build darwin arm64
# FreeBSD
build freebsd amd64
build freebsd arm64

echo ""
echo "==> 构建完成，产物列表："
ls -lh dist/
