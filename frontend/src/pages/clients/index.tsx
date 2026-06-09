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
