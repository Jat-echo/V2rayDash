-- 服务器表
CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    ssh_port INTEGER DEFAULT 22,
    ssh_user VARCHAR(50) DEFAULT 'root',
    ssh_key_type VARCHAR(10) DEFAULT 'key',
    ssh_key TEXT DEFAULT NULL,
    ssh_password TEXT DEFAULT NULL,
    tags JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'unknown',
    reality_enabled BOOLEAN DEFAULT false,
    reality_server_name VARCHAR(255) DEFAULT '',
    reality_public_key VARCHAR(255) DEFAULT '',
    reality_port INTEGER DEFAULT 443,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 订阅表
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    enable BOOLEAN DEFAULT true,
    traffic_limit BIGINT DEFAULT 0,
    traffic_used BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 订阅账号关联表 (多对多)
CREATE TABLE IF NOT EXISTS subscription_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_subscription_per_account UNIQUE (subscription_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_subscription_accounts_subscription ON subscription_accounts(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_accounts_account ON subscription_accounts(account_id);

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(30),
    target_id UUID,
    detail JSONB,
    ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 节点状态表
CREATE TABLE IF NOT EXISTS node_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    cpu_percent FLOAT,
    memory_percent FLOAT,
    disk_percent FLOAT,
    bandwidth_in BIGINT DEFAULT 0,
    bandwidth_out BIGINT DEFAULT 0,
    v2ray_status VARCHAR(20) DEFAULT 'unknown',
    reported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at ON operation_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_node_status_server_id ON node_status(server_id);
CREATE INDEX IF NOT EXISTS idx_node_status_reported_at ON node_status(reported_at);

-- 模板表
CREATE TABLE IF NOT EXISTS templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 账号表
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    uuid VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    protocols TEXT[] NOT NULL,
    enabled BOOLEAN DEFAULT true,
    traffic_limit BIGINT DEFAULT 0,
    traffic_used BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 订阅记录表（可选，用于追踪）
CREATE TABLE IF NOT EXISTS subscriptions_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    sub_type VARCHAR(50) NOT NULL,
    sub_link TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_server_id ON accounts(server_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_records_account_id ON subscriptions_records(account_id);

-- 系统设置表
CREATE TABLE IF NOT EXISTS system_settings (
    id VARCHAR(50) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 初始化默认设置
INSERT INTO system_settings (id, value, description) VALUES
    ('public_url', 'http://localhost:8080', '控制中心公网访问地址')
ON CONFLICT (id) DO NOTHING;
