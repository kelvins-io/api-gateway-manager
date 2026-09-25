export interface RtPair {
  key: string
  value: string
}

export interface RtNameGroups {
  headers: string[]
  querystring: string[]
  body: string[]
}

export interface RtPairGroups {
  headers: RtPair[]
  querystring: RtPair[]
  body: RtPair[]
}

export interface RequestTransformerModel {
  http_method: string
  remove: RtNameGroups
  rename: RtPairGroups
  replace: RtPairGroups & { uri: string }
  add: RtPairGroups
  append: RtPairGroups
}

export function emptyRequestTransformer(): RequestTransformerModel {
  return {
    http_method: '',
    remove: { headers: [], querystring: [], body: [] },
    rename: { headers: [], querystring: [], body: [] },
    replace: { headers: [], querystring: [], body: [], uri: '' },
    add: { headers: [], querystring: [], body: [] },
    append: { headers: [], querystring: [], body: [] },
  }
}

function asStringList(v: unknown): string[] {
  if (!Array.isArray(v)) return []
  return v.map((x) => String(x).trim()).filter(Boolean)
}

function asPairs(v: unknown): RtPair[] {
  if (!Array.isArray(v)) return []
  const out: RtPair[] = []
  for (const item of v) {
    const s = String(item)
    const idx = s.indexOf(':')
    if (idx <= 0) continue
    out.push({ key: s.slice(0, idx).trim(), value: s.slice(idx + 1) })
  }
  return out
}

function pairsToColon(list: RtPair[]): string[] {
  return list
    .map((p) => ({ key: p.key.trim(), value: p.value }))
    .filter((p) => p.key !== '')
    .map((p) => `${p.key}:${p.value}`)
}

function nameGroupsFrom(src: unknown): RtNameGroups {
  const o = src && typeof src === 'object' ? (src as Record<string, unknown>) : {}
  return {
    headers: asStringList(o.headers),
    querystring: asStringList(o.querystring),
    body: asStringList(o.body),
  }
}

function pairGroupsFrom(src: unknown): RtPairGroups {
  const o = src && typeof src === 'object' ? (src as Record<string, unknown>) : {}
  return {
    headers: asPairs(o.headers),
    querystring: asPairs(o.querystring),
    body: asPairs(o.body),
  }
}

export function fillRequestTransformer(src?: Record<string, unknown>): RequestTransformerModel {
  const base = emptyRequestTransformer()
  if (!src) return base
  base.http_method = src.http_method != null ? String(src.http_method) : ''
  base.remove = nameGroupsFrom(src.remove)
  base.rename = pairGroupsFrom(src.rename)
  const replace = pairGroupsFrom(src.replace)
  const replaceSrc = src.replace && typeof src.replace === 'object' ? (src.replace as Record<string, unknown>) : {}
  base.replace = {
    ...replace,
    uri: replaceSrc.uri != null && replaceSrc.uri !== '' ? String(replaceSrc.uri) : '',
  }
  base.add = pairGroupsFrom(src.add)
  base.append = pairGroupsFrom(src.append)
  return base
}

export function buildRequestTransformerPayload(model: RequestTransformerModel): Record<string, unknown> {
  const out: Record<string, unknown> = {
    remove: {
      headers: [...model.remove.headers].map((s) => s.trim()).filter(Boolean),
      querystring: [...model.remove.querystring].map((s) => s.trim()).filter(Boolean),
      body: [...model.remove.body].map((s) => s.trim()).filter(Boolean),
    },
    rename: {
      headers: pairsToColon(model.rename.headers),
      querystring: pairsToColon(model.rename.querystring),
      body: pairsToColon(model.rename.body),
    },
    replace: {
      headers: pairsToColon(model.replace.headers),
      querystring: pairsToColon(model.replace.querystring),
      body: pairsToColon(model.replace.body),
      ...(model.replace.uri.trim() ? { uri: model.replace.uri.trim() } : {}),
    },
    add: {
      headers: pairsToColon(model.add.headers),
      querystring: pairsToColon(model.add.querystring),
      body: pairsToColon(model.add.body),
    },
    append: {
      headers: pairsToColon(model.append.headers),
      querystring: pairsToColon(model.append.querystring),
      body: pairsToColon(model.append.body),
    },
  }
  if (model.http_method) out.http_method = model.http_method
  return out
}

export const JSON_TYPES = ['string', 'number', 'boolean'] as const
export type JsonValueType = (typeof JSON_TYPES)[number]

export interface RtJsonPair {
  key: string
  value: string
  type: JsonValueType
}

export interface ResponseTransformerModel {
  remove: { json: string[]; headers: string[] }
  rename: { headers: RtPair[] }
  replace: { json: RtJsonPair[]; headers: RtPair[] }
  add: { json: RtJsonPair[]; headers: RtPair[] }
  append: { json: RtJsonPair[]; headers: RtPair[] }
}

export function emptyResponseTransformer(): ResponseTransformerModel {
  return {
    remove: { json: [], headers: [] },
    rename: { headers: [] },
    replace: { json: [], headers: [] },
    add: { json: [], headers: [] },
    append: { json: [], headers: [] },
  }
}

function asJsonPairs(json: unknown, types: unknown): RtJsonPair[] {
  const pairs = asPairs(json)
  const typeList = Array.isArray(types) ? types.map(String) : []
  return pairs.map((p, i) => {
    const t = typeList[i]
    const type: JsonValueType = t === 'number' || t === 'boolean' || t === 'string' ? t : 'string'
    return { key: p.key, value: p.value, type }
  })
}

function jsonPairsToKong(list: RtJsonPair[]): { json: string[]; json_types: string[] } {
  const json: string[] = []
  const json_types: string[] = []
  for (const p of list) {
    const key = p.key.trim()
    if (!key) continue
    json.push(`${key}:${p.value}`)
    json_types.push(p.type || 'string')
  }
  return { json, json_types }
}

function respNameGroupsFrom(src: unknown): { json: string[]; headers: string[] } {
  const o = src && typeof src === 'object' ? (src as Record<string, unknown>) : {}
  return {
    json: asStringList(o.json),
    headers: asStringList(o.headers),
  }
}

function respJsonOpFrom(src: unknown): { json: RtJsonPair[]; headers: RtPair[] } {
  const o = src && typeof src === 'object' ? (src as Record<string, unknown>) : {}
  return {
    json: asJsonPairs(o.json, o.json_types),
    headers: asPairs(o.headers),
  }
}

export function fillResponseTransformer(src?: Record<string, unknown>): ResponseTransformerModel {
  const base = emptyResponseTransformer()
  if (!src) return base
  base.remove = respNameGroupsFrom(src.remove)
  const renameSrc = src.rename && typeof src.rename === 'object' ? (src.rename as Record<string, unknown>) : {}
  base.rename = { headers: asPairs(renameSrc.headers) }
  base.replace = respJsonOpFrom(src.replace)
  base.add = respJsonOpFrom(src.add)
  base.append = respJsonOpFrom(src.append)
  return base
}

export function buildResponseTransformerPayload(model: ResponseTransformerModel): Record<string, unknown> {
  const replaceJson = jsonPairsToKong(model.replace.json)
  const addJson = jsonPairsToKong(model.add.json)
  const appendJson = jsonPairsToKong(model.append.json)
  return {
    remove: {
      json: [...model.remove.json].map((s) => s.trim()).filter(Boolean),
      headers: [...model.remove.headers].map((s) => s.trim()).filter(Boolean),
    },
    rename: {
      headers: pairsToColon(model.rename.headers),
    },
    replace: {
      json: replaceJson.json,
      json_types: replaceJson.json_types,
      headers: pairsToColon(model.replace.headers),
    },
    add: {
      json: addJson.json,
      json_types: addJson.json_types,
      headers: pairsToColon(model.add.headers),
    },
    append: {
      json: appendJson.json,
      json_types: appendJson.json_types,
      headers: pairsToColon(model.append.headers),
    },
  }
}
