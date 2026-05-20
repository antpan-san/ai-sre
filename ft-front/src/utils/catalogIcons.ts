import type { Component } from 'vue'
import {
  Box,
  Coin,
  Connection,
  Cpu,
  DataLine,
  Download,
  FolderOpened,
  Link,
  MagicStick,
  Message,
  Monitor,
  Odometer,
  Platform,
  Reading,
  Refresh,
  Search,
  Setting,
  Share,
  Tools,
  TrendCharts,
  View
} from '@element-plus/icons-vue'

const ICON_MAP: Record<string, Component> = {
  Box,
  Coin,
  Connection,
  Cpu,
  DataLine,
  Download,
  FolderOpened,
  Link,
  MagicStick,
  Message,
  Monitor,
  Odometer,
  Platform,
  Reading,
  Refresh,
  Search,
  Setting,
  Share,
  Tools,
  TrendCharts,
  View
}

export function resolveCatalogIcon(name?: string): Component | null {
  if (!name) return null
  return ICON_MAP[name] || null
}
