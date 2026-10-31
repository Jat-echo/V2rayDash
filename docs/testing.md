# 测试说明

## 前端

需要 Node.js 24+。在 `frontend/` 执行：

```sh
npm ci
npm test
npm run build
```

- `npm test` 使用 Node 内置测试运行器验证实际 API 客户端的响应解包和排序请求结构。
- `npm run typecheck` 检查所有页面和 Vite 配置。
- `npm run build` 包含类型检查，成功后生成生产静态资源。

## Go 单元和路由测试

```sh
cd backend
go test ./... -v
```

默认测试不需要 PostgreSQL，会调用生产 `SetupRoutes`，检查健康接口、未登录访问控制和非法请求；同时验证时间聚合和国旗回退逻辑。数据库流程没有配置时会显示 `SKIP`。

Agent 是独立模块，需另外执行：

```sh
cd agent
go test ./...
go build ./cmd/agent
```

Agent 的指标采集依赖 Linux，Windows 上构建通过不代表已验证真实 Linux 节点采集。

## 真实 PostgreSQL 集成测试

准备一个专用 PostgreSQL 测试数据库，连接用户需要建表和创建 schema 的权限。不要使用生产连接串。

PowerShell：

```powershell
$env:TEST_DATABASE_URL = 'postgres://postgres:postgres@127.0.0.1:5432/v2ray_dash_test?sslmode=disable'
cd backend
go test ./... -count=1 -v
```

Linux/macOS：

```sh
cd backend
TEST_DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/v2ray_dash_test?sslmode=disable' go test ./... -count=1 -v
```

测试创建随机命名的独立 schema，结束时只删除该 schema；不重置已有应用表。流程调用真实处理器和数据库，覆盖：

- 新数据库初始化和重复迁移。
- 管理员初始化、错误密码拒绝和退出后的令牌失效。
- 服务器新增、列表、编辑和删除。
- 订阅创建、关联账号、生成链接和公开订阅读取。
- Agent 心跳、账号流量累计、订阅流量汇总和趋势查询。

测试直接准备账号，避免触发 SSH 同步；不会安装或重启远程 Xray。GitHub Actions 为此流程提供 PostgreSQL 17 服务。

## 浏览器与部署验收

前后端启动后访问 <http://localhost:3000>，检查首次初始化、登录、服务器与订阅页面、账号拖拽排序和移动端导航。访问 API 的管理接口需要 `Authorization: Bearer <token>`；健康检查 `GET /health` 无需登录。

远程 SSH 安装、Xray 同步、Linux 节点指标、真实代理连通性和线上部署版本需要单独验收，本地测试不会替代这些验证。

## 代理链路回归

`TEST_DATABASE_URL` 启用后还会验证：中转 / 落地服务器角色、手动创建代理链路时自动生成专用上游账号且不创建客户账号、拒绝重复路径和中转链、直连与多条链路混合订阅、同一链路的用户复用上游连接并保留独立入口凭据、批量关联与重复跳过、批量解除时保留其他节点和共享账号、单项事务回滚与部分失败、原有未接管直连节点立即导出兼容、上游账号隐藏及禁止手工修改 / 导出、同步收尾事务回滚、节点 PSK 校验和累计计数器去重。旧批量创建 API 保留兼容测试；界面采用已有订阅列表。

可额外设置 `XRAY_TEST_BINARY` 为官方 Xray 可执行文件的绝对路径，运行后端测试时会让真实核心检查生成的中转配置。未提供该变量时仅跳过这项核心校验。

配置文件替换和回滚使用隔离临时目录，模拟服务命令与 HTTPS 探测，不操作线上服务：

```sh
python3 -m unittest discover -s backend/internal/service -p node_remote_test.py -v
```

Agent 的队列测试覆盖发送失败、进程重新加载及持久化批次重放；协议兼容检查确保旧配置仍采用原有的增量计数，新接管监控使用累计计数。下载路由验证 `AGENT_BINARY_PATH` 生效并拒绝不支持的架构。前端目标判断测试防止将上游账号或其他链路账号作为直连用户账号。

浏览器验收检查服务器角色、手动创建链路、混合节点选择、列表勾选关联、逐项结果、同步状态图标与聚焦提示，以及桌面、平板和手机布局。具体操作和未实现的功能见 [代理链路管理说明](relay-management.md)。
