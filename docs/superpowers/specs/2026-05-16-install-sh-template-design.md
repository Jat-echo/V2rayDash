# 改造 install.sh 支持配置模板 - 设计文档

**日期：** 2026-05-16
**状态：** 已批准

---

## 1. 目标

将 install.sh 的所有交互式输入改为可通过命令行参数或配置文件传入，实现：
- **配置模板化** - 在 Web 界面选择/配置模板，生成安装命令
- **一键安装** - 节点执行一条命令完成 v2ray + Agent 安装
- **零交互** - 安装过程完全不需要人机交互

---

## 2. 设计原则

| 原则 | 说明 |
|------|------|
| 向后兼容 | 原有交互式安装方式保持不变 |
| 参数优先 | 命令行参数 > 配置文件 > 默认值 |
| 渐进增强 | 先实现核心场景，再扩展功能 |

---

## 3. 配置模板数据结构

### 3.1 完整模板结构

```json
{
  "id": "template-uuid",
  "name": "标准Reality配置",
  "description": "443端口 VLESS Reality Vision，适用于大多数场景",
  "version": 1,

  "core": {
    "type": "sing-box",
    "version": "latest"
  },

  "protocols": {
    "vless_reality_vision": {
      "enabled": true,
      "port": 443,
      "uuid": "",
      "email": "",
      "server_name": "download-installer.cdn.mozilla.net",
      "short_ids": ["", "6ba85179e30d4fc2"]
    },
    "vless_tcp_vision": {
      "enabled": false,
      "port": 4430
    },
    "hysteria2": {
      "enabled": false,
      "port": 44300,
      "down_speed": 100,
      "up_speed": 50
    },
    "tuic": {
      "enabled": false,
      "port": 44302
    }
  },

  "tls": {
    "type": "none"
  },

  "nginx_blog": {
    "enabled": true,
    "redirect_enabled": false
  },

  "agent": {
    "enabled": true,
    "report_interval": 30
  }
}
```

### 3.2 模板字段说明

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|-------|
| `core.type` | string | 核心类型：`sing-box` / `xray-core` | `sing-box` |
| `protocols.vless_reality_vision.enabled` | bool | 是否启用 | `false` |
| `protocols.vless_reality_vision.port` | int | 监听端口 | `443` |
| `protocols.vless_reality_vision.uuid` | string | UUID，空=自动生成 | `""` |
| `protocols.vless_reality_vision.server_name` | string | Reality目标域名 | - |
| `protocols.vless_reality_vision.short_ids` | array | Short ID列表 | `[""]` |
| `protocols.hysteria2.enabled` | bool | 是否启用Hysteria2 | `false` |
| `nginx_blog.enabled` | bool | 是否安装伪装站点 | `true` |
| `agent.enabled` | bool | 是否安装Agent | `false` |
| `agent.report_interval` | int | 上报间隔(秒) | `30` |

---

## 4. 命令行参数设计

### 4.1 参数列表

| 参数 | 说明 | 示例 |
|------|------|------|
| `--agent` | 启用Agent安装模式 | - |
| `--url <url>` | 控制中心地址 | `https://v2ray.example.com:8080` |
| `--id <uuid>` | 服务器ID | `a1b2c3d4-...` |
| `--psk <key>` | 预共享密钥（自动生成） | - |
| `--template <name>` | 使用预设模板名 | `standard` |
| `--config <base64>` | Base64编码的完整配置 | - |
| `--core <type>` | 核心类型 | `sing-box` |
| `--protocol <type>` | 协议类型 | `vless_reality_vision` |
| `--port <port>` | 端口 | `443` |
| `--uuid <uuid>` | 自定义UUID，空=自动 | - |
| `--server-name <domain>` | Reality目标域名 | `download-installer.cdn.mozilla.net` |

### 4.2 安装命令示例

```bash
# 方式1: 使用模板
curl -sL https://cdn/install.sh | bash -s -- \
  --template standard-reality \
  --agent \
  --url https://v2ray.example.com:8080 \
  --id a1b2c3d4-e5f6-7890 \
  --psk auto

# 方式2: 直接指定参数
curl -sL https://cdn/install.sh | bash -s -- \
  --agent \
  --core sing-box \
  --protocol vless_reality_vision \
  --port 443 \
  --server-name download-installer.cdn.mozilla.net \
  --url https://v2ray.example.com:8080 \
  --id a1b2c3d4-e5f6-7890 \
  --psk auto

# 方式3: 使用Base64配置
curl -sL https://cdn/install.sh | bash -s -- \
  --config <base64-encoded-config>
```

---

## 5. Web界面设计

### 5.1 配置模板管理页面

```
┌─────────────────────────────────────────────────────────────────┐
│  配置模板管理                                          [+ 新建] │
├────────────────────────────┬────────────────────────────────────┤
│  模板列表                  │  编辑模板: 标准Reality配置         │
│  ┌──────────────────────┐  │  ┌────────────────────────────┐   │
│  │ ● standard-reality   │  │  │ 模板名称: [标准Reality配置] │   │
│  │   minimal-reality   │  │  ├────────────────────────────┤   │
│  │   full-feature      │  │  │ 核心选择                    │   │
│  └──────────────────────┘  │  │ (•) sing-box  ( ) xray     │   │
│                            │  ├────────────────────────────┤   │
│  [复制] [删除] [导出]      │  │ 协议配置                    │   │
│                            │  │ [✓] VLESS Reality Vision   │   │
│                            │  │     端口: [443________]    │   │
│                            │  │     UUID: (•)自动 ( )手动 │   │
│                            │  │     目标域名: [mozilla.net]│   │
│                            │  │                             │   │
│                            │  │ [ ] Hysteria2               │   │
│                            │  │ [ ] Tuic                    │   │
│                            │  ├────────────────────────────┤   │
│                            │  │ 伪装站点: (•)启用 ( )禁用  │   │
│                            │  ├────────────────────────────┤   │
│                            │  │ Agent: (•)启用 ( )禁用      │   │
│                            │  │ 上报间隔: [30] 秒          │   │
│                            │  └────────────────────────────┘   │
│                            │                                    │
│                            │  [保存模板]  [生成安装命令]          │
└────────────────────────────┴────────────────────────────────────┘
```

### 5.2 生成安装命令

```
┌─────────────────────────────────────────────────────────────────┐
│  生成安装命令                                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  服务器ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890                 │
│  PSK密钥: (已自动生成) xK9#mP2$L8k@2...                         │
│                                                                 │
│  安装命令:                                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ curl -sL https://cdn/install.sh | bash -s -- \          │   │
│  │   --agent \                                             │   │
│  │   --template standard-reality \                         │   │
│  │   --url https://v2ray.example.com:8080 \               │   │
│  │   --id a1b2c3d4-e5f6-7890-abcd-ef1234567890 \           │   │
│  │   --psk xK9#mP2$L8k@2...                               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  [复制命令]                              [下载配置到节点]      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. 安装流程

### 6.1 节点安装流程

```
1. 用户在控制中心添加服务器
   └─> 生成 server_id 和 psk
   └─> 复制安装命令

2. 用户在节点执行安装命令
   └─> curl 下载 install.sh
   └─> bash -s -- 参数解析

3. install.sh 执行安装
   └─> 解析 --template 或 --config
   └─> 设置 selectCoreType、selectCustomInstallType 等变量
   └─> 调用 installSingBoxReality 或对应安装函数
   └─> 安装 v2ray-core / sing-box
   └─> 配置协议（端口、UUID、Reality等）

4. 如果 --agent
   └─> 下载 agent 二进制
   └─> 生成 /etc/v2ray-agent/agent.json
   └─> 安装 systemd 服务
   └─> 启动 agent
   └─> 验证连通性
```

### 6.2 参数解析逻辑

```bash
# 伪代码
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --agent)
            agent_mode=true
            ;;
        --template)
            load_template "$2"
            shift
            ;;
        --config)
            decode_config "$2"
            shift
            ;;
        --url|--id|--psk|--core|--port|--uuid|--server-name)
            set_var_from_arg "$1" "$2"
            shift
            ;;
        esac
        shift
    done
}

# 如果是 agent 模式但没有提供 --url/--id/--psk，报错
if [[ "${agent_mode}" == "true" && -z "${control_center_url}" ]]; then
    echo "Error: --url required for agent mode"
    exit 1
fi
```

---

## 7. 实现步骤

### Phase 1: install.sh 改造
- 添加命令行参数解析
- 添加 `--template` 和 `--config` 支持
- 改造现有函数支持非交互模式

### Phase 2: 模板管理 API
- 添加模板 CRUD 接口
- 模板存储到 PostgreSQL

### Phase 3: Web 前端
- 模板管理页面
- 安装命令生成
- 一键复制

### Phase 4: 测试验证
- 本地测试安装流程
- 验证 Agent 上报

---

## 8.向后兼容

| 原有安装方式 | 影响 |
|------------|------|
| `curl ... \| bash` 无参数 | 保持原有交互式安装 |
| `menu` 选择 | 保持原有交互式 |
| 新参数 | 仅在传入时生效 |

---

**审批状态：** 已通过
**审批日期：** 2026-05-16