# install.sh 配置模板化 - 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 install.sh 改造成支持命令行参数和配置模板，实现零交互一键安装 v2ray + Agent。

**Architecture:** 在 install.sh 中添加参数解析层，解析 --template/--config/--agent 等参数，设置内部变量后调用现有安装函数。

**Tech Stack:** Bash (install.sh), Go (Agent), React (Web)

---

## Phase 1: install.sh 参数解析

### Task 1: 添加参数解析框架

**Files:**
- Modify: `install.sh` (在文件开头 initVar 函数后添加参数解析)

- [ ] **Step 1: 创建参数解析函数**

在 install.sh 开头添加以下函数（在 `initVar` 函数后）:

```bash
# 配置模板模式
template_mode=false
template_name=""
template_config=""

# Agent 配置
agent_mode=false
agent_server_id=""
agent_control_center_url=""
agent_psk=""

# 显式参数
explicit_core_type=""
explicit_protocol=""
explicit_port=""
explicit_uuid=""
explicit_server_name=""

# 解析命令行参数
parse_cli_args() {
    if [[ $# -eq 0 ]]; then
        return
    fi

    while [[ $# -gt 0 ]]; do
        case "$1" in
        # Agent模式
        --agent)
            agent_mode=true
            shift
            ;;

        # 控制中心配置
        --url)
            agent_control_center_url="$2"
            shift 2
            ;;
        --id)
            agent_server_id="$2"
            shift 2
            ;;
        --psk)
            agent_psk="$2"
            shift 2
            ;;

        # 模板模式
        --template)
            template_mode=true
            template_name="$2"
            shift 2
            ;;
        --config)
            template_mode=true
            template_config="$2"
            shift 2
            ;;

        # 核心和协议配置
        --core)
            explicit_core_type="$2"
            shift 2
            ;;
        --protocol)
            explicit_protocol="$2"
            shift 2
            ;;
        --port)
            explicit_port="$2"
            shift 2
            ;;
        --uuid)
            explicit_uuid="$2"
            shift 2
            ;;
        --server-name)
            explicit_server_name="$2"
            shift 2
            ;;
        --template-name)
            template_name="$2"
            shift 2
            ;;

        # 兼容旧参数
        -a|--auto)
            template_mode=true
            template_name="standard"
            shift
            ;;

        *)
            echo "Unknown option: $1"
            shift
            ;;
        esac
    done

    # Agent模式校验
    if [[ "${agent_mode}" == "true" ]]; then
        if [[ -z "${agent_control_center_url}" ]]; then
            echoContent red "Error: --url required for agent mode"
            exit 1
        fi
        if [[ -z "${agent_server_id}" ]]; then
            echoContent red "Error: --id required for agent mode"
            exit 1
        fi
        if [[ -z "${agent_psk}" || "${agent_psk}" == "auto" ]]; then
            # 自动生成PSK
            agent_psk=$(openssl rand -base64 32)
        fi
    fi
}
```

- [ ] **Step 2: 在脚本入口处调用解析函数**

找到脚本最后的 `menu` 调用（约line 10079），在调用前添加:

```bash
# 解析命令行参数
parse_cli_args "$@"
```

- [ ] **Step 3: 添加帮助信息**

```bash
show_help() {
    echoContent green "install.sh 帮助信息"
    echoContent yellow "用法: curl -sL https://xxx/install.sh | bash -s -- [选项]"
    echo
    echoContent yellow "选项:"
    echoContent yellow "  --agent                    启用Agent安装模式"
    echoContent yellow "  --url <url>                控制中心地址"
    echoContent yellow "  --id <uuid>                服务器ID"
    echoContent yellow "  --psk <key>                PSK密钥 (auto=自动生成)"
    echoContent yellow "  --template <name>          使用预设模板"
    echoContent yellow "  --config <base64>          Base64编码的配置"
    echoContent yellow "  --core <type>              核心类型 (sing-box/xray-core)"
    echoContent yellow "  --protocol <type>          协议类型"
    echoContent yellow "  --port <port>              端口"
    echoContent yellow "  --uuid <uuid>              自定义UUID"
    echoContent yellow "  --server-name <domain>     Reality目标域名"
    echo
    echoContent yellow "示例:"
    echoContent yellow "  # Agent模式安装"
    echoContent yellow "  curl -sL ... | bash -s -- --agent --url https://... --id xxx --psk auto"
    echoContent yellow "  # 使用模板"
    echoContent yellow "  curl -sL ... | bash -s -- --template standard-reality"
}
```

- [ ] **Step 4: 提交**

```bash
git add install.sh
git commit -m "feat: add CLI argument parsing framework"
```

---

### Task 2: 实现模板加载逻辑

**Files:**
- Modify: `install.sh` (添加模板加载和变量设置函数)

- [ ] **Step 1: 添加模板映射表**

```bash
# 预设模板映射
declare -A TEMPLATE_MAP=(
    ["minimal-reality"]="sing-box|,7,,|,0,,"
    ["standard-reality"]="sing-box|,7,|,0,"
    ["full-reality"]="sing-box|,7,|,0,|,8,|,6,|,9,"
)

# 加载模板
load_template() {
    local template_name="$1"
    local template_spec="${TEMPLATE_MAP[${template_name}]}"

    if [[ -z "${template_spec}" ]]; then
        echoContent red "Unknown template: ${template_name}"
        exit 1
    fi

    # 解析模板规格: core_type|protocol_types
    local core_type=$(echo "${template_spec}" | cut -d'|' -f1)
    local protocol_types=$(echo "${template_spec}" | cut -d'|' -f2-)

    # 设置核心类型
    if [[ "${core_type}" == "sing-box" ]]; then
        selectCoreType=2
        coreInstallType=2
    else
        selectCoreType=1
        coreInstallType=1
    fi

    # 设置协议类型
    selectCustomInstallType="${protocol_types}"
}
```

- [ ] **Step 2: 在 parse_cli_args 后调用模板加载**

在 `parse_cli_args` 函数后添加:

```bash
# 如果指定了模板，加载模板配置
if [[ "${template_mode}" == "true" && -n "${template_name}" ]]; then
    load_template "${template_name}"
fi

# 如果指定了核心类型，覆盖模板设置
if [[ -n "${explicit_core_type}" ]]; then
    if [[ "${explicit_core_type}" == "sing-box" ]]; then
        selectCoreType=2
        coreInstallType=2
    else
        selectCoreType=1
        coreInstallType=1
    fi
fi

# 如果指定了协议，覆盖模板设置
if [[ -n "${explicit_protocol}" ]]; then
    case "${explicit_protocol}" in
    vless_reality_vision)
        selectCustomInstallType=",7,"
        ;;
    vless_tcp_vision)
        selectCustomInstallType=",0,"
        ;;
    all)
        selectCustomInstallType="all"
        ;;
    esac
fi
```

- [ ] **Step 3: 添加自动变量设置**

```bash
# 应用显式参数覆盖
apply_explicit_args() {
    # 端口
    if [[ -n "${explicit_port}" ]]; then
        singBoxVLESSRealityVisionPort="${explicit_port}"
        singBoxVLESSVisionPort="${explicit_port}"
    fi

    # UUID
    if [[ -n "${explicit_uuid}" ]]; then
        currentUUID="${explicit_uuid}"
    fi

    # Reality目标域名
    if [[ -n "${explicit_server_name}" ]]; then
        realityServerName="${explicit_server_name}"
    fi
}
```

- [ ] **Step 4: 在安装流程开始前调用 apply_explicit_args**

在 `selectCoreInstall` 函数开始处（或各安装函数开始处）调用:

```bash
apply_explicit_args
```

- [ ] **Step 5: 提交**

```bash
git add install.sh
git commit -m "feat: add template loading and explicit args override"
```

---

### Task 3: 添加 Agent 安装逻辑

**Files:**
- Modify: `install.sh` (在文件末尾添加 Agent 安装函数)

- [ ] **Step 1: 创建 Agent 安装函数**

```bash
# 安装 Agent
install_agent() {
    echoContent skyBlue "\n进度 X/Y : 安装Agent"

    # 下载 Agent 二进制
    local agent_url="${agent_control_center_url}/agents/latest/linux_$(uname -m)/agent"
    local agent_bin="/usr/local/bin/v2ray-agent"

    echoContent green " ---> 下载Agent: ${agent_url}"
    wget -q -O "${agent_bin}" "${agent_url}"
    chmod +x "${agent_bin}"

    # 创建配置目录
    mkdir -p /etc/v2ray-agent

    # 生成配置文件
    cat > /etc/v2ray-agent/agent.json <<EOF
{
    "server_id": "${agent_server_id}",
    "control_center_url": "${agent_control_center_url}",
    "psk": "${agent_psk}",
    "report_interval": 30
}
EOF

    # 创建 systemd 服务
    cat > /etc/systemd/system/v2ray-agent.service <<EOF
[Unit]
Description=V2ray Agent
After=network.target

[Service]
ExecStart=${agent_bin} -config /etc/v2ray-agent/agent.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    # 启动服务
    systemctl daemon-reload
    systemctl enable v2ray-agent
    systemctl start v2ray-agent

    echoContent green " ---> Agent安装完成"
    echoContent yellow " ---> 上报间隔: 30秒"
    echoContent yellow " ---> 控制中心: ${agent_control_center_url}"
}
```

- [ ] **Step 2: 在安装完成后调用 install_agent**

在 `installSingBoxReality` 函数最后（约line 8459）添加:

```bash
# 如果是agent模式，安装agent
if [[ "${agent_mode}" == "true" ]]; then
    install_agent
fi
```

- [ ] **Step 3: 提交**

```bash
git add install.sh
git commit -m "feat: add agent installation to installSingBoxReality"
```

---

## Phase 2: Web端模板管理

### Task 4: 添加模板 API

**Files:**
- Create: `backend/internal/model/template.go`
- Create: `backend/internal/repository/template.go`
- Create: `backend/internal/handler/template.go`
- Modify: `backend/internal/handler/routes.go`

- [ ] **Step 1: 创建模板模型**

```go
// backend/internal/model/template.go
package model

import (
	"time"
)

type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Config      TemplateConfig `json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TemplateConfig struct {
	Core   string `json:"core"`   // "sing-box" or "xray-core"
	Port   int    `json:"port"`
	UUID   string `json:"uuid"`   // empty = auto
	ServerName string `json:"server_name"`
	Protocols    []string `json:"protocols"`
	AgentEnabled bool    `json:"agent_enabled"`
	ReportInterval int   `json:"report_interval"`
}
```

- [ ] **Step 2: 创建模板 Repository**

```go
// backend/internal/repository/template.go
package repository

import (
	"database/sql"
	"encoding/json"

	"v2ray-dash/backend/internal/model"
)

type TemplateRepository struct {
	db *sql.DB
}

func NewTemplateRepository(db *sql.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(tmpl *model.Template) error {
	configJSON, _ := json.Marshal(tmpl.Config)
	result, err := r.db.Exec(
		`INSERT INTO templates (name, description, config) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		tmpl.Name, tmpl.Description, configJSON,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	tmpl.ID = string(rune(id))
	return nil
}

func (r *TemplateRepository) List() ([]*model.Template, error) {
	rows, err := r.db.Query(`SELECT id, name, description, config, created_at, updated_at FROM templates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.Template
	for rows.Next() {
		var t model.Template
		var configJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &configJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(configJSON, &t.Config)
		templates = append(templates, &t)
	}
	return templates, nil
}
```

- [ ] **Step 3: 创建模板 Handler**

```go
// backend/internal/handler/template.go
package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type TemplateHandler struct {
	repo *repository.TemplateRepository
}

func NewTemplateHandler(db *sql.DB) *TemplateHandler {
	return &TemplateHandler{repo: repository.NewTemplateRepository(db)}
}

func (h *TemplateHandler) List(c *gin.Context) {
	templates, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (h *TemplateHandler) Create(c *gin.Context) {
	var tmpl model.Template
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(&tmpl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tmpl)
}
```

- [ ] **Step 4: 注册路由**

在 `backend/internal/handler/routes.go` 添加:

```go
// 模板管理
templateHandler := NewTemplateHandler(db)
api.GET("/templates", templateHandler.List)
api.POST("/templates", templateHandler.Create)
```

- [ ] **Step 5: 添加数据库表**

在 `backend/pkg/database/schema.sql` 添加:

```sql
CREATE TABLE IF NOT EXISTS templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Step 6: 提交**

```bash
git add backend/internal/model/template.go
git add backend/internal/repository/template.go
git add backend/internal/handler/template.go
git add backend/internal/handler/routes.go
git add backend/pkg/database/schema.sql
git commit -m "feat: add template management API"
```

---

### Task 5: Web前端模板管理页面

**Files:**
- Create: `frontend/src/pages/templates/index.tsx`
- Modify: `frontend/src/App.tsx` (添加路由)

- [ ] **Step 1: 创建模板管理页面**

```tsx
// frontend/src/pages/templates/index.tsx
import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, Switch, message } from 'antd'

interface Template {
  id: number
  name: string
  description: string
  config: {
    core: string
    port: number
    uuid: string
    server_name: string
    protocols: string[]
    agent_enabled: boolean
  }
}

export default function TemplateList() {
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadTemplates()
  }, [])

  const loadTemplates = async () => {
    setLoading(true)
    try {
      const data = await api.get('/templates')
      setTemplates(data)
    } catch (e) {
      message.error('加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async (values: any) => {
    try {
      await api.post('/templates', values)
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      loadTemplates()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const generateInstallCommand = (template: Template, serverId: string, psk: string) => {
    const baseURL = window.location.origin
    return `curl -sL ${baseURL}/install.sh | bash -s -- \\
  --agent \\
  --template ${template.name.toLowerCase().replace(/\\s+/g, '-')} \\
  --url ${baseURL} \\
  --id ${serverId} \\
  --psk ${psk}`
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: '描述', dataIndex: 'description' },
    { title: '核心', dataIndex: ['config', 'core'] },
    { title: '端口', dataIndex: ['config', 'port'] },
    { title: 'Agent', dataIndex: ['config', 'agent_enabled'], render: (v: boolean) => v ? '是' : '否' },
    {
      title: '操作',
      render: (_: any, record: Template) => (
        <Space>
          <Button size="small">编辑</Button>
          <Button size="small" type="primary" onClick={() => handleGenerate(record)}>生成命令</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>新建模板</Button>
      </Space>

      <Table columns={columns} dataSource={templates} rowKey="id" loading={loading} />

      <Modal title="新建模板" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form form={form} onFinish={handleAdd} layout="vertical">
          <Form.Item name="name" label="模板名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea />
          </Form.Item>
          <Form.Item name={['config', 'core']} label="核心" initialValue="sing-box">
            <Select>
              <Select.Option value="sing-box">sing-box</Select.Option>
              <Select.Option value="xray-core">xray-core</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name={['config', 'port']} label="端口" initialValue={443}>
            <Input type="number" />
          </Form.Item>
          <Form.Item name={['config', 'server_name']} label="Reality目标域名" initialValue="download-installer.cdn.mozilla.net">
            <Input />
          </Form.Item>
          <Form.Item name={['config', 'agent_enabled']} label="启用Agent" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: 更新路由**

在 `frontend/src/App.tsx` 添加:

```tsx
import TemplateList from './pages/templates'

<Route path="/templates" element={<TemplateList />} />
```

- [ ] **Step 3: 添加安装命令生成弹窗**

```tsx
const [commandModalVisible, setCommandModalVisible] = useState(false)
const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
const [installCommand, setInstallCommand] = useState('')

const handleGenerate = (template: Template) => {
  // 模拟生成 serverId 和 psk（实际从API获取）
  const serverId = crypto.randomUUID()
  const psk = btoa(Math.random().toString()).slice(0, 32)
  setSelectedTemplate(template)
  setInstallCommand(generateInstallCommand(template, serverId, psk))
  setCommandModalVisible(true)
}
```

- [ ] **Step 4: 提交**

```bash
git add frontend/src/pages/templates/
git commit -m "feat: add template management frontend"
```

---

## Phase 3: 安装命令复制功能

### Task 6: 添加一键复制功能

**Files:**
- Modify: `frontend/src/pages/templates/index.tsx`

- [ ] **Step 1: 添加复制按钮**

在命令展示弹窗中添加复制功能:

```tsx
import { CopyOutlined } from '@ant-design/icons'

<Modal
  title="安装命令"
  open={commandModalVisible}
  onCancel={() => setCommandModalVisible(false)}
  footer={[
    <Button key="copy" icon={<CopyOutlined />} onClick={() => {
      navigator.clipboard.writeText(installCommand)
      message.success('已复制到剪贴板')
    }}>
      复制命令
    </Button>
  ]}
>
  <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4 }}>
    {installCommand}
  </pre>
</Modal>
```

- [ ] **Step 2: 提交**

```bash
git add frontend/src/pages/templates/
git commit -m "feat: add copy install command functionality"
```

---

## 实施检查清单

| Phase | Task | Description | Status |
|-------|------|-------------|--------|
| 1 | 1 | install.sh 参数解析框架 | ☐ |
| 1 | 2 | 模板加载逻辑 | ☐ |
| 1 | 3 | Agent 安装逻辑 | ☐ |
| 2 | 4 | 模板管理 API | ☐ |
| 2 | 5 | Web 模板管理页面 | ☐ |
| 3 | 6 | 复制安装命令功能 | ☐ |

---

**Plan saved to:** `docs/superpowers/plans/2026-05-16-install-sh-template-plan.md`

**执行选项：**

**1. Subagent-Driven (推荐)** - 每个任务分配一个 subagent 执行，完成后 review，快速迭代

**2. Inline Execution** - 在当前 session 中执行任务，带 checkpoint 审核

**选择哪个方式？**