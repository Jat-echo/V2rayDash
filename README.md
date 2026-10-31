# V2rayDash

私有化部署的 V2ray/Xray 管理面板，包含服务器管理、SSH 安装、账号与订阅管理、节点和流量监控、客户端下载。

## 开发环境

- Node.js 24+、npm
- Go 1.23+（本次使用 Go 1.27.3 验证）
- PostgreSQL 15+（本次使用 PostgreSQL 17 验证）

前端使用 React、TypeScript、Ant Design 和 Vite；后端使用 Go、Gin 和 PostgreSQL。`agent/` 是实际部署到 Linux 节点的独立 Go 模块；根目录 `cmd/agent/` 是历史原型，不作为部署入口。

## 本地启动

如果已安装 Docker，可以先启动开发数据库：

```sh
docker compose up -d db
```

在 PowerShell 中，从项目根目录启动后端：

```powershell
$env:DATABASE_URL = 'postgres://postgres:postgres@127.0.0.1:5432/v2ray_dash?sslmode=disable'
cd backend
go run ./cmd/server
```

在另一个终端启动前端：

```sh
cd frontend
npm ci
npm run dev
```

打开 <http://localhost:3000>，首次访问时创建管理员。前端将 `/api` 请求代理到本机 `8080` 端口；后端启动时自动建表和执行迁移。Linux/macOS 通过 `export DATABASE_URL='…'` 设置相同环境变量。

也可以使用已有的本地 PostgreSQL，但需要先创建 `v2ray_dash` 数据库并相应修改连接串。数据库密码仅用于本地开发，线上配置应由部署环境提供。

本机已准备好的便携 Go 位于 `.tools/go/bin/`，可在当前 PowerShell 会话中临时加入 PATH：

```powershell
$env:Path = "$(Resolve-Path .tools/go/bin);$env:Path"
```

`.tools/` 不提交到 Git；其他机器从 [Go 官网](https://go.dev/dl/) 安装工具链即可。节点 Agent 的系统指标采集面向 Linux；控制中心在 Windows 上可运行，CPU/内存监控仍依赖 Linux `/proc`。

本机也已在 `.tools/pgsql/` 准备便携 PostgreSQL，开发库为 `v2ray_dash_dev`。不使用 Docker 时，可从项目根目录启动它：

```powershell
.\.tools\pgsql\bin\pg_ctl.exe -D .tools/pgdata -l .tools/postgres.log -o '-h 127.0.0.1 -p 55432' -w start
$env:DATABASE_URL = 'postgres://postgres@127.0.0.1:55432/v2ray_dash_dev?sslmode=disable'
```

随后按上面的命令运行后端。用完后执行 `.\.tools\pgsql\bin\pg_ctl.exe -D .tools/pgdata -m fast -w stop`。此便携数据库仅监听本机；这些 `.tools/` 文件不会随 Git 克隆分发。

## 验证与构建

```sh
cd frontend
npm ci
npm test
npm run build
```

`npm run build` 会先执行严格 TypeScript 检查，再生成 `frontend/dist/`。依赖按 `package-lock.json` 重装，不能复制其他操作系统的 `node_modules`。

```sh
cd backend
go test ./...
go build -o v2ray-dash-server ./cmd/server
```

Windows 下可给输出文件加 `.exe`。Agent 单独构建：

```sh
cd agent
go test ./...
go build -o v2ray-agent ./cmd/agent
```

真实数据库集成测试需要显式设置 `TEST_DATABASE_URL`，详见 [测试说明](docs/testing.md)。CI 会提供 PostgreSQL 并执行该流程；普通 `go test ./...` 会明确跳过未配置数据库的流程。

## 仓库约定

- 跟踪源码、依赖锁文件和文档；不跟踪 `node_modules`、`dist`、编译二进制、本地工具和凭据。
- `.gitattributes` 统一文本使用 LF，确保 Linux shell 脚本不会因 Windows 换行失败。
- 不将 `.env`、Agent 配置、SSH 密钥或数据库密码提交到仓库。
- 本地构建和测试通过不代表线上版本、远程安装或真实代理流量已经验证；部署单独执行。
