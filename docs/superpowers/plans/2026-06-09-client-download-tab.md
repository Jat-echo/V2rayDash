# 客户端下载 Tab 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 V2rayDash 侧边栏新增「客户端」Tab，展示 8 个主流 Clash 系客户端的下载链接（版本动态拉取 GitHub Releases，缓存 1 天），以及 Linux 服务器一键安装命令（订阅地址预填）。

**Architecture:** 纯前端实现，新增一个页面组件 `frontend/src/pages/clients/index.tsx`，包含类型定义、静态配置、GitHub API 缓存逻辑、下载表格和安装命令区块。在 `App.tsx` 中注册路由和侧边栏导航项。无需修改后端。

**Tech Stack:** React 18, TypeScript, Ant Design 5, react-router-dom 7, Vite

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `frontend/src/pages/clients/index.tsx` | 新建 | 完整页面：类型、配置、缓存、表格、安装区块 |
| `frontend/src/App.tsx` | 修改 | 添加侧边栏导航项 + `/clients` 路由 |

---

## Task 1: 创建页面文件 — 类型定义与静态客户端配置

**Files:**
- Create: `frontend/src/pages/clients/index.tsx`

- [ ] **Step 1: 创建文件，写入类型定义和静态客户端列表**

```tsx
// frontend/src/pages/clients/index.tsx
import { useState, useEffect, useCallback } from 'react'
import { Select, Spin, Button, message } from 'antd'
import { ReloadOutlined, CopyOutlined } from '@ant-design/icons'
import { subscriptionAPI, Subscription } from '../../services/api'

// ── 类型定义 ──────────────────────────────────────────────────────────────────

interface MacOSAssets {
  intel?: string
  apple?: string
}

interface ResolvedAssets {
  windows?: string
  macos?: MacOSAssets
  linux?: string
  android?: string
  ios?: string
}

interface ReleaseInfo {
  version: string
  assets: ResolvedAssets
  releasesUrl: string
  error?: boolean
}

interface ClientsCache {
  fetchedAt: number
  data: Record<string, ReleaseInfo>
}

interface AssetPatterns {
  windows?: RegExp
  macosIntel?: RegExp
  macosApple?: RegExp
  macosUniversal?: RegExp
  linux?: RegExp
  android?: RegExp
  ios?: RegExp
}

interface ClientConfig {
  name: string
  description: string
  source: 'github' | 'appstore'
  repo?: string
  storeUrl?: string
  iosStoreUrl?: string
  platforms: Array<'windows' | 'macos' | 'linux' | 'android' | 'ios'>
  patterns?: AssetPatterns
}

// ── 静态客户端列表（按 star 数排序）────────────────────────────────────────────

const CLIENT_LIST: ClientConfig[] = [
  {
    name: 'Clash Verge Rev',
    description: '基于 Tauri 的跨平台代理客户端',
    source: 'github',
    repo: 'clash-verge-rev/clash-verge-rev',
    platforms: ['windows', 'macos', 'linux'],
    patterns: {
      windows: /clash[\._]verge.*x64.*(\.exe|\.msi)$/i,
      macosIntel: /clash[\._]verge.*x64\.dmg$/i,
      macosApple: /clash[\._]verge.*aarch64\.dmg$/i,
      linux: /clash[\._]verge.*amd64\.AppImage$/i,
    },
  },
  {
    name: 'FlClash',
    description: '基于 Flutter 的现代化跨平台客户端',
    source: 'github',
    repo: 'chen08209/FlClash',
    platforms: ['windows', 'macos', 'linux', 'android'],
    patterns: {
      windows: /FlClash-.*windows-amd64.*\.exe$/i,
      macosIntel: /FlClash-.*macos-amd64\.dmg$/i,
      macosApple: /FlClash-.*macos-arm64\.dmg$/i,
      linux: /FlClash-.*linux-amd64\.AppImage$/i,
      android: /FlClash-.*android.*\.apk$/i,
    },
  },
  {
    name: 'ClashMetaForAndroid',
    description: 'Android 专属 Clash Meta 客户端',
    source: 'github',
    repo: 'MetaCubeX/ClashMetaForAndroid',
    platforms: ['android'],
    patterns: {
      android: /cmfa-.*universal.*\.apk$/i,
    },
  },
  {
    name: 'Mihomo',
    description: 'Clash Meta 核心代理引擎（命令行）',
    source: 'github',
    repo: 'MetaCubeX/mihomo',
    platforms: ['windows', 'macos', 'linux', 'android'],
    patterns: {
      windows: /mihomo-windows-amd64-v.*\.zip$/i,
      macosIntel: /mihomo-darwin-amd64-v.*\.gz$/i,
      macosApple: /mihomo-darwin-arm64-v.*\.gz$/i,
      linux: /mihomo-linux-amd64-v.*\.gz$/i,
      android: /mihomo-android-arm64-v.*\.gz$/i,
    },
  },
  {
    name: 'Hiddify',
    description: '全平台开源代理客户端',
    source: 'github',
    repo: 'hiddify/hiddify-app',
    platforms: ['windows', 'macos', 'linux', 'android', 'ios'],
    iosStoreUrl: 'https://apps.apple.com/app/hiddify-proxy-vpn/id6596777532',
    patterns: {
      windows: /Hiddify-Windows-Setup-x64\.exe$/i,
      macosIntel: /Hiddify-MacOS-x64\.dmg$/i,
      macosApple: /Hiddify-MacOS-arm64\.dmg$/i,
      linux: /Hiddify-Linux-x64\.AppImage$/i,
      android: /Hiddify-Android-universal\.apk$/i,
    },
  },
  {
    name: 'ClashMi',
    description: '全平台 Clash Meta 客户端',
    source: 'github',
    repo: 'KaringX/clashmi',
    platforms: ['windows', 'macos', 'linux', 'android', 'ios'],
    patterns: {
      windows: /\.exe$/i,
      macosIntel: /(x64|amd64|intel).*\.dmg$|(\.dmg).*(x64|amd64|intel)/i,
      macosApple: /(arm64|aarch64).*\.dmg$|(\.dmg).*(arm64|aarch64)/i,
      linux: /\.AppImage$/i,
      android: /\.apk$/i,
      ios: /\.ipa$/i,
    },
  },
  {
    name: 'Stash',
    description: 'iOS 高级代理客户端（付费）',
    source: 'appstore',
    storeUrl: 'https://apps.apple.com/app/stash-rule-based-proxy/id1596063349',
    platforms: ['ios'],
  },
  {
    name: 'Shadowrocket',
    description: 'iOS 轻量代理工具（付费）',
    source: 'appstore',
    storeUrl: 'https://apps.apple.com/app/shadowrocket/id932747118',
    platforms: ['ios'],
  },
]

export default function ClientDownload() {
  return <div>placeholder</div>
}
```

- [ ] **Step 2: 检查 TypeScript 类型无误**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npx tsc --noEmit
```

期望：无报错（或仅有 placeholder 相关的 unused import 警告，可忽略）

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/clients/index.tsx
git commit -m "feat(clients): scaffold page with types and client config"
```

---

## Task 2: 实现 Asset 匹配与 GitHub API 拉取

**Files:**
- Modify: `frontend/src/pages/clients/index.tsx`

- [ ] **Step 1: 在 `export default` 前插入 parseAssets 和 fetchRelease 函数**

将以下代码插入到 `CLIENT_LIST` 常量定义之后、`export default function ClientDownload` 之前：

```tsx
// ── Asset 匹配 ────────────────────────────────────────────────────────────────

function parseAssets(
  rawAssets: Array<{ name: string; browser_download_url: string }>,
  patterns: AssetPatterns,
): ResolvedAssets {
  const find = (re: RegExp) =>
    rawAssets.find(a => re.test(a.name))?.browser_download_url

  const result: ResolvedAssets = {}

  if (patterns.windows) {
    const url = find(patterns.windows)
    if (url) result.windows = url
  }

  // macOS: universal 作为 Intel 和 M芯片的 fallback
  const universalUrl = patterns.macosUniversal ? find(patterns.macosUniversal) : undefined
  const intelUrl = patterns.macosIntel ? find(patterns.macosIntel) : undefined
  const appleUrl = patterns.macosApple ? find(patterns.macosApple) : undefined
  const resolvedIntel = intelUrl ?? universalUrl
  const resolvedApple = appleUrl ?? universalUrl
  if (resolvedIntel || resolvedApple) {
    result.macos = { intel: resolvedIntel, apple: resolvedApple }
  }

  if (patterns.linux) {
    const url = find(patterns.linux)
    if (url) result.linux = url
  }

  if (patterns.android) {
    const url = find(patterns.android)
    if (url) result.android = url
  }

  if (patterns.ios) {
    const url = find(patterns.ios)
    if (url) result.ios = url
  }

  return result
}

// ── GitHub API ────────────────────────────────────────────────────────────────

async function fetchRelease(client: ClientConfig): Promise<ReleaseInfo> {
  const releasesUrl = `https://github.com/${client.repo}/releases`
  try {
    const res = await fetch(
      `https://api.github.com/repos/${client.repo}/releases/latest`,
      { headers: { Accept: 'application/vnd.github+json' } },
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const json = await res.json()
    const resolved = client.patterns
      ? parseAssets(json.assets ?? [], client.patterns)
      : {}
    // GitHub 客户端且有 iosStoreUrl：iOS 下载链接用 App Store
    if (client.iosStoreUrl) resolved.ios = client.iosStoreUrl
    return { version: json.tag_name ?? '', assets: resolved, releasesUrl }
  } catch {
    return { version: '', assets: {}, releasesUrl, error: true }
  }
}
```

- [ ] **Step 2: 检查 TypeScript 类型无误**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npx tsc --noEmit
```

期望：无报错

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/clients/index.tsx
git commit -m "feat(clients): add asset parsing and GitHub API fetch"
```

---

## Task 3: 实现缓存逻辑，替换占位组件为完整 UI（表格）

**Files:**
- Modify: `frontend/src/pages/clients/index.tsx`

- [ ] **Step 1: 在 `fetchRelease` 之后、`export default` 之前插入缓存常量和工具函数**

（`loadReleases` 调用 `fetchRelease`，必须在其定义之后）

```tsx
// ── 缓存 ──────────────────────────────────────────────────────────────────────

const CACHE_KEY = 'v2raydash_clients_cache'
const CACHE_TTL = 86400000 // 1 天 ms

function loadCache(): ClientsCache | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as ClientsCache
  } catch {
    return null
  }
}

function saveCache(data: Record<string, ReleaseInfo>): void {
  const cache: ClientsCache = { fetchedAt: Date.now(), data }
  localStorage.setItem(CACHE_KEY, JSON.stringify(cache))
}

function clearCache(): void {
  localStorage.removeItem(CACHE_KEY)
}

async function loadReleases(force = false): Promise<{ data: Record<string, ReleaseInfo>; fetchedAt: number }> {
  if (!force) {
    const cached = loadCache()
    if (cached && Date.now() - cached.fetchedAt < CACHE_TTL) {
      return { data: cached.data, fetchedAt: cached.fetchedAt }
    }
  }
  const githubClients = CLIENT_LIST.filter(c => c.source === 'github')
  const results = await Promise.all(githubClients.map(c => fetchRelease(c)))
  const data: Record<string, ReleaseInfo> = {}
  githubClients.forEach((c, i) => { data[c.repo!] = results[i] })
  saveCache(data)
  return { data, fetchedAt: Date.now() }
}
```

- [ ] **Step 2: 替换 `export default function ClientDownload` 为完整实现（表格部分）**

用以下代码替换 `export default function ClientDownload() { return <div>placeholder</div> }`：

```tsx
// ── 样式常量 ──────────────────────────────────────────────────────────────────

const btnStyle: React.CSSProperties = {
  background: 'rgba(255,255,255,0.06)',
  border: '1px solid rgba(255,255,255,0.12)',
  color: 'rgba(255,255,255,0.82)',
  padding: '2px 12px',
  borderRadius: 4,
  textDecoration: 'none',
  fontSize: 12,
  display: 'inline-block',
  whiteSpace: 'nowrap',
}

const dashStyle: React.CSSProperties = { color: 'rgba(255,255,255,0.18)', fontSize: 13 }

const thStyle: React.CSSProperties = {
  textAlign: 'center',
  padding: '10px 10px',
  color: 'rgba(255,255,255,0.45)',
  fontWeight: 500,
  fontSize: 13,
  width: 90,
}

// ── 子组件 ────────────────────────────────────────────────────────────────────

function DownloadBtn({ url, label = '下载' }: { url: string; label?: string }) {
  return (
    <a href={url} target="_blank" rel="noreferrer" style={btnStyle}>
      {label}
    </a>
  )
}

function MacOSCell({ macos }: { macos?: MacOSAssets }) {
  if (!macos?.intel && !macos?.apple) return <span style={dashStyle}>—</span>
  // 同一 URL 说明是 universal 包，只显示一个按钮
  if (macos.intel && macos.apple && macos.intel === macos.apple) {
    return <DownloadBtn url={macos.intel} />
  }
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, alignItems: 'center' }}>
      {macos.intel && <DownloadBtn url={macos.intel} label="Intel" />}
      {macos.apple && <DownloadBtn url={macos.apple} label="M芯片" />}
    </div>
  )
}

function PlatformCell({ url, storeLabel }: { url?: string; storeLabel?: string }) {
  if (!url) return <span style={dashStyle}>—</span>
  return <DownloadBtn url={url} label={storeLabel ?? '下载'} />
}

// ── 主组件 ────────────────────────────────────────────────────────────────────

export default function ClientDownload() {
  const [releases, setReleases] = useState<Record<string, ReleaseInfo>>({})
  const [fetchedAt, setFetchedAt] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [selectedSubId, setSelectedSubId] = useState<string | undefined>()
  const [subLink, setSubLink] = useState<string>('')
  const [copying, setCopying] = useState(false)

  const load = useCallback(async (force = false) => {
    setLoading(true)
    try {
      const result = await loadReleases(force)
      setReleases(result.data)
      setFetchedAt(result.fetchedAt)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    subscriptionAPI.list().then(list => setSubscriptions(list ?? []))
  }, [load])

  const handleRefresh = () => {
    clearCache()
    load(true)
  }

  const handleSubChange = async (id: string) => {
    setSelectedSubId(id)
    try {
      const { link } = await subscriptionAPI.getLink(id)
      setSubLink(link)
    } catch {
      setSubLink('')
    }
  }

  const installCmd = `curl -fsSL https://gh-proxy.com/https://raw.githubusercontent.com/Jat-echo/InstallMihomo/main/install-mihomo.sh \\
  | sudo bash -s -- \\
      --sub '${subLink || '你的订阅地址'}' \\
      --secret '你的面板密码'`

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(installCmd)
      setCopying(true)
      message.success('命令已复制')
      setTimeout(() => setCopying(false), 2000)
    } catch {
      message.error('复制失败，请手动选中')
    }
  }

  const cacheTimeStr = fetchedAt
    ? new Date(fetchedAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
    : ''

  return (
    <div style={{ maxWidth: 1000 }}>
      {/* ── 表格区块 ── */}
      <div style={{
        background: 'var(--bg-card, rgba(255,255,255,0.03))',
        border: '1px solid rgba(255,255,255,0.08)',
        borderRadius: 8,
        overflow: 'hidden',
        marginBottom: 24,
      }}>
        {/* 表头行 */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '12px 16px',
          borderBottom: '1px solid rgba(255,255,255,0.08)',
        }}>
          <span style={{ fontWeight: 600, fontSize: 14 }}>客户端下载</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            {fetchedAt && (
              <span style={{ fontSize: 12, color: 'rgba(255,255,255,0.35)' }}>
                版本缓存至 {cacheTimeStr}
              </span>
            )}
            <Button
              size="small"
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={handleRefresh}
              style={{ fontSize: 12 }}
            >
              刷新
            </Button>
          </div>
        </div>

        {/* 表格 */}
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 680 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.08)' }}>
                <th style={{ textAlign: 'left', padding: '10px 16px', color: 'rgba(255,255,255,0.45)', fontWeight: 500, fontSize: 13 }}>
                  客户端
                </th>
                <th style={{ ...thStyle, width: 80 }}>版本</th>
                <th style={thStyle}>🪟 Windows</th>
                <th style={thStyle}>🍎 macOS</th>
                <th style={thStyle}>🐧 Linux</th>
                <th style={thStyle}>🤖 Android</th>
                <th style={thStyle}>📱 iOS</th>
              </tr>
            </thead>
            <tbody>
              {CLIENT_LIST.map((client, idx) => {
                const info = client.repo ? releases[client.repo] : undefined
                const isLast = idx === CLIENT_LIST.length - 1
                return (
                  <tr key={client.name} style={{ borderBottom: isLast ? 'none' : '1px solid rgba(255,255,255,0.05)' }}>
                    {/* 名称 + 简介 */}
                    <td style={{ padding: '12px 16px' }}>
                      <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 2 }}>{client.name}</div>
                      <div style={{ color: 'rgba(255,255,255,0.35)', fontSize: 11 }}>{client.description}</div>
                    </td>

                    {/* 版本 */}
                    <td style={{ textAlign: 'center', padding: '12px 10px' }}>
                      {client.source === 'appstore' ? (
                        <span style={{ color: 'rgba(255,255,255,0.3)', fontSize: 11 }}>App Store</span>
                      ) : loading && !info ? (
                        <Spin size="small" />
                      ) : info?.error ? (
                        <span style={{ color: 'rgba(255,255,255,0.3)', fontSize: 11 }}>获取失败</span>
                      ) : (
                        <span style={{ color: 'var(--morandi-dusty, #c9a9a6)', fontFamily: 'monospace', fontSize: 12 }}>
                          {info?.version ?? '—'}
                        </span>
                      )}
                    </td>

                    {/* Windows */}
                    <td style={{ textAlign: 'center', padding: '12px 10px' }}>
                      {info?.error
                        ? <DownloadBtn url={info.releasesUrl} label="GitHub ↗" />
                        : <PlatformCell url={info?.assets.windows} />
                      }
                    </td>

                    {/* macOS */}
                    <td style={{ textAlign: 'center', padding: '12px 10px' }}>
                      {info?.error
                        ? <DownloadBtn url={info.releasesUrl} label="GitHub ↗" />
                        : <MacOSCell macos={info?.assets.macos} />
                      }
                    </td>

                    {/* Linux */}
                    <td style={{ textAlign: 'center', padding: '12px 10px' }}>
                      {info?.error
                        ? <DownloadBtn url={info.releasesUrl} label="GitHub ↗" />
                        : <PlatformCell url={info?.assets.linux} />
                      }
                    </td>

                    {/* Android */}
                    <td style={{ textAlign: 'center', padding: '12px 10px' }}>
                      {info?.error
                        ? <DownloadBtn url={info.releasesUrl} label="GitHub ↗" />
                        : <PlatformCell url={info?.assets.android} />
                      }
                    </td>

                    {/* iOS */}
                    <td style={{ textAlign: 'center', padding: '12px 10px' }}>
                      {client.source === 'appstore' ? (
                        <DownloadBtn url={client.storeUrl!} label="前往 ↗" />
                      ) : info?.error ? (
                        <DownloadBtn url={info.releasesUrl} label="GitHub ↗" />
                      ) : (
                        <PlatformCell
                          url={info?.assets.ios ?? client.iosStoreUrl}
                          storeLabel={client.iosStoreUrl ? '前往 ↗' : '下载'}
                        />
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* ── Linux 服务器安装区块 ── */}
      <div style={{
        background: 'var(--bg-card, rgba(255,255,255,0.03))',
        border: '1px solid rgba(255,255,255,0.08)',
        borderRadius: 8,
        overflow: 'hidden',
      }}>
        <div style={{ padding: '12px 16px', borderBottom: '1px solid rgba(255,255,255,0.08)', fontWeight: 600, fontSize: 14 }}>
          🖥️ Linux 服务器 — 一键安装 Mihomo
        </div>
        <div style={{ padding: 16 }}>
          <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', marginBottom: 14 }}>
            在服务器上执行以下命令，自动安装 Mihomo 代理服务并配置每日订阅更新
          </div>

          {/* 订阅选择 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
            <span style={{ fontSize: 12, color: 'rgba(255,255,255,0.55)', whiteSpace: 'nowrap' }}>选择订阅</span>
            <Select
              size="small"
              style={{ minWidth: 200 }}
              placeholder="选择订阅以预填地址"
              value={selectedSubId}
              onChange={handleSubChange}
              options={subscriptions.map(s => ({ value: s.id, label: s.name }))}
            />
            <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.25)' }}>订阅 URL 仅在本地使用，不上传</span>
          </div>

          {/* 命令块 */}
          <div style={{ position: 'relative' }}>
            <pre style={{
              background: '#0c0c0c',
              border: '1px solid rgba(255,255,255,0.07)',
              borderRadius: 6,
              padding: '14px 16px',
              fontFamily: "'Courier New', monospace",
              fontSize: 12,
              color: '#c9d1d9',
              lineHeight: 1.8,
              margin: 0,
              overflowX: 'auto',
              whiteSpace: 'pre',
            }}>
              {installCmd}
            </pre>
            <Button
              size="small"
              icon={<CopyOutlined />}
              onClick={handleCopy}
              loading={copying}
              style={{ position: 'absolute', top: 8, right: 8, fontSize: 11 }}
            >
              复制
            </Button>
          </div>

          {/* 直连提示 */}
          <div style={{ marginTop: 10, fontSize: 11, color: 'rgba(255,255,255,0.3)' }}>
            服务器可直连 GitHub？去掉 gh-proxy 前缀并加{' '}
            <code style={{ background: 'rgba(255,255,255,0.05)', padding: '1px 5px', borderRadius: 3, color: 'rgba(255,255,255,0.5)' }}>
              --no-github-proxy
            </code>
          </div>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 3: 检查 TypeScript 类型无误**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npx tsc --noEmit
```

期望：无报错

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/clients/index.tsx
git commit -m "feat(clients): implement full page with table and Linux install section"
```

---

## Task 4: 在 App.tsx 中注册导航项和路由

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: 在 App.tsx 中添加 import**

在现有 import 区域（`import ServerList` 等行的附近）插入：

```tsx
import ClientDownload from './pages/clients'
```

- [ ] **Step 2: 添加侧边栏图标组件**

在 `SettingsIcon` 组件定义之后，插入：

```tsx
const ClientIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="2" y="3" width="20" height="14" rx="2"/>
    <path d="M8 21h8M12 17v4"/>
    <path d="M9 10l2 2 4-4"/>
  </svg>
)
```

- [ ] **Step 3: 在 navItems 数组中添加「客户端」导航项**

找到 `navItems` 数组，在 `{ path: '/logs', ... }` 和 `{ path: '/settings', ... }` 之间插入：

```tsx
{ path: '/clients', label: '客户端', icon: <ClientIcon /> },
```

完整的 `navItems` 应为：

```tsx
const navItems = [
  { path: '/servers', label: '服务器', icon: <ServerIcon /> },
  { path: '/subscriptions', label: '订阅', icon: <SubIcon /> },
  { path: '/logs', label: '日志', icon: <LogIcon /> },
  { path: '/clients', label: '客户端', icon: <ClientIcon /> },
  { path: '/settings', label: '设置', icon: <SettingsIcon /> },
]
```

- [ ] **Step 4: 在 Routes 中添加路由**

在 `<Route path="/settings" element={<Settings />} />` 之前插入：

```tsx
<Route path="/clients" element={<ClientDownload />} />
```

- [ ] **Step 5: 检查 TypeScript 类型无误**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npx tsc --noEmit
```

期望：无报错

- [ ] **Step 6: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(clients): add nav item and route for client download tab"
```

---

## Task 5: 构建前端，验证页面正常运行

**Files:**
- 无新增/修改

- [ ] **Step 1: 启动开发服务器，访问页面**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npm run dev
```

访问 `http://localhost:5173`（或控制台输出的端口），登录后点击「客户端」Tab，检查：
- 侧边栏出现「客户端」导航项
- 表格正确渲染 8 个客户端（6 个 GitHub + 2 个 App Store）
- 版本列显示 Spin 动画（加载中），随后显示版本号
- macOS 列对有 arm64/x64 区分的客户端显示两个按钮（Intel / M芯片）
- App Store 客户端（Stash、Shadowrocket）版本列显示「App Store」，iOS 列显示「前往 ↗」
- Linux 服务器区块可选择订阅，命令随之更新，复制按钮可用

- [ ] **Step 2: 验证缓存行为**

刷新页面后，版本数据不触发新的 GitHub API 请求（通过浏览器 DevTools Network 面板验证），缓存时间显示正确。

点击「刷新」按钮，应触发新的 GitHub API 请求并更新缓存时间。

- [ ] **Step 3: 构建生产包，确认无 TypeScript 错误**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npm run build
```

期望：构建成功，无报错

- [ ] **Step 4: Commit 构建产物（如项目惯例提交 dist）**

```bash
cd /home/jat-id/Project/V2rayDash && git add frontend/dist
git commit -m "build: rebuild frontend with client download tab"
```
