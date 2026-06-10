// frontend/src/pages/clients/index.tsx
import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { Select, Spin, Button, Switch, message } from 'antd'
import { ReloadOutlined, CopyOutlined, WindowsOutlined, AppleOutlined, AndroidOutlined, MobileOutlined, StarOutlined } from '@ant-design/icons'
import { subscriptionAPI, Subscription } from '../../services/api'

// ── 类型定义 ──────────────────────────────────────────────────────────────────

interface AssetEntry {
  pattern: RegExp
  label: string   // 显示在按钮上的文字，如 "x64"、"arm64"、"universal"
}

interface PlatformVariant {
  url: string
  label: string
}

interface MacOSAssets {
  intel?: string
  apple?: string
}

interface ResolvedAssets {
  windows?: PlatformVariant[]
  macos?: MacOSAssets
  linux?: PlatformVariant[]
  android?: PlatformVariant[]
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
  windows?: AssetEntry[]       // 多版本数组
  macosIntel?: RegExp
  macosApple?: RegExp
  macosUniversal?: RegExp
  linux?: AssetEntry[]         // 多版本数组
  android?: AssetEntry[]       // 多版本数组
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
      windows: [
        { pattern: /Clash\.Verge_.*x64.*\.(exe|msi)$/i,   label: 'x64' },
        { pattern: /Clash\.Verge_.*arm64.*\.(exe|msi)$/i,  label: 'arm64' },
      ],
      macosIntel: /Clash\.Verge_.*x64\.dmg$/i,
      macosApple: /Clash\.Verge_.*(aarch64|arm64)\.dmg$/i,
      linux: [
        { pattern: /Clash\.Verge_.*amd64\.deb$/i,       label: 'x64 .deb' },
        { pattern: /Clash\.Verge-.*x86_64\.rpm$/i,       label: 'x64 .rpm' },
        { pattern: /Clash\.Verge_.*arm64\.deb$/i,        label: 'arm64 .deb' },
      ],
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
      windows: [
        { pattern: /FlClash-.*windows-amd64.*\.exe$/i,  label: 'x64' },
        { pattern: /FlClash-.*windows-arm64.*\.exe$/i,  label: 'arm64' },
      ],
      macosIntel: /FlClash-.*macos-amd64\.dmg$/i,
      macosApple: /FlClash-.*macos-arm64\.dmg$/i,
      linux: [
        { pattern: /FlClash-.*linux-amd64\.AppImage$/i, label: 'x64' },
      ],
      android: [
        { pattern: /FlClash-.*android-arm64.*\.apk$/i,     label: 'arm64' },
        { pattern: /FlClash-.*android-armeabi.*\.apk$/i,   label: 'arm' },
        { pattern: /FlClash-.*android-x86_64.*\.apk$/i,    label: 'x86_64' },
      ],
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
      android: [
        { pattern: /cmfa-.*universal.*\.apk$/i,    label: 'universal' },
        { pattern: /cmfa-.*arm64-v8a.*\.apk$/i,    label: 'arm64' },
        { pattern: /cmfa-.*armeabi-v7a.*\.apk$/i,  label: 'arm' },
      ],
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
      windows: [
        { pattern: /mihomo-windows-amd64-v\d+\.\d+\.\d+\.zip$/i,  label: 'x64' },
        { pattern: /mihomo-windows-arm64-v\d+\.\d+\.\d+\.zip$/i,  label: 'arm64' },
      ],
      macosIntel: /mihomo-darwin-amd64-v\d+\.\d+\.\d+\.gz$/i,
      macosApple: /mihomo-darwin-arm64-v\d+\.\d+\.\d+\.gz$/i,
      linux: [
        { pattern: /mihomo-linux-amd64-v\d+\.\d+\.\d+\.gz$/i,    label: 'x64' },
      ],
      android: [
        { pattern: /mihomo-android-arm64-v8-v\d+\.\d+\.\d+\.gz$/i, label: 'arm64' },
      ],
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
      windows: [
        { pattern: /Hiddify-Windows-Setup-x64\.exe$/i, label: 'x64' },
      ],
      macosUniversal: /Hiddify-MacOS\.dmg$/i,
      linux: [
        { pattern: /Hiddify-Linux-x64-AppImage\.AppImage$/i, label: 'x64' },
      ],
      android: [
        { pattern: /Hiddify-Android-universal\.apk$/i, label: 'universal' },
      ],
    },
  },
  {
    name: 'ClashMi',
    description: '全平台 Clash Meta 客户端',
    source: 'github',
    repo: 'KaringX/clashmi',
    stars: 7465,
    platforms: ['windows', 'macos', 'linux', 'android', 'ios'],
    iosStoreUrl: 'https://apps.apple.com/us/app/clash-mi/id6744321968',
    patterns: {
      windows: [
        { pattern: /clashmi.*windows.*x64\.exe$/i,       label: 'x64' },
      ],
      macosUniversal: /clashmi.*macos.*universal.*\.dmg$/i,
      linux: [
        { pattern: /clashmi.*linux.*amd64\.AppImage$/i,  label: 'x64' },
      ],
      android: [
        { pattern: /clashmi.*android.*arm64.*\.apk$/i,    label: 'arm64' },
        { pattern: /clashmi.*android.*armeabi.*\.apk$/i,  label: 'arm' },
      ],
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
  const findOne = (re: RegExp) =>
    rawAssets.find(a => re.test(a.name))?.browser_download_url

  const findAll = (entries: AssetEntry[]): PlatformVariant[] =>
    entries
      .map(e => {
        const hit = rawAssets.find(a => e.pattern.test(a.name))
        return hit ? { url: hit.browser_download_url, label: e.label } : null
      })
      .filter((v): v is PlatformVariant => v !== null)

  const result: ResolvedAssets = {}

  if (patterns.windows) {
    const variants = findAll(patterns.windows)
    if (variants.length) result.windows = variants
  }

  // macOS: universal 作为 Intel 和 M芯片的 fallback
  const universalUrl = patterns.macosUniversal ? findOne(patterns.macosUniversal) : undefined
  const intelUrl     = patterns.macosIntel     ? findOne(patterns.macosIntel)     : undefined
  const appleUrl     = patterns.macosApple     ? findOne(patterns.macosApple)     : undefined
  const resolvedIntel = intelUrl ?? universalUrl
  const resolvedApple = appleUrl ?? universalUrl
  if (resolvedIntel || resolvedApple) {
    result.macos = { intel: resolvedIntel, apple: resolvedApple }
  }

  if (patterns.linux) {
    const variants = findAll(patterns.linux)
    if (variants.length) result.linux = variants
  }

  if (patterns.android) {
    const variants = findAll(patterns.android)
    if (variants.length) result.android = variants
  }

  if (patterns.ios) {
    const url = findOne(patterns.ios)
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

// Bump version suffix whenever ReleaseInfo/ResolvedAssets schema changes to invalidate stale caches
const CACHE_KEY = 'v2raydash_clients_cache_v4'
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
  const githubClients = CLIENT_LIST.filter(c => c.source === 'github' && !!c.repo)
  const results = await Promise.all(githubClients.map(c => fetchRelease(c)))
  const data: Record<string, ReleaseInfo> = {}
  githubClients.forEach((c, i) => { data[c.repo!] = results[i] })
  try { saveCache(data) } catch { /* quota exceeded — skip caching */ }
  return { data, fetchedAt: Date.now() }
}

// ── 平台配色系统（Morandi 调色板） ────────────────────────────────────────────
// TODO: expose these as CSS custom properties in theme.css so they respond to theme changes

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

function PlatformCell({ variants, variant: theme, error, releasesUrl, declared }: {
  variants?: PlatformVariant[]
  variant: keyof typeof PLATFORM_THEME
  error?: boolean; releasesUrl?: string; declared: boolean
}) {
  if (error && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" />
  if (!variants?.length) {
    if (declared && releasesUrl) return <DownloadBtn url={releasesUrl} label="GitHub ↗" variant="github" />
    return <span style={dashStyle}>—</span>
  }
  if (variants.length === 1) {
    return <DownloadBtn url={variants[0].url} label={variants[0].label} variant={theme} />
  }
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, alignItems: 'center' }}>
      {variants.map(v => (
        <DownloadBtn key={v.label} url={v.url} label={v.label} variant={theme} />
      ))}
    </div>
  )
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
  const [editedCmd, setEditedCmd] = useState('')

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
    subscriptionAPI.list().then(list => setSubscriptions(list ?? [])).catch(() => {})
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

  const installCmd = useMemo(() => {
    // Escape single quotes to prevent shell quoting breakage
    const safeSubLink = (subLink || '你的订阅地址').replace(/'/g, "'\\''")
    const parts = [
      `      --sub '${safeSubLink}'`,
      ...(installPanel ? [`      --secret '你的面板密码'`] : []),
      `      --update-interval ${updateInterval}`,
      ...(!installPanel ? ['      --no-ui'] : []),
    ]
    return [
      'curl -fsSL https://gh-proxy.com/https://raw.githubusercontent.com/Jat-echo/InstallMihomo/main/install-mihomo.sh \\',
      '  | sudo bash -s -- \\',
      ...parts.slice(0, -1).map(p => p + ' \\'),
      parts[parts.length - 1],
    ].join('\n')
  }, [subLink, updateInterval, installPanel])

  // 选项变化时同步命令（用户手动编辑的内容会被覆盖）
  useEffect(() => { setEditedCmd(installCmd) }, [installCmd])

  const handleCopy = async () => {
    const text = editedCmd
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text)
      } else {
        // HTTP 环境降级方案
        const el = document.createElement('textarea')
        el.value = text
        el.style.cssText = 'position:fixed;opacity:0;pointer-events:none'
        document.body.appendChild(el)
        el.select()
        document.execCommand('copy')
        document.body.removeChild(el)
      }
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

  const optionLabelStyle: React.CSSProperties = {
    fontSize: 12,
    color: 'var(--text-secondary)',
    fontWeight: 500,
    whiteSpace: 'nowrap',
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
                <th style={{ ...thStyle, width: 70 }}>GitHub</th>
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
                    {/* 名称 + 描述 */}
                    <td style={{ padding: '12px 16px' }}>
                      <div style={{ fontWeight: 600, fontSize: 13, color: 'var(--text-primary)' }}>{client.name}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{client.description}</div>
                    </td>

                    {/* Stars（可点击跳转 GitHub） */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      {client.stars && client.repo ? (
                        <a
                          href={`https://github.com/${client.repo}`}
                          target="_blank"
                          rel="noreferrer"
                          style={{ fontSize: 12, color: 'var(--text-secondary)', fontWeight: 500, textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 3 }}
                        >
                          <StarOutlined style={{ fontSize: 11, color: 'var(--morandi-terracotta)' }} />
                          {formatStars(client.stars)}
                        </a>
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
                        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                          {info?.version || '—'}
                        </span>
                      )}
                    </td>

                    {/* Windows */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <PlatformCell variants={info?.assets.windows} variant="windows"
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('windows')} />
                    </td>

                    {/* macOS */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <MacOSCell macos={info?.assets.macos}
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('macos')} />
                    </td>

                    {/* Linux */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <PlatformCell variants={info?.assets.linux} variant="linux"
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('linux')} />
                    </td>

                    {/* Android */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      <PlatformCell variants={info?.assets.android} variant="android"
                        error={info?.error} releasesUrl={info?.releasesUrl} declared={has('android')} />
                    </td>

                    {/* iOS */}
                    <td style={{ textAlign: 'center', padding: '12px 6px' }}>
                      {client.source === 'appstore' ? (
                        <DownloadBtn url={client.storeUrl!} label="前往 ↗" variant="ios" />
                      ) : (() => {
                        const iosUrl = info?.assets.ios ?? client.iosStoreUrl
                        const iosLabel = client.iosStoreUrl ? '前往 ↗' : '下载'
                        return (
                          <PlatformCell
                            variants={iosUrl ? [{ url: iosUrl, label: iosLabel }] : undefined}
                            variant="ios"
                            error={info?.error} releasesUrl={info?.releasesUrl}
                            declared={has('ios')}
                          />
                        )
                      })()}
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
          {/* 选项区 — 单行 */}
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 20,
            flexWrap: 'wrap',
            background: 'var(--bg-secondary)',
            borderRadius: 8,
            padding: '10px 14px',
            marginBottom: 14,
            border: '1px solid var(--border-color)',
          }}>
            {/* 订阅链接 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={optionLabelStyle}>订阅链接</span>
              <Select
                showSearch
                filterOption={(input, opt) =>
                  (opt?.label as string ?? '').toLowerCase().includes(input.toLowerCase())
                }
                style={{ minWidth: 180 }}
                placeholder="选择或搜索订阅"
                value={selectedSubId}
                onChange={handleSubChange}
                options={subscriptions.map(s => ({ value: s.id, label: s.name }))}
              />
            </div>

            <div style={{ width: 1, height: 20, background: 'var(--border-color)', flexShrink: 0 }} />

            {/* 更新间隔 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={optionLabelStyle}>更新间隔</span>
              <Select
                style={{ width: 120 }}
                value={updateInterval}
                onChange={setUpdateInterval}
                options={INTERVAL_OPTIONS}
              />
            </div>

            <div style={{ width: 1, height: 20, background: 'var(--border-color)', flexShrink: 0 }} />

            {/* 控制面板 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={optionLabelStyle}>控制面板</span>
              <Switch
                size="small"
                checked={installPanel}
                onChange={setInstallPanel}
                style={installPanel ? { background: 'var(--morandi-dusty-rose)' } : {}}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                {installPanel ? 'MetaCubeXD' : '不安装'}
              </span>
            </div>
          </div>

          {/* 命令块（可编辑） */}
          {editedCmd !== installCmd && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
              <span style={{ fontSize: 11, color: 'var(--morandi-terracotta)' }}>已手动修改</span>
              <a
                onClick={() => setEditedCmd(installCmd)}
                style={{ fontSize: 11, color: 'var(--text-muted)', cursor: 'pointer', textDecoration: 'underline' }}
              >
                重置
              </a>
            </div>
          )}
          <div style={{ position: 'relative' }}>
            <textarea
              value={editedCmd}
              onChange={e => setEditedCmd(e.target.value)}
              spellCheck={false}
              style={{
                width: '100%',
                background: 'var(--bg-sidebar)',
                borderRadius: 8,
                padding: '14px 60px 14px 18px',
                fontFamily: "'Courier New', monospace",
                fontSize: 12.5,
                color: 'var(--morandi-cream)',
                lineHeight: 1.9,
                border: '1px solid rgba(255,255,255,0.06)',
                resize: 'vertical',
                minHeight: 180,
                outline: 'none',
                boxSizing: 'border-box',
                whiteSpace: 'pre',
                overflowX: 'auto',
              }}
            />
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
