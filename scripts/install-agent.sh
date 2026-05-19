#!/usr/bin/env bash
# V2rayDash Agent 安装脚本
# 用法: curl -sL http://your-control-center:8080/install-agent.sh | bash -s -- --server-id <id> --center http://your-control-center:8080

set -e

SERVER_ID=""
CONTROL_CENTER=""
AGENT_PORT=9090
AGENT_DIR="/opt/v2ray-dash-agent"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --server-id)
            SERVER_ID="$2"
            shift 2
            ;;
        --center)
            CONTROL_CENTER="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

if [[ -z "$SERVER_ID" ]] || [[ -z "$CONTROL_CENTER" ]]; then
    log_error "缺少必要参数"
    echo "用法: $0 --server-id <id> --center <control-center-url>"
    exit 1
fi

# 下载 Agent 二进制
download_agent() {
    log_info "下载 Agent..."

    # 获取架构
    ARCH=$(uname -m)
    if [[ "$ARCH" == "x86_64" ]]; then
        ARCH="amd64"
    elif [[ "$ARCH" == "aarch64" ]]; then
        ARCH="arm64"
    fi

    # 这里应该从控制中心下载实际的 agent 二进制
    # 暂时使用一个占位脚本
    mkdir -p $AGENT_DIR

    cat > $AGENT_DIR/agent.sh << 'AGENT_SCRIPT'
#!/bin/bash
# V2rayDash Agent - 轻量级监控客户端

SERVER_ID="__SERVER_ID__"
CONTROL_CENTER="__CONTROL_CENTER__"
INTERVAL=30

while true; do
    # 收集 CPU 使用率
    CPU=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | sed 's/%us,//')
    if [[ -z "$CPU" ]]; then CPU="0"; fi

    # 收集内存使用率
    MEM=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100}')
    if [[ -z "$MEM" ]]; then MEM="0"; fi

    # 收集磁盘使用率
    DISK=$(df / | tail 1 | awk '{print $5}' | sed 's/%//')
    if [[ -z "$DISK" ]]; then DISK="0"; fi

    # 检查 v2ray/xray 状态
    if pgrep -x xray > /dev/null || pgrep -x v2ray > /dev/null; then
        V2RAY_STATUS="running"
    else
        V2RAY_STATUS="stopped"
    fi

    # 计算流量 (从 /proc/net/dev)
    BANWIDTH_IN=$(cat /proc/net/dev | grep -i eth0 | awk '{print $2}' | head -1)
    BANWIDTH_OUT=$(cat /proc/net/dev | grep -i eth0 | awk '{print $10}' | head -1)

    # 发送心跳
    JSON_DATA=$(cat <<EOF
{
    "server_id": "$SERVER_ID",
    "cpu_percent": $CPU,
    "mem_percent": $MEM,
    "disk_percent": $DISK,
    "bandwidth_in": ${BANWIDTH_IN:-0},
    "bandwidth_out": ${BANWIDTH_OUT:-0},
    "v2ray_status": "$V2RAY_STATUS"
}
EOF
)

    curl -s -X POST "$CONTROL_CENTER/api/agent/heartbeat" \
        -H "Content-Type: application/json" \
        -d "$JSON_DATA" > /dev/null 2>&1

    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Heartbeat sent - CPU: ${CPU}% MEM: ${MEM}% DISK: ${DISK}% V2ray: $V2RAY_STATUS"

    sleep $INTERVAL
done
AGENT_SCRIPT

    # 替换变量
    sed -i "s/__SERVER_ID__/$SERVER_ID/g" $AGENT_DIR/agent.sh
    sed -i "s/__CONTROL_CENTER__/$CONTROL_CENTER/g" $AGENT_DIR/agent.sh

    chmod +x $AGENT_DIR/agent.sh
    log_info "Agent 安装完成"
}

# 配置 systemd 服务
setup_service() {
    log_info "配置系统服务..."

    cat > /etc/systemd/system/v2ray-dash-agent.service << EOF
[Unit]
Description=V2rayDash Agent
After=network.target

[Service]
ExecStart=$AGENT_DIR/agent.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable v2ray-dash-agent
    systemctl start v2ray-dash-agent

    log_info "Agent 服务已启动"
    systemctl status v2ray-dash-agent --no-pager
}

# 主流程
main() {
    log_info "开始安装 V2rayDash Agent..."
    log_info "服务器ID: $SERVER_ID"
    log_info "控制中心: $CONTROL_CENTER"

    download_agent
    setup_service

    log_info "安装完成！"
}

main