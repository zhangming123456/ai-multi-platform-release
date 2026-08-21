#!/bin/bash

# 多平台发布项目启动脚本
# 后端: Go/Beego (默认 8000)
# 前端: Vue/Vite (默认 5500)

cd "$(dirname "$0")"

MODE=${1:-start}
if [[ "$MODE" != "start" && "$MODE" != "restart" ]]; then
    printf '[ERROR] 用法: ./start.sh [start|restart]\n'
    exit 1
fi

# 默认端口
DEFAULT_BACKEND_PORT=8000
DEFAULT_FRONTEND_PORT=5500

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()  { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 杀掉占用端口的进程
kill_port() {
    local port=$1
    local pid=$(lsof -ti :$port 2>/dev/null || true)
    if [ -n "$pid" ]; then
        warn "端口 $port 被占用 (PID: $pid)"
        if [[ "$MODE" == "restart" ]]; then
            confirm=y
        else
            read -p "是否杀掉该进程? (y/N): " confirm
        fi
        if [[ "$confirm" =~ ^[Yy]$ ]]; then
            kill -9 $pid 2>/dev/null || true
            sleep 0.5
            success "已杀掉端口 $port 上的进程"
        else
            error "取消启动"
            exit 1
        fi
    fi
}

echo "========================================="
echo "    多平台发布项目启动脚本"
echo "========================================="
echo ""

# 交互式输入端口；重启时可复用原端口，避免再次询问。
if [[ -n "${2:-}" ]]; then
    BACKEND_PORT=$2
else
    read -p "请输入后端端口 (默认: $DEFAULT_BACKEND_PORT): " input
    BACKEND_PORT=${input:-$DEFAULT_BACKEND_PORT}
fi

if [[ -n "${3:-}" ]]; then
    FRONTEND_PORT=$3
else
    read -p "请输入前端端口 (默认: $DEFAULT_FRONTEND_PORT): " input
    FRONTEND_PORT=${input:-$DEFAULT_FRONTEND_PORT}
fi

echo ""
info "后端端口: $BACKEND_PORT"
info "前端端口: $FRONTEND_PORT"
echo ""

# 检查并处理端口占用
info "检查端口占用..."
kill_port $BACKEND_PORT
kill_port $FRONTEND_PORT
echo ""

# 检查依赖
info "检查依赖..."
if [ ! -d "node_modules" ] || [ ! -d "frontend/node_modules" ]; then
    warn "依赖未安装，正在安装..."
    npm install
    cd frontend && npm install && cd ..
    success "依赖安装完成"
fi

# 预编译后端二进制（避免 go run 每次重新编译）
info "编译后端..."
cd backend-go
go build -o ./mpr-backend .
cd ..
success "后端编译完成"

echo ""
success "正在启动项目..."
echo ""

# 使用本地已安装的 concurrently，避免 npx 下载
node_modules/.bin/concurrently -n backend,frontend -c blue,green \
    "cd backend-go && ./mpr-backend -port $BACKEND_PORT" \
    "cd frontend && npm run dev -- --port $FRONTEND_PORT" &
APP_PID=$!

if [[ ! -t 0 ]]; then
    wait "$APP_PID"
    exit $?
fi

echo ""
info "运行中：按 R 重启，按 Q 退出"
while kill -0 "$APP_PID" 2>/dev/null; do
    if read -r -s -n 1 -t 1 key; then
        case "$key" in
            [Rr])
                echo ""
                info "正在重启项目..."
                kill "$APP_PID" 2>/dev/null || true
                wait "$APP_PID" 2>/dev/null || true
                exec "$0" restart "$BACKEND_PORT" "$FRONTEND_PORT"
                ;;
            [Qq])
                echo ""
                info "正在停止项目..."
                kill "$APP_PID" 2>/dev/null || true
                wait "$APP_PID" 2>/dev/null || true
                exit 0
                ;;
        esac
    fi
done
wait "$APP_PID"
