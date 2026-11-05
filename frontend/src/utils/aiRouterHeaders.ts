import type { PluginItem } from '@/types'

export const AI_ROUTER_VIRTUAL_KEY_HEADER = 'Authorization'

export interface HeaderKV {
  name: string
  value: string
}

function asRecord(v: unknown): Record<string, unknown> | null {
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : null
}

function firstVirtualKey(config: Record<string, unknown>): string {
  const master = String(config.master_key || '').trim()
  if (master) return master
  const list = config.virtual_keys
  if (!Array.isArray(list)) return ''
  for (const item of list) {
    const row = asRecord(item)
    if (!row) continue
    if (row.blocked === true) continue
    const key = String(row.key || '').trim()
    if (key) return key
  }
  return ''
}

/** Pick Authorization Bearer value from linked ai-router plugins with virtual keys enabled. */
export function resolveAIRouterVirtualKeyHeader(
  plugins: Array<Pick<PluginItem, 'plugin' | 'enabled' | 'config'> | null | undefined>,
): HeaderKV | null {
  for (const p of plugins) {
    if (!p || !p.enabled || p.plugin !== 'ai-router') continue
    const cfg = asRecord(p.config) || {}
    if (cfg.enable_virtual_keys !== true) continue
    const token = firstVirtualKey(cfg)
    if (!token) continue
    const value = token.toLowerCase().startsWith('bearer ') ? token : `Bearer ${token}`
    return { name: AI_ROUTER_VIRTUAL_KEY_HEADER, value }
  }
  return null
}

/**
 * When linked ai-router enables virtual keys, preset/refresh Authorization.
 * Does not remove Authorization if virtual keys are later disabled (avoid wiping manual edits).
 */
export function syncAIRouterVirtualKeyHeaders(
  headerList: HeaderKV[],
  plugins: Array<Pick<PluginItem, 'plugin' | 'enabled' | 'config'> | null | undefined>,
): HeaderKV[] {
  const preset = resolveAIRouterVirtualKeyHeader(plugins)
  if (!preset) return headerList
  const nameLc = AI_ROUTER_VIRTUAL_KEY_HEADER.toLowerCase()
  const others = headerList.filter((h) => (h.name || '').trim().toLowerCase() !== nameLc)
  return [...others, { name: preset.name, value: preset.value }]
}
