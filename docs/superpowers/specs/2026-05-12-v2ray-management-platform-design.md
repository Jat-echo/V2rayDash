# V2ray 服务管理平台 - 设计文档

**日期：** 2026-05-12
**状态：** 已批准

---

## 1. 项目概述

**目的：** 构建一个 Web 管理平台，用于管理多台 v2ray 代理服务器，实现服务器管理、用户订阅账号管理、健康度监控等功能。

**用户：** 个人使用，管理自己的多台 v2ray 服务器。

**技术栈：**
- 前端：React + Ant Design Pro
- 后端：Go + Gin
- 数据库：PostgreSQL
- Agent：Go（编译成单文件，随 install.sh 一起安装）

---

## 2. 整体架构

```
                        ┌─────────────────────────────────┐
                        │  控制中心（云服务器 2C2G）        │
                        │                                 │
  ┌─────────┐           │  ┌─────────┐  ┌──────────┐     │
  │  你    │──浏览器──▶│  │  React  │──│   Go     │     │
  │        │           │  │  Frontend│  │   API   │     │
  └─────────┘           │  └─────────┘  └────┬─────┘     │
                        │                    │           │
                        │              ┌─────▼─────┐     │
                        │              │ PostgreSQL│     │
                        │              └───────────┘     │
                        │                    ▲           │
                        │              ┌─────┴─────┐     │
                        │              │ Agent     │     │
                        │              │ Receiver  │     │
                        │              └───────────┘     │
                        └──────────────────┬───────────┘
                                           │ HTTP/WebSocket
                        ┌──────────────────┴───────────┐
                        │         节点（v2ray服务器）    │
                        │  ┌─────────┐    ┌──────────┐  │
                        │  │ v2ray  │    │  Agent   │──┼──▶ 上报状态
                        │  └─────────┘    └──────────┘  │
                        └─────────────────────────────────┘
```

**通信方式：** 控制中心有公网IP，Agent 定期 POST 上报状态到控制中心。

---

## 3. 核心模块

| 模块 | 技术 | 说明 |
|------|------|------|
| 前端 | React + Ant Design Pro | Web管理界面 |
| 后端API | Go + Gin | RESTful API |
| 数据库 | PostgreSQL | 存储服务器、用户、订阅配置 |
| Agent | Go（编译成单文件） | 节点轻量客户端，随 install.sh 一起装 |
| 通信 | HTTP POST + WebSocket | Agent主动上报，API被动接收 |

---

## 4. 功能模块

### 4.1 服务器管理

- 添加/删除/编辑服务器（IP、SSH端口、凭证）
- 服务器分组/标签
- 一键执行 install.sh 安装/重装 v2ray
- SSH 远程命令执行
- 批量操作支持

### 4.2 订阅账号管理

- 添加/编辑/删除订阅账号
- 账号与服务器关联（一台服务器多个账号）
- 生成订阅链接（Base64编码）
- 流量统计（Agent上报）
- 账号启用/禁用
- 订阅链接二维码生成

### 4.3 健康度监控

- 节点 CPU、内存、带宽、硬盘 使用率
- v2ray 服务状态（运行中/已停止）
- 在线状态心跳（Agent每30秒上报）
- Web界面实时展示
- 历史数据图表

### 4.4 日志系统

#### 操作日志（谁做了什么）

| 操作类型 | 记录内容 | 保留时间 |
|---------|---------|---------|
| 服务器增删改 | 操作人、时间、对象、变更前后 | 永久 |
| 订阅账号管理 | 操作人、时间、账号、操作类型 | 永久 |
| SSH命令执行 | 操作人、时间、服务器、执行的命令、结果 | 30天 |
| 登录日志 | 操作人、时间、IP、浏览器 | 永久 |

#### 系统日志（平台自身运行状况）

| 日志类型 | 说明 |
|---------|------|
| API请求日志 | 请求时间、接口、耗时、状态码 |
| Agent上报日志 | 各节点心跳状态、上报异常 |
| 定时任务日志 | 监控采集结果、告警触发记录 |
| 错误日志 | 平台异常、数据库连接失败等 |

**存储方案：**
- 操作日志 → PostgreSQL（方便查询）
- 系统日志 → 文件存储 + Logrotate（控制大小）
- 可选：接入 Loki/Prometheus 做可视化（后续）

**日志展示：** Web界面提供"操作日志"页面，支持筛选：
- 按时间范围
- 按服务器
- 按操作类型
- 按操作人

---

## 5. 数据模型

### 5.1 服务器表 (servers)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| name | varchar(100) | 服务器名称 |
| ip | varchar(45) | IP地址 |
| ssh_port | int | SSH端口 |
| ssh_user | varchar(50) | SSH用户名 |
| ssh_key_path | text | SSH私钥路径（加密存储） |
| tags | jsonb | 标签/分组 |
| status | varchar(20) | online/offline/unknown |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 5.2 订阅账号表 (subscriptions)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| server_id | uuid | 关联服务器 |
| name | varchar(100) | 账号名称 |
| uuid | varchar(36) | v2ray用户UUID |
| enable | boolean | 是否启用 |
| traffic_limit | bigint | 流量限制(bytes) |
| traffic_used | bigint | 已用流量 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 5.3 操作日志表 (operation_logs)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| operator | varchar(50) | 操作人 |
| action | varchar(50) | 操作类型 |
| target_type | varchar(30) | 目标类型（server/subscription） |
| target_id | uuid | 目标ID |
| detail | jsonb | 变更详情 |
| ip | varchar(45) | 操作来源IP |
| created_at | timestamp | 操作时间 |

### 5.4 节点状态表 (node_status)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uuid | 主键 |
| server_id | uuid | 关联服务器 |
| cpu_percent | float | CPU使用率 |
| memory_percent | float | 内存使用率 |
| disk_percent | float | 硬盘使用率 |
| bandwidth_in | bigint | 入带宽 |
| bandwidth_out | bigint | 出带宽 |
| v2ray_status | varchar(20) | v2ray状态 |
| reported_at | timestamp | 上报时间 |

---

## 6. API 设计

### 6.1 服务器管理

- `GET /api/servers` - 列表
- `POST /api/servers` - 添加
- `PUT /api/servers/:id` - 编辑
- `DELETE /api/servers/:id` - 删除
- `POST /api/servers/:id/install` - 执行安装
- `POST /api/servers/:id/command` - 执行SSH命令

### 6.2 订阅管理

- `GET /api/subscriptions` - 列表
- `POST /api/subscriptions` - 添加
- `PUT /api/subscriptions/:id` - 编辑
- `DELETE /api/subscriptions/:id` - 删除
- `GET /api/subscriptions/:id/link` - 获取订阅链接

### 6.3 监控

- `GET /api/servers/:id/status` - 获取节点状态
- `GET /api/servers/:id/history` - 历史监控数据

### 6.4 日志

- `GET /api/logs/operation` - 操作日志列表
- `GET /api/logs/system` - 系统日志

### 6.5 Agent上报

- `POST /api/agent/heartbeat` - Agent心跳
- `POST /api/agent/status` - 上报节点状态
- `GET /api/agent/config/:server_id` - 获取最新配置

---

## 7. Agent 设计

### 7.1 功能

- 心跳上报（每30秒）
- 节点状态采集（CPU、内存、带宽、硬盘）
- v2ray服务状态检测
- 配置文件更新（从控制中心拉取）
- 日志上报

### 7.2 通信

- 启动时从控制中心获取配置（服务器ID、API地址等）
- 定期 POST 心跳和状态到 `http://控制中心:8080/api/agent/*`
- 支持 WebSocket 实时通道（备选）

### 7.3 集成

- Agent 编译成单文件，随 install.sh 一起分发
- 安装时通过 install.sh 的某个选项安装 agent

---

## 8. 安全考虑

- SSH 凭证加密存储
- API 认证（JWT token）
- Agent 与控制中心通信使用预共享密钥验证
- 操作日志完整记录
- 定期备份数据库

---

## 9. 部署架构

### 控制中心（云服务器）

- 操作系统：Ubuntu 22.04 LTS
- 资源配置：2核2G
- 开放端口：80/443（Web）、8080（Agent通信）

### 节点（v2ray服务器）

- 支持系统：CentOS 7+/Debian 10+/Ubuntu 18.04+/Alpine
- 通过 install.sh 安装 v2ray + Agent

---

## 10. 实施计划

### Phase 1：基础框架
- 项目结构搭建（Go API + React Frontend）
- 数据库设计与迁移
- 基础的服务器管理功能

### Phase 2：Agent集成
- Agent 开发
- 心跳与状态上报
- 与 install.sh 集成

### Phase 3：订阅管理
- 订阅账号 CRUD
- 订阅链接生成
- 流量统计

### Phase 4：监控与日志
- 监控数据展示
- 历史图表
- 操作日志
- 系统日志

---

**审批状态：** 已通过
**审批日期：** 2026-05-12