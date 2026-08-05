#!/usr/bin/env bash
# ============================================================
# 生成 Release 发布说明
# 用法: ./scripts/gen-release-notes.sh <版本号> [说明文字]
# 输出到 RELEASE_NOTES.md
# ============================================================
set -euo pipefail

VERSION="${1:?用法: gen-release-notes.sh <版本号> [说明文字]}"
NOTE="${2:-由 cnb.cool 云原生构建自动生成}"

{
  echo "## AI Models Gateway $VERSION"
  echo ""
  echo "$NOTE"
  echo ""
  echo "### 构建产物"
  echo ""
  echo "| 平台 | 架构 | 文件名 |"
  echo "|------|------|--------|"
  echo "| Windows | amd64 | \`aimodels-windows-amd64.exe\` |"
  echo "| Windows | arm64 | \`aimodels-windows-arm64.exe\` |"
  echo "| Linux | amd64 | \`aimodels-linux-amd64\` |"
  echo "| Linux | arm64 | \`aimodels-linux-arm64\` |"
  echo "| Linux | loong64 | \`aimodels-linux-loong64\` |"
  echo "| Linux | ppc64le | \`aimodels-linux-ppc64le\` |"
  echo "| Linux | riscv64 | \`aimodels-linux-riscv64\` |"
  echo "| Linux | s390x | \`aimodels-linux-s390x\` |"
  echo "| macOS | amd64 | \`aimodels-darwin-amd64\` |"
  echo "| macOS | arm64 | \`aimodels-darwin-arm64\` |"
  echo "| FreeBSD | amd64 | \`aimodels-freebsd-amd64\` |"
  echo "| FreeBSD | arm64 | \`aimodels-freebsd-arm64\` |"
  echo ""
  echo "### 运行说明"
  echo ""
  echo "1. 下载对应平台的二进制文件"
  echo "2. Linux/macOS 执行前先赋予执行权限: \`chmod +x aimodels-*\`"
  echo "3. 启动: \`./aimodels-<平台>-<架构> -port 3458\`"
  echo "4. 打开管理后台: \`http://127.0.0.1:3458/admin/\`"
  echo ""
  echo "> 版本号: $VERSION | 构建来源: cnb.cool 云原生构建"
} > RELEASE_NOTES.md

cat RELEASE_NOTES.md
