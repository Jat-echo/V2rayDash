#!/bin/bash
#==========================================
# V2rayDash 自动化部署脚本
# 用于部署前端和后端到服务器
#==========================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认配置
SERVER_IP=""
SERVER_PORT=22
SERVER_USER="root"
SERVER_PASSWORD=""
BACKEND_DIR="/opt/v2ray-dash"
FRONTEND_DIST_DIR="/home/jat-id/Project/V2rayDash/frontend/dist"
BACKEND_BINARY_NAME="v2ray-dash-backend"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --ip)
            SERVER_IP="$2"
            shift 2
            ;;
        --port)
            SERVER_PORT="$2"
            shift 2
            ;;
        --user)
            SERVER_USER="$2"
            shift 2
            ;;
        --password)
            SERVER_PASSWORD="$2"
            shift 2
            ;;
        --help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  --ip <IP>           服务器IP地址 (必需)"
            echo "  --port <端口>       SSH端口 (默认: 22)"
            echo "  --user <用户>       SSH用户 (默认: root)"
            echo "  --password <密码>   SSH密码 (必需)"
            exit 0
            ;;
        *)
            echo "未知选项: $1"
            exit 1
            ;;
    esac
done

# 检查必需参数
if [[ -z "$SERVER_IP" ]] || [[ -z "$SERVER_PASSWORD" ]]; then
    echo -e "${RED}错误: 请提供服务器IP和密码${NC}"
    echo "使用 --help 查看帮助"
    exit 1
fi

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo -e "${GREEN}====== V2rayDash 部署脚本 ======${NC}"
echo -e "${BLUE}服务器: ${SERVER_USER}@${SERVER_IP}:${SERVER_PORT}${NC}"
echo ""

# 预检查：测试SSH连接
echo -e "${YELLOW}[预检查] 测试SSH连接...${NC}"
sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" "echo 'SSH OK'" > /dev/null 2>&1
if [[ $? -ne 0 ]]; then
    echo -e "${RED}错误: SSH连接失败，请检查IP、端口和密码${NC}"
    exit 1
fi
echo -e "${GREEN}SSH连接成功${NC}"

# 获取nginx配置的root路径
echo -e "${YELLOW}[1/5] 获取服务器nginx配置...${NC}"
NGINX_ROOT=$(sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" "grep -m1 'root' /etc/nginx/sites-enabled/*.conf 2>/dev/null | awk '{print \$2}' | tr -d ';'")
if [[ -z "$NGINX_ROOT" ]]; then
    echo -e "${YELLOW}警告: 无法从nginx配置获取root路径，使用默认: ${BACKEND_DIR}/frontend/dist${NC}"
    FRONTEND_TARGET_DIR="${BACKEND_DIR}/frontend/dist"
else
    FRONTEND_TARGET_DIR="$NGINX_ROOT"
fi
echo -e "前端部署路径: ${FRONTEND_TARGET_DIR}"

# 获取服务器后端当前运行状态
echo -e "${YELLOW}[2/5] 检查服务器状态...${NC}"
REMOTE_BACKEND_PATH=$(sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" "systemctl show v2ray-dash --property=ExecStart --value 2>/dev/null | grep -o '/[^ ]*v2ray-dash-server' | head -1")
if [[ -z "$REMOTE_BACKEND_PATH" ]]; then
    REMOTE_BACKEND_PATH="${BACKEND_DIR}/v2ray-dash-server"
fi
echo -e "后端运行路径: ${REMOTE_BACKEND_PATH}"

# 构建前端
echo -e "${YELLOW}[3/5] 构建前端...${NC}"
cd "$PROJECT_DIR/frontend"
if [[ ! -d "dist" ]]; then
    echo -e "${YELLOW}执行 npm install && npm run build...${NC}"
    npm install > /dev/null 2>&1
fi
npm run build > /dev/null 2>&1
echo -e "${GREEN}前端构建完成${NC}"

# 构建后端
echo -e "${YELLOW}[4/5] 构建后端...${NC}"
cd "$PROJECT_DIR/backend"
if [[ ! -f "/usr/local/go/bin/go" ]]; then
    echo -e "${RED}错误: 服务器上未安装Go，无法编译后端${NC}"
    echo "请先在服务器上安装Go或使用已编译的后端"
    exit 1
fi
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn,direct
go build -o "$BACKEND_BINARY_NAME" ./cmd/server/main.go
echo -e "${GREEN}后端构建完成${NC}"

# 上传前端
echo -e "${YELLOW}[5/5] 上传文件到服务器...${NC}"

# 删除远程旧文件并创建目录
sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" "rm -rf ${FRONTEND_TARGET_DIR} && mkdir -p ${FRONTEND_TARGET_DIR}" 2>/dev/null

# 上传前端文件
echo -e "${BLUE}  上传前端...${NC}"
sshpass -p "$SERVER_PASSWORD" scp -o StrictHostKeyChecking=no -P "$SERVER_PORT" -r "$PROJECT_DIR/frontend/dist/"* "${SERVER_USER}@${SERVER_IP}:${FRONTEND_TARGET_DIR}/"
echo -e "${GREEN}  前端上传完成${NC}"

# 上传后端
echo -e "${BLUE}  上传后端...${NC}"
sshpass -p "$SERVER_PASSWORD" scp -o StrictHostKeyChecking=no -P "$SERVER_PORT" "$PROJECT_DIR/backend/$BACKEND_BINARY_NAME" "${SERVER_USER}@${SERVER_IP}:/tmp/$BACKEND_BINARY_NAME"

# 安装后端并重启服务
sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" << 'ENDSSH'
    # 停止旧服务
    pkill -9 v2ray-dash-server 2>/dev/null || true

    # 备份旧版本
    if [[ -f /opt/v2ray-dash/v2ray-dash-server ]]; then
        mv /opt/v2ray-dash/v2ray-dash-server /opt/v2ray-dash/v2ray-dash-server.bak.$(date +%Y%m%d%H%M%S)
    fi

    # 安装新版本
    mv /tmp/v2ray-dash-backend /opt/v2ray-dash/v2ray-dash-server
    chmod +x /opt/v2ray-dash/v2ray-dash-server

    # 重启服务
    systemctl restart v2ray-dash

    echo "后端部署完成"
ENDSSH

echo -e "${GREEN}后端上传完成${NC}"

# 验证部署
echo -e "${YELLOW}[验证] 检查部署结果...${NC}"
sleep 2

# 检查后端服务状态
BACKEND_STATUS=$(sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" "systemctl is-active v2ray-dash")
if [[ "$BACKEND_STATUS" == "active" ]]; then
    echo -e "${GREEN}✓ 后端服务运行正常${NC}"
else
    echo -e "${RED}✗ 后端服务启动失败${NC}"
fi

# 检查前端文件
FRONTEND_FILES=$(sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no -p "$SERVER_PORT" "${SERVER_USER}@${SERVER_IP}" "ls ${FRONTEND_TARGET_DIR}/index.html 2>/dev/null && echo 'OK'")
if [[ "$FRONTEND_FILES" == "OK" ]]; then
    echo -e "${GREEN}✓ 前端文件部署成功${NC}"
else
    echo -e "${RED}✗ 前端文件部署失败${NC}"
fi

# 检查API健康状态
API_HEALTH=$(curl -s --max-time 5 "http://${SERVER_IP}:8080/health" 2>/dev/null || echo "")
if [[ "$API_HEALTH" == *"ok"* ]]; then
    echo -e "${GREEN}✓ API服务健康检查通过${NC}"
else
    echo -e "${YELLOW}⚠ API服务健康检查失败，请手动检查${NC}"
fi

echo ""
echo -e "${GREEN}====== 部署完成 ======${NC}"
echo -e "访问地址: http://${SERVER_IP}"
echo ""