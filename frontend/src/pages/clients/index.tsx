// frontend/src/pages/clients/index.tsx
import React, { useState, useEffect, useCallback } from 'react'
import { Select, Spin, Button, Switch, message } from 'antd'
import { ReloadOutlined, CopyOutlined, WindowsOutlined, AppleOutlined, AndroidOutlined, MobileOutlined, StarOutlined } from '@ant-design/icons'
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
  stars?: number
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
    stars: 124277,
    platforms: ['windows', 'macos', 'linux'],
    patterns: {
      windows: /clash[._-]verge[._-]rev.*x64.*\.(exe|msi)$/i,
      macosIntel: /clash[._-]verge[._-]rev.*x64.*\.dmg$/i,
      macosApple: /clash[._-]verge[._-]rev.*(aarch64|arm64).*\.dmg$/i,
      linux: /clash[._-]verge[._-]rev.*(amd64|x86_64).*\.AppImage$/i,
    },
  },
  {
    name: 'FlClash',
    description: '基于 Flutter 的现代化跨平台客户端',
    source: 'github',
    repo: 'chen08209/FlClash',
    stars: 41795,
    platforms: ['windows', 'macos', 'linux', 'android'],
    patterns: {
      windows: /FlClash-.*windows.*(amd64|x64).*\.exe$/i,
      macosIntel: /FlClash-.*macos.*(amd64|x64).*\.dmg$/i,
      macosApple: /FlClash-.*macos.*arm64.*\.dmg$/i,
      linux: /FlClash-.*linux.*(amd64|x64).*\.AppImage$/i,
      android: /FlClash-.*android.*\.(apk)$/i,
    },
  },
  {
    name: 'ClashMetaForAndroid',
    description: 'Android 专属 Clash Meta 客户端',
    source: 'github',
    repo: 'MetaCubeX/ClashMetaForAndroid',
    stars: 40949,
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
    stars: 31128,
    platforms: ['windows', 'macos', 'linux', 'android'],
    patterns: {
      windows: /mihomo-windows-amd64[^-].*\.(zip|exe)$/i,
      macosIntel: /mihomo-darwin-amd64[^-].*\.gz$/i,
      macosApple: /mihomo-darwin-arm64[^-].*\.gz$/i,
      linux: /mihomo-linux-amd64[^-].*\.gz$/i,
      android: /mihomo-android-arm64[^-].*\.gz$/i,
    },
  },
  {
    name: 'Hiddify',
    description: '全平台开源代理客户端',
    source: 'github',
    repo: 'hiddify/hiddify-app',
    stars: 30517,
    platforms: ['windows', 'macos', 'linux', 'android', 'ios'],
    iosStoreUrl: 'https://apps.apple.com/app/hiddify-proxy-vpn/id6596777532',
    patterns: {
      windows: /hiddify.*windows.*(setup|installer)?.*x64.*\.exe$/i,
      macosIntel: /hiddify.*macos.*x64.*\.dmg$/i,
      macosApple: /hiddify.*macos.*arm64.*\.dmg$/i,
      linux: /hiddify.*linux.*x64.*\.AppImage$/i,
      android: /hiddify.*android.*universal.*\.apk$/i,
    },
  },
  {
    name: 'ClashMi',
    description: '全平台 Clash Meta 客户端',
    source: 'github',
    repo: 'KaringX/clashmi',
    stars: 7465,
    platforms: ['windows', 'macos', 'linux', 'android', 'ios'],
    patterns: {
      windows: /clashmi.*windows.*(amd64|x64).*\.exe$|clashmi.*\.exe$/i,
      macosIntel: /clashmi.*(amd64|x64).*\.dmg$|clashmi.*macos.*(amd64|x64).*\.dmg$/i,
      macosApple: /clashmi.*(arm64|aarch64).*\.dmg$|clashmi.*macos.*(arm64|aarch64).*\.dmg$/i,
      linux: /clashmi.*linux.*\.AppImage$|clashmi.*\.AppImage$/i,
      android: /clashmi.*\.apk$/i,
      ios: /clashmi.*\.ipa$/i,
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

const formatStars = (n: number) => n >= 1000 ? `${Math.round(n / 1000)}k` : String(n)

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

const dashStyle: React.CSSProperties = { color: 'var(--border-color)', fontSize: 16 }

const thStyle: React.CSSProperties = {
  textAlign: 'center',
  padding: '11px 6px',
  color: 'var(--text-secondary)',
  fontWeight: 500,
  fontSize: 12,
  width: 84,
  background: 'var(--bg-secondary)',
  borderBottom: '1px solid var(--border-color)',
}

// Linux platform icon (no Ant Design equivalent)
const LinuxIcon = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" style={{ verticalAlign: 'middle' }}>
    <path d="M12 2C8.686 2 6 4.686 6 7c0 1.548.75 2.918 1.894 3.789C6.74 11.625 6 13.22 6 15c0 1.306.37 2.52 1.006 3.553C5.804 19.353 5 20.576 5 22h14c0-1.424-.804-2.647-2.006-3.447A7.04 7.04 0 0019 15c0-1.78-.74-3.375-1.894-4.211A4.992 4.992 0 0018 7c0-2.314-2.686-5-6-5zm-1.5 6.25a1.25 1.25 0 110-2.5 1.25 1.25 0 010 2.5zm3 0a1.25 1.25 0 110-2.5 1.25 1.25 0 010 2.5z"/>
  </svg>
)

// ── 子组件 ────────────────────────────────────────────────────────────────────

function DownloadBtn({
  url, label = '下载', variant = 'github',
}: { url: string; label?: string; variant?: keyof typeof PLATFORM_THEME }) {
  const t = PLATFORM_THEME[variant]
  return (
    <a href={url} target="_blank" rel="noreferrer" style={{
      display: 'inline-block',
      padding: '3px 10px',
      background: t.bg,
      color: t.text,
      border: `1px solid ${t.border}`,
      borderRadius: 6,
      fontSize: 12,
      fontWeight: 500,
      textDecoration: 'none',
      whiteSpace: 'nowrap',
    }}>
      {label}
    </a>
  )
}

function MacOSCell({ macos, error, releasesUrl, declared }: {
  macos?: MacOSAssets; error?: boolean; releasesUrl?: string; declared: boolean
}) {
  if (error && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" />
  if (!macos?.intel && !macos?.apple) {
    if (declared && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" variant="github" />
    return <span style={dashStyle}>—</span>
  }
  if (macos.intel && macos.apple && macos.intel === macos.apple) {
    return <DownloadBtn url={macos.intel} variant="macos" />
  }
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, alignItems: 'center' }}>
      {macos.intel && <DownloadBtn url={macos.intel} label="Intel" variant="macos" />}
      {macos.apple && <DownloadBtn url={macos.apple} label="M 芯片" variant="macos" />}
    </div>
  )
}

function PlatformCell({ url, label, variant, error, releasesUrl, declared }: {
  url?: string; label?: string; variant: keyof typeof PLATFORM_THEME;
  error?: boolean; releasesUrl?: string; declared: boolean
}) {
  if (error && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" />
  if (!url) {
    if (declared && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" variant="github" />
    return <span style={dashStyle}>—</span>
  }
  return <DownloadBtn url={url} label={label ?? '下载'} variant={variant} />
}

// ── 主组件 ────────────────────────────────────────────────────────────────────

const INTERVAL_OPTIONS = [
  { value: '30m', label: '30 分钟' },
  { value: '1h',  label: '1 小时' },
  { value: '2h',  label: '2 小时' },
  { value: '6h',  label: '6 小时' },
  { value: '12h', label: '12 小时' },
  { value: 'daily', label: '每天一次' },
]

export default function ClientDownload() {
  const [releases, setReleases] = useState<Record<string, ReleaseInfo>>({})
  const [fetchedAt, setFetchedAt] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [selectedSubId, setSelectedSubId] = useState<string | undefined>()
  const [subLink, setSubLink] = useState<string>('')
  const [copying, setCopying] = useState(false)
  const [updateInterval, setUpdateInterval] = useState('1h')
  const [installPanel, setInstallPanel] = useState(true)

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

  const handleRefresh = () => { clearCache(); load(true) }

  const handleSubChange = async (id: string) => {
    setSelectedSubId(id)
    try {
      const { link } = await subscriptionAPI.getLink(id)
      setSubLink(link)
    } catch {
      setSubLink('')
    }
  }

  const cmdParts = [
    `      --sub '${subLink || '你的订阅地址'}'`,
    `      --secret '你的面板密码'`,
    `      --update-interval ${updateInterval}`,
    ...(!installPanel ? ['      --no-ui'] : []),
  ]
  const installCmd = [
    'curl -fsSL https://gh-proxy.com/https://raw.githubusercontent.com/Jat-echo/InstallMihomo/main/install-mihomo.sh \\',
    '  | sudo bash -s -- \\',
    ...cmdParts.slice(0, -1).map(p => p + ' \\'),
    cmdParts[cmdParts.length - 1],
  ].join('\n')

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

  const optionRowStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    padding: '8px 0',
    borderBottom: '1px solid var(--border-color)',
    marginBottom: 4,
  }

  const optionLabelStyle: React.CSSProperties = {
    fontSize: 13,
    color: 'var(--text-secondary)',
    fontWeight: 500,
    width: 110,
    flexShrink: 0,
  }

  return (
    <div style={{ maxWidth: 1060 }}>

      {/* ── 页面标题 ── */}
      <div className="page-header">
        <h1>客户端下载</h1>
        <p>主流 Clash 系代理客户端，版本信息每日自动更新</p>
      </div>

      {/* ── 下载表格 ── */}
      <div style={{ ...cardStyle, marginBottom: 20 }}>
        <div style={cardHeadStyle}>
          <span style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)' }}>客户端列表</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            {fetchedAt && (
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>缓存至 {cacheTimeStr}</span>
            )}
            <Button size="small" icon={<ReloadOutlined />} loading={loading} onClick={handleRefresh}>
              刷新版本
            </Button>
          </div>
        </div>

        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 720 }}>
            <thead>
              <tr>
                <th style={{
                  textAlign: 'left', padding: '11px 16px',
                  color: 'var(--text-secondary)', fontWeight: 500, fontSize: 12,
                  background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-color)',
                  width: 160,
                }}>客户端</th>
                <th style={{ ...thStyle, width: 60 }}>
                  <StarOutlined style={{ fontSize: 12 }} />
                </th>
                <th style={{ ...thStyle, width: 80 }}>版本</th>
                <th style={thStyle}>
                  <WindowsOutlined style={{ fontSize: 13 }} />
                  <div style={{ fontSize: 10, marginTop: 2, fontWeight: 400 }}>Windows</div>
                </th>
                <th style={thStyle}>
                  <AppleOutlined style={{ fontSize: 13 }} />
                  <div style={{ fontSize: 10, marginTop: 2, fontWeight: 400 }}>macOS</div>
                </th>
                <th style={thStyle}>
                  <LinuxIcon />
                  <div style={{ fontSize: 10, marginTop: 2, fontWeight: 400 }}>Linux</div>
                </th>
                <th style={thStyle}>
                  <AndroidOutlined style={{ fontSize: 13 }} />
                  <div style={{ fontSize: 10, marginTop: 2, fontWeight: 400 }}>Android</div>
                </th>
                <th style={thStyle}>
                  <MobileOutlined style={{ fontSize: 13 }} />
                  <div style={{ fontSize: 10, marginTop: 2, fontWeight: 400 }}>iOS</div>
                </th>
              </tr>
            </thead>
            <tbody>
              {CLIENT_LIST.map((client, idx) => {
                const info = client.repo ? releases[client.repo] : undefined
                const isLast = idx === CLIENT_LIST.length - 1
                const has = (p: typeof client.platforms[number]) => client.platforms.includes(p)
                return (
                  <tr
                    key={client.name}
                    style={{ borderBottom: isLast ? 'none' : '1px solid var(--border-color)', transition: 'background 0.15s' }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'rgba(201,169,166,0.04)')}
                    onMouseLeave={e => (e.currentTarget.style.background = '')}
                  >
                    {/* 名称（紧凑） */}
                    <td style={{ padding: '12px 16px' }}>
                      <div style={{ fontWeight: 600, fontSize: 13, color: 'var(--text-primary)' }}>{client.name}</div>
                    </td>

                    {/* Stars */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      {client.stars ? (
                        <span style={{ fontSize: 11, color: 'var(--morandi-sand)', fontWeight: 500 }}>
                          {formatStars(client.stars)}
                        </span>
                      ) : <span style={dashStyle}>—</span>}
                    </td>

                    {/* 版本 */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      {client.source === 'appstore' ? (
                        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>App Store</span>
                      ) : loading && !info ? (
                        <Spin size="small" />
                      ) : info?.error ? (
                        <span style={{ fontSize: 11, color: 'var(--morandi-terracotta)' }}>失败</span>
                      ) : (
                        <span style={{
                          fontSize: 11, fontFamily: 'monospace',
                          color: 'var(--morandi-dusty-rose)',
                          background: 'rgba(201,169,166,0.1)',
                          border: '1px solid rgba(201,169,166,0.25)',
                          padding: '2px 6px', borderRadius: 4, display: 'inline-block',
                        }}>
                          {info?.version || '—'}
                        </span>
                      )}
                    </td>

                    {/* Windows */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <PlatformCell url={info?.assets.windows} variant="windows"
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('windows')} />
                    </td>

                    {/* macOS */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <MacOSCell macos={info?.assets.macos}
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('macos')} />
                    </td>

                    {/* Linux */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <PlatformCell url={info?.assets.linux} variant="linux"
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('linux')} />
                    </td>

                    {/* Android */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <PlatformCell url={info?.assets.android} variant="android"
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('android')} />
                    </td>

                    {/* iOS */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      {client.source === 'appstore' ? (
                        <DownloadBtn url={client.storeUrl!} label="前往 ↗" variant="ios" />
                      ) : (
                        <PlatformCell
                          url={info?.assets.ios ?? client.iosStoreUrl}
                          label={client.iosStoreUrl ? '前往 ↗' : '下载'}
                          variant="ios"
                          error={info?.error} releasesUrl={info?.releasesUrl}
                          declared={has('ios')}
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
              一键安装 Mihomo 代理服务并自动配置订阅更新
            </div>
          </div>
        </div>

        <div style={{ padding: '16px 20px' }}>
          {/* 选项区 */}
          <div style={{
            background: 'var(--bg-secondary)',
            borderRadius: 8,
            padding: '12px 16px',
            marginBottom: 14,
            border: '1px solid var(--border-color)',
          }}>
            {/* 订阅选择 */}
            <div style={optionRowStyle}>
              <span style={optionLabelStyle}>订阅链接</span>
              <Select
                style={{ minWidth: 220 }}
                placeholder="选择订阅以预填地址"
                value={selectedSubId}
                onChange={handleSubChange}
                options={subscriptions.map(s => ({ value: s.id, label: s.name }))}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>仅在本地生成，不上传</span>
            </div>

            {/* 自动更新间隔 */}
            <div style={optionRowStyle}>
              <span style={optionLabelStyle}>自动更新间隔</span>
              <Select
                style={{ width: 140 }}
                value={updateInterval}
                onChange={setUpdateInterval}
                options={INTERVAL_OPTIONS}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>定时重拉订阅并重启服务</span>
            </div>

            {/* 安装控制面板 */}
            <div style={{ ...optionRowStyle, borderBottom: 'none', paddingBottom: 0, marginBottom: 0 }}>
              <span style={optionLabelStyle}>安装控制面板</span>
              <Switch
                size="small"
                checked={installPanel}
                onChange={setInstallPanel}
                style={installPanel ? { background: 'var(--morandi-dusty-rose)' } : {}}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                {installPanel ? '安装 MetaCubeXD 面板（推荐）' : '跳过面板安装，仅安装核心'}
              </span>
            </div>
          </div>

          {/* 命令块 */}
          <div style={{ position: 'relative' }}>
            <pre style={{
              background: 'var(--bg-sidebar)',
              borderRadius: 8,
              padding: '14px 56px 14px 18px',
              fontFamily: "'Courier New', monospace",
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
                position: 'absolute', top: 10, right: 10,
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
              background: 'var(--bg-secondary)', border: '1px solid var(--border-color)',
              padding: '1px 6px', borderRadius: 4, color: 'var(--text-secondary)', fontSize: 11,
            }}>
              --no-github-proxy
            </code>
          </div>
        </div>
      </div>
    </div>
  )
}
