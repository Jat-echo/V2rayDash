# 设计文档：客户端下载 Tab

**日期：** 2026-06-09  
**状态：** 已批准，待实现

---

## 1. 目标

在 V2rayDash 侧边栏新增「客户端」Tab，展示主流 Clash 系客户端的下载链接。版本号从 GitHub Releases API 动态获取，缓存 1 天后自动刷新，无需后端改动。

---

## 2. 客户端列表（静态配置）

| 客户端 | 维护状态 | 来源 | 平台 | GitHub Repo | Stars |
|--------|----------|------|------|-------------|-------|
| Clash Verge Rev | 🟢 活跃 | GitHub | Windows / macOS / Linux | `clash-verge-rev/clash-verge-rev` | 124k |
| FlClash | 🟢 活跃 | GitHub | Windows / macOS / Linux / Android | `chen08209/FlClash` | 42k |
| ClashMetaForAndroid | 🟢 活跃 | GitHub | Android | `MetaCubeX/ClashMetaForAndroid` | 41k |
| Mihomo | 🟢 活跃 | GitHub | Windows / macOS / Linux / Android | `MetaCubeX/mihomo` | 31k |
| Hiddify | 🟢 活跃 | GitHub | Windows / macOS / Linux / Android / iOS | `hiddify/hiddify-app` | 31k |
| ClashMi | 🟢 活跃 | GitHub | Windows / macOS / Linux / Android / iOS | `KaringX/clashmi` | 7k |
| Stash | 🟢 活跃 | App Store | iOS | — | — |
| Shadowrocket | 🟢 活跃 | App Store | iOS | — | — |

App Store 客户端不拉版本，直接链到 App Store 页面。已停更客户端仍通过 GitHub API 拉取最后发布版本并提供下载。

---

## 3. 架构与文件变更

### 新增文件

- `frontend/src/pages/clients/index.tsx` — 客户端下载页面组件

### 修改文件

- `frontend/src/App.tsx` — 添加路由 `/clients` 及侧边栏导航项「客户端」

### 不需要后端改动

---

## 4. 数据结构

```typescript
type Platform = 'windows' | 'macos' | 'linux' | 'android' | 'ios'

interface ClientConfig {
  name: string
  description: string
  platforms: Platform[]
  source: 'github' | 'appstore'
  repo?: string          // github: "owner/repo"
  storeUrl?: string      // appstore 直链
  // 各平台对应 release asset 的文件名匹配规则（正则字符串）
  assetPatterns?: Partial<Record<Platform, string>>
}

interface ReleaseInfo {
  version: string
  assets: Partial<Record<Platform, string>>  // platform -> download URL
}

// localStorage 缓存结构
interface ClientsCache {
  fetchedAt: number   // Unix timestamp (ms)
  data: Record<string, ReleaseInfo>  // repo -> release info
}
```

---

## 5. GitHub API 集成

**接口：** `GET https://api.github.com/repos/{owner}/{repo}/releases/latest`

**返回字段使用：**
- `tag_name` → 版本号
- `assets[].browser_download_url` → 下载地址，按 `assetPatterns` 匹配平台

**Asset 匹配规则（默认，各客户端可覆盖）：**
- Windows：`\.exe$` 或 `\.msi$`，排除 `arm`
- macOS：`\.dmg$` 或 `darwin`，x64 优先
- Linux：`\.AppImage$`，x86_64/amd64 优先
- Android：`\.apk$`，排除 `arm64` 只保留通用包（或 arm64 包）
- iOS：通过 `assets` 中含 `ipa` 或跳 App Store

如果一个平台有多个匹配文件（如 arm64、x64），优先选 x64 / universal，其余作为备选不展示（保持简洁）。

---

## 6. 缓存逻辑

```
localStorage key: "v2raydash_clients_cache"
TTL: 86400000 ms（1 天）

加载流程：
  1. 读取 localStorage 中的缓存
  2. 若缓存存在且 (now - fetchedAt) < 1天 → 直接使用缓存数据
  3. 否则 → 并发请求所有 GitHub 客户端的 releases/latest
  4. 请求完成后更新缓存（含 fetchedAt = now）
  5. 任一请求失败 → 该客户端显示"获取失败"状态，其余正常展示

手动刷新：
  右上角「刷新」按钮 → 清除缓存 → 重新拉取
```

---

## 7. UI 设计

### 7.1 客户端下载表格

**列结构：** 客户端（含简介）/ 版本 / 🪟 Windows / 🍎 macOS / 🐧 Linux / 🤖 Android / 📱 iOS

每个平台独立成列，对齐展示。有下载链接的单元格显示「下载」按钮（直接触发文件下载），不支持的平台显示「—」，App Store 客户端显示「前往 ↗」（新标签打开）。

**状态展示：**
- 加载中：版本列显示 spin 动画
- 成功：显示版本号（如 `v2.2.3`，monospace 字体）
- App Store 客户端：版本列显示「App Store」灰色文字
- 获取失败：版本列显示「获取失败」，下载列显示「前往 GitHub ↗」

**右上角：**
- 缓存时间：「版本数据缓存至 YYYY-MM-DD HH:mm」
- 「刷新」按钮（清除缓存，重新拉取）

### 7.2 Linux 服务器安装区块

位于表格下方，独立卡片。

**交互流程：**
1. 下拉选择器列出所有订阅（调用 `subscriptionAPI.list()`）
2. 用户选中后调用 `subscriptionAPI.getLink(id)` 获取订阅 URL
3. 将 URL 填入命令中的 `--sub` 参数位置，实时更新代码块
4. `--secret` 保留占位符「你的面板密码」，由用户自行填写
5. 「复制」按钮复制完整命令到剪贴板

**展示的安装命令（默认含 gh-proxy）：**
```bash
curl -fsSL https://gh-proxy.com/https://raw.githubusercontent.com/Jat-echo/InstallMihomo/main/install-mihomo.sh \
  | sudo bash -s -- \
      --sub '${订阅URL}' \
      --secret '你的面板密码'
```

附注说明：服务器可直连 GitHub 时，去掉 gh-proxy 前缀并加 `--no-github-proxy`。

**样式：** 跟随现有 Ant Design + CSS 变量主题，代码块使用深色背景，不引入新依赖

---

## 8. 错误处理

| 场景 | 处理方式 |
|------|----------|
| GitHub API 限速（403/429） | 显示"获取失败"，链到 GitHub releases 页 |
| 网络超时 | 同上，但提示"网络错误" |
| Asset 无匹配 | 该平台按钮不显示 |
| 缓存读取异常 | 忽略缓存，重新拉取 |

---

## 9. 不在本期范围内

- 管理员在后台动态配置客户端列表
- 显示 Changelog 或 Release Notes
- 多架构下载选择（如 arm64 vs x64 分别展示）
- Linux 服务器安装命令中 `--secret` 自动填入（密码属于运维配置，不从系统读取）
