#!/bin/bash
#==========================================
# V2rayDash 阿里云一键部署脚本
# Author: Jat-echo
#==========================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 加载环境配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/env.conf"

echo -e "${GREEN}====== V2rayDash 部署脚本 ======${NC}"

# 检测是否为 root 用户
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}请使用 root 用户或 sudo 运行此脚本${NC}"
    exit 1
fi

#==========================================
# 1. 系统环境准备
#==========================================
echo -e "${GREEN}[1/7] 更新系统并安装基础软件...${NC}"
apt update && apt upgrade -y
apt install -y curl wget git unzip zip ufw

# 配置防火墙
echo -e "${YELLOW}配置防火墙...${NC}"
ufw --force enable
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow 8080/tcp  # Backend API
ufw reload

#==========================================
# 2. 安装 PostgreSQL
#==========================================
echo -e "${GREEN}[2/7] 安装 PostgreSQL...${NC}"

# 安装 PostgreSQL
apt install -y postgresql postgresql-contrib

# 启动服务
systemctl enable postgresql
systemctl start postgresql

# 创建数据库和用户
su - postgres -c "psql -c \"CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';\"" 2>/dev/null || true
su - postgres -c "psql -c \"CREATE DATABASE $DB_NAME OWNER $DB_USER;\"" 2>/dev/null || true
su - postgres -c "psql -c \"ALTER USER $DB_USER CREATEDB;\"" 2>/dev/null || true

# 初始化数据库表结构
su - postgres -c "psql -d $DB_NAME -f $SCRIPT_DIR/init-db.sql" 2>/dev/null || true

# 修复数据库权限（解决 "must be owner" 问题）
echo -e "${YELLOW}修复数据库权限...${NC}"
su - postgres -c "psql -d $DB_NAME -c \"GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;\"" 2>/dev/null || true
su - postgres -c "psql -d $DB_NAME -c \"GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;\"" 2>/dev/null || true
su - postgres -c "psql -d $DB_NAME -c \"GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $DB_USER;\"" 2>/dev/null || true
su - postgres -c "psql -d $DB_NAME -c \"ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $DB_USER;\"" 2>/dev/null || true
su - postgres -c "psql -d $DB_NAME -c \"ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $DB_USER;\"" 2>/dev/null || true

echo -e "${GREEN}PostgreSQL 安装完成${NC}"

#==========================================
# 3. 安装 Go
#==========================================
echo -e "${GREEN}[3/7] 安装 Go...${NC}"

if ! command -v go &> /dev/null; then
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
fi

export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# 回滚函数
rollback() {
    echo -e "${YELLOW}正在回滚...${NC}"
    systemctl stop v2ray-dash 2>/dev/null || true
    if [ -d "$APP_DIR/.git" ]; then
        git -C $APP_DIR checkout HEAD~1 2>/dev/null || true
    fi
    systemctl start v2ray-dash
    echo -e "${GREEN}回滚完成${NC}"
}

echo -e "${GREEN}Go 安装完成${NC}"

#==========================================
# 4. 安装 Node.js
#==========================================
echo -e "${GREEN}[4/7] 安装 Node.js...${NC}"

if ! command -v node &> /dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
    apt install -y nodejs
fi

npm install -g npm@latest

echo -e "${GREEN}Node.js 安装完成${NC}"

#==========================================
# 5. 部署后端
#==========================================
echo -e "${GREEN}[5/7] 部署后端服务...${NC}"

# 创建应用目录
mkdir -p $APP_DIR
mkdir -p $DATA_DIR

# 拉取代码（如果使用 git）
if [ "$USE_GIT" = "true" ]; then
    cd $APP_DIR
    if [ -d ".git" ]; then
        git pull
    else
        git clone $GIT_REPO .
    fi
fi

# 编译后端
cd $APP_DIR/backend
export PATH=$PATH:/usr/local/go/bin
go mod download
go build -o $APP_DIR/v2ray-dash-server ./cmd/server

# 创建 systemd 服务文件
cat > /etc/systemd/system/v2ray-dash.service <<EOF
[Unit]
Description=V2rayDash Backend Service
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
Environment=DATABASE_URL=$DB_URL
ExecStart=$APP_DIR/v2ray-dash-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable v2ray-dash
systemctl restart v2ray-dash

echo -e "${GREEN}后端部署完成${NC}"

#==========================================
# 6. 部署前端
#==========================================
echo -e "${GREEN}[6/7] 部署前端...${NC}"

cd $APP_DIR/frontend
npm install
npm run build

# 配置 Nginx
cp $SCRIPT_DIR/nginx.conf /etc/nginx/sites-available/v2ray-dash
sed -i "s/YOUR_DOMAIN/$DOMAIN/g" /etc/nginx/sites-available/v2ray-dash
sed -i "s|$APP_DIR|$APP_DIR|g" /etc/nginx/sites-available/v2ray-dash

# 启用站点
ln -sf /etc/nginx/sites-available/v2ray-dash /etc/nginx/sites-enabled/
nginx -t

# 安装 SSL（如果配置了域名）
if [ "$USE_SSL" = "true" ]; then
    apt install -y certbot python3-certbot-nginx
    certbot --nginx -d $DOMAIN --non-interactive --agree-tos -m $ADMIN_EMAIL
fi

systemctl reload nginx

#==========================================
# 7. 验证部署
#==========================================
echo -e "${GREEN}[7/7] 验证部署...${NC}"

# 检查后端服务状态
sleep 3
if systemctl is-active --quiet v2ray-dash; then
    echo -e "${GREEN}✓ 后端服务运行中${NC}"
else
    echo -e "${RED}✗ 后端服务启动失败，请检查日志: journalctl -u v2ray-dash -n 50${NC}"
fi

# 检查端口
if curl -sf http://127.0.0.1:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 后端 API 正常${NC}"
else
    echo -e "${YELLOW}⚠ 后端 API 暂时不可用，将在配置域名后恢复${NC}"
fi

# 检查前端
if [ -f "$APP_DIR/frontend/dist/index.html" ]; then
    echo -e "${GREEN}✓ 前端文件已部署${NC}"
else
    echo -e "${RED}✗ 前端文件未找到${NC}"
fi

#==========================================
# 完成
#==========================================
echo -e "${GREEN}====== 部署完成 ======${NC}"
echo ""
echo "访问地址:"
echo "  前端: http://$DOMAIN"
echo "  API:  http://$DOMAIN/api/"
echo "  健康检查: curl http://127.0.0.1:8080/health"
echo ""
echo "常用命令:"
echo "  systemctl status v2ray-dash  # 查看后端状态"
echo "  systemctl restart v2ray-dash # 重启后端"
echo "  journalctl -u v2ray-dash -f   # 查看后端日志"
echo "  nginx -t                     # 检查 Nginx 配置"
echo ""
echo "回滚命令:"
echo "  systemctl stop v2ray-dash"
echo "  git -C $APP_DIR checkout HEAD~1 && systemctl restart v2ray-dash"