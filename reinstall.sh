#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
echo "========================================="
echo "  重新安装 & 启动项目"
echo "  目录: $PROJECT_DIR"
echo "========================================="

echo ""
echo "[1/4] 停止已有进程..."
lsof -ti:8000 -ti:5500 2>/dev/null | xargs kill -9 2>/dev/null || true
echo "     ✓ 已停止"

echo ""
echo "[2/4] 清除 node_modules..."
rm -rf "$PROJECT_DIR/node_modules" "$PROJECT_DIR/frontend/node_modules" 2>/dev/null || true
echo "     ✓ 已清除"

echo ""
echo "[3/4] 安装依赖..."
cd "$PROJECT_DIR/frontend" && npm install && cd "$PROJECT_DIR" && npm install
echo "     ✓ 安装完成"

echo ""
echo "[4/4] 启动项目..."
echo ""
echo "  ┌─────────────────────────────────────┐"
echo "  │  前端  http://localhost:5500/       │"
echo "  │  后端  http://localhost:8000/       │"
echo "  └─────────────────────────────────────┘"
echo ""
echo "========================================="
npm run dev
