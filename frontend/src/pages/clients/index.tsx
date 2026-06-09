// frontend/src/pages/clients/index.tsx
import React, { useState, useEffect, useCallback } from 'react'
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
      windows: /clash[._-]verge.*x64.*(\.exe|\.msi)$/i,
      macosIntel: /clash[._-]verge.*x64\.dmg$/i,
      macosApple: /clash[._-]verge.*aarch64\.dmg$/i,
      linux: /clash[._-]verge.*amd64\.AppImage$/i,
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
  try { saveCache(data) } catch { /* quota exceeded — skip caching */ }
  return { data, fetchedAt: Date.now() }
}

// ── 平台配色系统（Morandi 调色板） ────────────────────────────────────────────

const PLATFORM_THEME = {
  windows: { bg: 'rgba(157,180,192,0.12)', text: '#4E7A8A', border: 'rgba(157,180,192,0.4)' },
  macos:   { bg: 'rgba(180,167,199,0.12)', text: '#6A5F85', border: 'rgba(180,167,199,0.4)' },
  linux:   { bg: 'rgba(168,181,160,0.12)', text: '#526B4C', border: 'rgba(168,181,160,0.4)' },
  android: { bg: 'rgba(196,131,106,0.12)', text: '#8B5640', border: 'rgba(196,131,106,0.4)' },
  ios:     { bg: 'rgba(201,169,166,0.12)', text: '#8B5552', border: 'rgba(201,169,166,0.4)' },
  github:  { bg: 'rgba(158,154,147,0.1)',  text: '#6B6760', border: 'rgba(158,154,147,0.35)' },
}

const dashStyle: React.CSSProperties = {
  color: 'var(--border-color)',
  fontSize: 16,
  lineHeight: 1,
}

const thStyle: React.CSSProperties = {
  textAlign: 'center',
  padding: '11px 8px',
  color: 'var(--text-secondary)',
  fontWeight: 500,
  fontSize: 12,
  width: 88,
  background: 'var(--bg-secondary)',
  borderBottom: '1px solid var(--border-color)',
}

// ── 子组件 ────────────────────────────────────────────────────────────────────

function DownloadBtn({
  url,
  label = '下载',
  variant = 'github',
}: {
  url: string
  label?: string
  variant?: keyof typeof PLATFORM_THEME
}) {
  const t = PLATFORM_THEME[variant]
  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      style={{
        display: 'inline-block',
        padding: '3px 11px',
        background: t.bg,
        color: t.text,
        border: `1px solid ${t.border}`,
        borderRadius: 6,
        fontSize: 12,
        fontWeight: 500,
        textDecoration: 'none',
        whiteSpace: 'nowrap',
        transition: 'opacity 0.15s',
      }}
    >
      {label}
    </a>
  )
}

function MacOSCell({ macos, error, releasesUrl }: { macos?: MacOSAssets; error?: boolean; releasesUrl?: string }) {
  if (error && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" />
  if (!macos?.intel && !macos?.apple) return <span style={dashStyle}>—</span>
  if (macos.intel && macos.apple && macos.intel === macos.apple) {
    return <DownloadBtn url={macos.intel} variant="macos" />
  }
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 5, alignItems: 'center' }}>
      {macos.intel && <DownloadBtn url={macos.intel} label="Intel" variant="macos" />}
      {macos.apple && <DownloadBtn url={macos.apple} label="M 芯片" variant="macos" />}
    </div>
  )
}

function PlatformCell({
  url,
  label,
  variant,
  error,
  releasesUrl,
}: {
  url?: string
  label?: string
  variant: keyof typeof PLATFORM_THEME
  error?: boolean
  releasesUrl?: string
}) {
  if (error && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" />
  if (!url) return <span style={dashStyle}>—</span>
  return <DownloadBtn url={url} label={label ?? '下载'} variant={variant} />
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

  const cardStyle: React.CSSProperties = {
    background: 'var(--bg-card)',
    border: '1px solid var(--border-color)',
    borderRadius: 12,
    overflow: 'hidden',
    boxShadow: 'var(--shadow-soft)',
  }

  const cardHeadStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '14px 20px',
    background: 'linear-gradient(135deg, var(--morandi-cream) 0%, var(--bg-secondary) 100%)',
    borderBottom: '1px solid var(--border-color)',
  }

  return (
    <div style={{ maxWidth: 1040 }}>

      {/* ── 页面标题 ── */}
      <div className="page-header">
        <h1>客户端下载</h1>
        <p>主流 Clash 系代理客户端，版本信息每日自动更新</p>
      </div>

      {/* ── 下载表格 ── */}
      <div style={{ ...cardStyle, marginBottom: 20 }}>
        <div style={cardHeadStyle}>
          <span style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)' }}>
            客户端列表
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            {fetchedAt && (
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                缓存至 {cacheTimeStr}
              </span>
            )}
            <Button
              size="small"
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={handleRefresh}
            >
              刷新版本
            </Button>
          </div>
        </div>

        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 700 }}>
            <thead>
              <tr>
                <th style={{
                  textAlign: 'left',
                  padding: '11px 20px',
                  color: 'var(--text-secondary)',
                  fontWeight: 500,
                  fontSize: 12,
                  background: 'var(--bg-secondary)',
                  borderBottom: '1px solid var(--border-color)',
                }}>
                  客户端
                </th>
                <th style={{ ...thStyle, width: 82 }}>版本</th>
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
                  <tr
                    key={client.name}
                    style={{
                      borderBottom: isLast ? 'none' : '1px solid var(--border-color)',
                      transition: 'background 0.15s',
                    }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'rgba(201,169,166,0.04)')}
                    onMouseLeave={e => (e.currentTarget.style.background = '')}
                  >
                    {/* 名称 + 简介 */}
                    <td style={{ padding: '13px 20px' }}>
                      <div style={{ fontWeight: 600, fontSize: 13, color: 'var(--text-primary)', marginBottom: 2 }}>
                        {client.name}
                      </div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                        {client.description}
                      </div>
                    </td>

                    {/* 版本 */}
                    <td style={{ textAlign: 'center', padding: '13px 8px' }}>
                      {client.source === 'appstore' ? (
                        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>App Store</span>
                      ) : loading && !info ? (
                        <Spin size="small" />
                      ) : info?.error ? (
                        <span style={{ fontSize: 11, color: 'var(--morandi-terracotta)' }}>获取失败</span>
                      ) : (
                        <span style={{
                          fontSize: 11,
                          fontFamily: 'monospace',
                          color: 'var(--morandi-dusty-rose)',
                          background: 'rgba(201,169,166,0.1)',
                          border: '1px solid rgba(201,169,166,0.25)',
                          padding: '2px 7px',
                          borderRadius: 4,
                          display: 'inline-block',
                        }}>
                          {info?.version || '—'}
                        </span>
                      )}
                    </td>

                    {/* Windows */}
                    <td style={{ textAlign: 'center', padding: '13px 8px' }}>
                      <PlatformCell url={info?.assets.windows} variant="windows" error={info?.error} releasesUrl={info?.releasesUrl} />
                    </td>

                    {/* macOS */}
                    <td style={{ textAlign: 'center', padding: '13px 8px' }}>
                      <MacOSCell macos={info?.assets.macos} error={info?.error} releasesUrl={info?.releasesUrl} />
                    </td>

                    {/* Linux */}
                    <td style={{ textAlign: 'center', padding: '13px 8px' }}>
                      <PlatformCell url={info?.assets.linux} variant="linux" error={info?.error} releasesUrl={info?.releasesUrl} />
                    </td>

                    {/* Android */}
                    <td style={{ textAlign: 'center', padding: '13px 8px' }}>
                      <PlatformCell url={info?.assets.android} variant="android" error={info?.error} releasesUrl={info?.releasesUrl} />
                    </td>

                    {/* iOS */}
                    <td style={{ textAlign: 'center', padding: '13px 8px' }}>
                      {client.source === 'appstore' ? (
                        <DownloadBtn url={client.storeUrl!} label="前往 ↗" variant="ios" />
                      ) : (
                        <PlatformCell
                          url={info?.assets.ios ?? client.iosStoreUrl}
                          label={client.iosStoreUrl ? '前往 ↗' : '下载'}
                          variant="ios"
                          error={info?.error}
                          releasesUrl={info?.releasesUrl}
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
      <div style={cardStyle}>
        <div style={cardHeadStyle}>
          <div>
            <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)', marginBottom: 2 }}>
              Linux 服务器安装
            </div>
            <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              一键安装 Mihomo 代理服务，自动配置每日订阅更新
            </div>
          </div>
        </div>

        <div style={{ padding: '16px 20px' }}>
          {/* 订阅选择 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
            <span style={{ fontSize: 13, color: 'var(--text-secondary)', whiteSpace: 'nowrap', fontWeight: 500 }}>
              选择订阅
            </span>
            <Select
              style={{ minWidth: 220 }}
              placeholder="选择订阅以预填地址"
              value={selectedSubId}
              onChange={handleSubChange}
              options={subscriptions.map(s => ({ value: s.id, label: s.name }))}
            />
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
              订阅 URL 仅在本地生成，不上传
            </span>
          </div>

          {/* 命令块 */}
          <div style={{ position: 'relative' }}>
            <pre style={{
              background: 'var(--bg-sidebar)',
              borderRadius: 8,
              padding: '14px 48px 14px 18px',
              fontFamily: "'Courier New', 'Noto Sans SC', monospace",
              fontSize: 12.5,
              color: 'var(--morandi-cream)',
              lineHeight: 1.9,
              margin: 0,
              overflowX: 'auto',
              whiteSpace: 'pre',
              border: '1px solid rgba(255,255,255,0.06)',
            }}>
              {installCmd}
            </pre>
            <Button
              size="small"
              icon={<CopyOutlined />}
              onClick={handleCopy}
              loading={copying}
              style={{
                position: 'absolute',
                top: 10,
                right: 10,
                background: 'rgba(201,169,166,0.15)',
                borderColor: 'rgba(201,169,166,0.3)',
                color: 'var(--morandi-dusty-rose)',
              }}
            >
              复制
            </Button>
          </div>

          {/* 直连提示 */}
          <div style={{ marginTop: 10, fontSize: 12, color: 'var(--text-muted)' }}>
            服务器可直连 GitHub？去掉 gh-proxy 前缀并加{' '}
            <code style={{
              background: 'var(--bg-secondary)',
              border: '1px solid var(--border-color)',
              padding: '1px 6px',
              borderRadius: 4,
              color: 'var(--text-secondary)',
              fontSize: 11,
            }}>
              --no-github-proxy
            </code>
          </div>
        </div>
      </div>
    </div>
  )
}
