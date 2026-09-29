<template>
  <el-dialog
    :model-value="modelValue"
    :title="dialogTitle"
    width="960px"
    top="4vh"
    destroy-on-close
    append-to-body
    class="api-debugger-dialog"
    @update:model-value="emit('update:modelValue', $event)"
    @opened="onOpened"
  >
    <div class="debugger">
      <div class="req-bar">
        <el-select v-model="method" class="method-select">
          <el-option v-for="m in methodOptions" :key="m" :label="m" :value="m" />
        </el-select>
        <el-input
          v-model="url"
          class="url-input"
          placeholder="https://example.com/path"
          clearable
          @keyup.enter="send"
        />
        <el-button type="primary" :loading="sending" @click="send">发送</el-button>
      </div>

      <div class="proxy-row">
        <span class="proxy-label">请求方式</span>
        <el-radio-group v-model="proxyMode" size="small">
          <el-radio-button value="auto">自动（CORS 失败走代理）</el-radio-button>
          <el-radio-button value="direct">浏览器直连</el-radio-button>
          <el-radio-button value="proxy">服务端代理</el-radio-button>
        </el-radio-group>
        <el-tag v-if="usedProxy" type="warning" size="small" effect="plain">已走服务端代理</el-tag>
      </div>

      <el-tabs v-model="reqTab" class="req-tabs">
        <el-tab-pane label="Params" name="params">
          <KeyValueEditor v-model="params" key-placeholder="参数名" value-placeholder="参数值" />
        </el-tab-pane>
        <el-tab-pane label="Headers" name="headers">
          <KeyValueEditor v-model="headers" key-placeholder="Header 名" value-placeholder="值" />
        </el-tab-pane>
        <el-tab-pane label="Body" name="body">
          <div class="body-toolbar">
            <el-radio-group v-model="bodyType" size="small">
              <el-radio-button value="none">none</el-radio-button>
              <el-radio-button value="json">JSON</el-radio-button>
              <el-radio-button value="raw">Raw</el-radio-button>
              <el-radio-button value="x-www-form-urlencoded">x-www-form-urlencoded</el-radio-button>
            </el-radio-group>
            <el-button v-if="bodyType === 'json'" link type="primary" @click="formatJsonBody">
              格式化 JSON
            </el-button>
          </div>
          <el-input
            v-if="bodyType === 'json' || bodyType === 'raw'"
            v-model="bodyText"
            type="textarea"
            :rows="8"
            :placeholder="bodyPlaceholder"
            class="body-textarea"
          />
          <KeyValueEditor
            v-else-if="bodyType === 'x-www-form-urlencoded'"
            v-model="formFields"
            key-placeholder="字段名"
            value-placeholder="字段值"
          />
          <el-empty v-else description="该请求无 Body" :image-size="56" />
        </el-tab-pane>
        <el-tab-pane label="Auth" name="auth">
          <el-form label-width="100px" class="auth-form">
            <el-form-item label="认证类型">
              <el-select v-model="authType" style="width: 220px">
                <el-option label="无" value="none" />
                <el-option label="API Key" value="apikey" />
                <el-option label="Basic Auth" value="basic" />
                <el-option label="Bearer Token" value="bearer" />
              </el-select>
            </el-form-item>
            <template v-if="authType === 'apikey'">
              <el-form-item label="Key 名称">
                <el-input v-model="authKeyName" placeholder="apikey" />
              </el-form-item>
              <el-form-item label="Key 值">
                <el-input v-model="authKeyValue" placeholder="YOUR_API_KEY" show-password />
              </el-form-item>
              <el-form-item label="位置">
                <el-radio-group v-model="authKeyIn">
                  <el-radio value="header">Header</el-radio>
                  <el-radio value="query">Query</el-radio>
                </el-radio-group>
              </el-form-item>
            </template>
            <template v-else-if="authType === 'basic'">
              <el-form-item label="用户名">
                <el-input v-model="authUsername" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input v-model="authPassword" show-password />
              </el-form-item>
            </template>
            <template v-else-if="authType === 'bearer'">
              <el-form-item label="Token">
                <el-input
                  v-model="authBearer"
                  type="textarea"
                  :rows="3"
                  placeholder="JWT / Access Token"
                />
              </el-form-item>
            </template>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <div class="resp-section">
        <div class="resp-head">
          <span class="resp-title">Response</span>
          <template v-if="response">
            <el-tag :type="statusTagType(response.status)" size="small">
              {{ response.status }} {{ statusLabel(response.status) }}
            </el-tag>
            <span class="meta">{{ response.durationMs }} ms</span>
            <span class="meta">{{ formatSize(response.size) }}</span>
            <el-tag v-if="response.truncated" type="info" size="small" effect="plain">响应已截断</el-tag>
          </template>
          <span v-else-if="!sending" class="meta">尚未发送请求</span>
        </div>
        <el-alert
          v-if="errorMsg"
          :title="errorMsg"
          type="error"
          show-icon
          :closable="false"
          class="err-alert"
        />
        <el-tabs v-if="response" v-model="respTab" class="resp-tabs">
          <el-tab-pane label="Body" name="body">
            <pre class="resp-pre">{{ prettyBody }}</pre>
          </el-tab-pane>
          <el-tab-pane :label="`Headers (${respHeaderCount})`" name="headers">
            <el-table :data="respHeaderRows" size="small" stripe empty-text="无响应头" max-height="240">
              <el-table-column prop="name" label="名称" width="200" />
              <el-table-column prop="value" label="值" min-width="320" />
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { ApiItem, ConsumerItem } from '@/types'
import { proxyDebugRequest } from '@/api/debug'
import KeyValueEditor from './debugger/KeyValueEditor.vue'
import type { KvPair } from './debugger/KeyValueEditor.vue'

interface DebugResponse {
  status: number
  statusText: string
  headers: Record<string, string[]>
  body: string
  durationMs: number
  size: number
  truncated: boolean
}

const props = defineProps<{
  modelValue: boolean
  api?: ApiItem | null
  consumer?: ConsumerItem | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const method = ref('GET')
const url = ref('')
const proxyMode = ref<'auto' | 'direct' | 'proxy'>('auto')
const usedProxy = ref(false)
const sending = ref(false)
const reqTab = ref('headers')
const respTab = ref('body')
const errorMsg = ref('')

const params = ref<KvPair[]>([emptyPair()])
const headers = ref<KvPair[]>([emptyPair()])
const formFields = ref<KvPair[]>([emptyPair()])
const bodyType = ref<'none' | 'json' | 'raw' | 'x-www-form-urlencoded'>('none')
const bodyText = ref('')

const authType = ref<'none' | 'apikey' | 'basic' | 'bearer'>('none')
const authKeyName = ref('apikey')
const authKeyValue = ref('')
const authKeyIn = ref<'header' | 'query'>('header')
const authUsername = ref('')
const authPassword = ref('')
const authBearer = ref('')

const response = ref<DebugResponse | null>(null)

const dialogTitle = computed(() =>
  props.api?.name ? `调试 API · ${props.api.name}` : 'API 调试',
)

const prettyBody = computed(() => {
  const body = response.value?.body ?? ''
  if (!body) return '(empty)'
  try {
    return JSON.stringify(JSON.parse(body), null, 2)
  } catch {
    return body
  }
})

const respHeaderRows = computed(() => {
  const h = response.value?.headers || {}
  return Object.entries(h).map(([name, values]) => ({
    name,
    value: (values || []).join(', '),
  }))
})

const respHeaderCount = computed(() => respHeaderRows.value.length)

const bodyPlaceholder = computed(() =>
  bodyType.value === 'json' ? '{\n  "key": "value"\n}' : '请求体内容',
)

function emptyPair(): KvPair {
  return { enabled: true, key: '', value: '' }
}

function onOpened() {
  resetResponse()
  hydrateFromApi()
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) hydrateFromApi()
  },
)

function resetResponse() {
  response.value = null
  errorMsg.value = ''
  usedProxy.value = false
  respTab.value = 'body'
}

function hydrateFromApi() {
  const api = props.api
  if (!api) {
    method.value = 'GET'
    url.value = ''
    params.value = [emptyPair()]
    headers.value = [emptyPair()]
    formFields.value = [emptyPair()]
    bodyType.value = 'none'
    bodyText.value = ''
    authType.value = 'none'
    return
  }

  const methods = (api.access_methods || 'GET')
    .split(/[,;\s]+/)
    .map((s) => s.trim().toUpperCase())
    .filter(Boolean)
  method.value = methods[0] || 'GET'

  const protocol =
    (api.access_protocols || 'http')
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)[0] || 'http'
  const domain = (api.group?.gateway?.domain || '')
    .replace(/^(https?:)?\/\//i, '')
    .replace(/\/$/, '')
  const accessHosts = (api.access_hosts || '')
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
  const host = domain || accessHosts[0] || 'localhost'
  url.value = `${protocol}://${host}${firstAccessPath(api)}`

  const hdrs: KvPair[] = []
  if (domain && accessHosts[0] && accessHosts[0].toLowerCase() !== domain.toLowerCase()) {
    hdrs.push({ enabled: true, key: 'Host', value: accessHosts[0] })
  }
  for (const [name, values] of Object.entries(api.access_headers || {})) {
    if (!name.trim()) continue
    for (const value of values || []) {
      hdrs.push({ enabled: true, key: name, value })
    }
  }
  hdrs.push(emptyPair())
  headers.value = hdrs
  params.value = [emptyPair()]
  formFields.value = [emptyPair()]
  bodyType.value = method.value === 'GET' || method.value === 'HEAD' ? 'none' : 'json'
  bodyText.value = bodyType.value === 'json' ? '{\n  \n}' : ''
  applyAuthFromApi(api, props.consumer || null)
}

function firstAccessPath(api: ApiItem): string {
  const raw =
    (api.access_path || '/')
      .split(/[,;\n]/)
      .map((s) => s.trim())
      .filter(Boolean)[0] || '/'
  let p = raw
  if (p.startsWith('~')) p = p.slice(1)
  if (!p.startsWith('/')) p = `/${p}`
  const prefix = (api.group?.space?.prefix || '').replace(/\/$/, '')
  if (prefix && p !== prefix && !p.startsWith(`${prefix}/`)) {
    p = `${prefix}${p}`
  }
  return p
}

function firstConfigName(value: unknown, fallback: string): string {
  if (Array.isArray(value) && value.length) return String(value[0]).trim() || fallback
  if (typeof value === 'string' && value.trim()) {
    return (
      value
        .split(/[,;\s]+/)
        .map((s) => s.trim())
        .filter(Boolean)[0] || fallback
    )
  }
  return fallback
}

function applyAuthFromApi(api: ApiItem, consumer: ConsumerItem | null) {
  authType.value = 'none'
  authKeyName.value = 'apikey'
  authKeyValue.value = ''
  authKeyIn.value = 'header'
  authUsername.value = ''
  authPassword.value = ''
  authBearer.value = ''
  if (!api.auth_enabled || !api.auth_plugin) return

  const cfg = api.auth_config || {}
  const cred = consumer?.credentials?.find((c) => c.plugin === api.auth_plugin)

  switch (api.auth_plugin) {
    case 'key-auth':
      authType.value = 'apikey'
      authKeyName.value = firstConfigName(cfg.key_names, 'apikey')
      authKeyValue.value = cred?.config?.key || ''
      authKeyIn.value = 'header'
      break
    case 'basic-auth':
      authType.value = 'basic'
      authUsername.value = cred?.config?.username || ''
      authPassword.value = cred?.config?.password || ''
      break
    case 'jwt':
      authType.value = 'bearer'
      authBearer.value = ''
      break
    default:
      authType.value = 'none'
  }
}

function formatJsonBody() {
  try {
    bodyText.value = JSON.stringify(JSON.parse(bodyText.value || '{}'), null, 2)
  } catch {
    ElMessage.warning('JSON 格式不正确')
  }
}

function buildFinalUrl(): string {
  const base = url.value.trim()
  if (!base) throw new Error('请填写请求 URL')
  const u = new URL(base)
  for (const p of params.value) {
    if (!p.enabled || !p.key.trim()) continue
    u.searchParams.append(p.key.trim(), p.value)
  }
  if (authType.value === 'apikey' && authKeyIn.value === 'query' && authKeyName.value.trim()) {
    u.searchParams.set(authKeyName.value.trim(), authKeyValue.value)
  }
  return u.toString()
}

function buildHeaders(): Record<string, string> {
  const out: Record<string, string> = {}
  for (const h of headers.value) {
    if (!h.enabled || !h.key.trim()) continue
    out[h.key.trim()] = h.value
  }
  if (authType.value === 'apikey' && authKeyIn.value === 'header' && authKeyName.value.trim()) {
    out[authKeyName.value.trim()] = authKeyValue.value
  } else if (authType.value === 'basic') {
    out.Authorization = `Basic ${btoa(`${authUsername.value}:${authPassword.value}`)}`
  } else if (authType.value === 'bearer' && authBearer.value.trim()) {
    out.Authorization = `Bearer ${authBearer.value.trim()}`
  }
  if (bodyType.value === 'json' && !hasHeader(out, 'Content-Type')) {
    out['Content-Type'] = 'application/json'
  } else if (bodyType.value === 'x-www-form-urlencoded' && !hasHeader(out, 'Content-Type')) {
    out['Content-Type'] = 'application/x-www-form-urlencoded'
  }
  return out
}

function hasHeader(headersMap: Record<string, string>, name: string) {
  const lower = name.toLowerCase()
  return Object.keys(headersMap).some((k) => k.toLowerCase() === lower)
}

function buildBody(): string | undefined {
  if (method.value === 'GET' || method.value === 'HEAD' || bodyType.value === 'none') return undefined
  if (bodyType.value === 'json' || bodyType.value === 'raw') return bodyText.value
  if (bodyType.value === 'x-www-form-urlencoded') {
    const usp = new URLSearchParams()
    for (const f of formFields.value) {
      if (!f.enabled || !f.key.trim()) continue
      usp.append(f.key.trim(), f.value)
    }
    return usp.toString()
  }
  return undefined
}

function isCorsOrNetworkError(err: unknown): boolean {
  if (!(err instanceof Error)) return false
  const msg = err.message.toLowerCase()
  return (
    err.name === 'TypeError' ||
    msg.includes('failed to fetch') ||
    msg.includes('networkerror') ||
    msg.includes('cors') ||
    msg.includes('network request failed') ||
    msg.includes('load failed')
  )
}

async function sendDirect(finalUrl: string, hdrs: Record<string, string>, body?: string) {
  const start = performance.now()
  const res = await fetch(finalUrl, {
    method: method.value,
    headers: hdrs,
    body: body && method.value !== 'GET' && method.value !== 'HEAD' ? body : undefined,
    credentials: 'omit',
  })
  const durationMs = Math.round(performance.now() - start)
  const buf = await res.arrayBuffer()
  const max = 2 << 20
  const truncated = buf.byteLength > max
  const slice = truncated ? buf.slice(0, max) : buf
  const text = new TextDecoder().decode(slice)
  const respHeaders: Record<string, string[]> = {}
  res.headers.forEach((value, key) => {
    respHeaders[key] = [value]
  })
  return {
    status: res.status,
    statusText: res.statusText,
    headers: respHeaders,
    body: text,
    durationMs,
    size: slice.byteLength,
    truncated,
  }
}

async function sendProxy(finalUrl: string, hdrs: Record<string, string>, body?: string) {
  const data = await proxyDebugRequest({
    method: method.value,
    url: finalUrl,
    headers: hdrs,
    body,
    timeout_ms: 30000,
  })
  return {
    status: data.status,
    statusText: data.status_text,
    headers: data.headers || {},
    body: data.body || '',
    durationMs: data.duration_ms,
    size: data.size,
    truncated: !!data.truncated,
  }
}

async function send() {
  resetResponse()
  let finalUrl = ''
  let hdrs: Record<string, string> = {}
  let body: string | undefined
  try {
    finalUrl = buildFinalUrl()
    hdrs = buildHeaders()
    body = buildBody()
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '请求参数无效'
    return
  }

  sending.value = true
  try {
    const needsHostHeader = Object.keys(hdrs).some((k) => k.toLowerCase() === 'host')
    if (proxyMode.value === 'proxy' || (proxyMode.value === 'auto' && needsHostHeader)) {
      response.value = await sendProxy(finalUrl, hdrs, body)
      usedProxy.value = true
      return
    }
    if (proxyMode.value === 'direct') {
      response.value = await sendDirect(finalUrl, hdrs, body)
      usedProxy.value = false
      return
    }
    try {
      response.value = await sendDirect(finalUrl, hdrs, body)
      usedProxy.value = false
    } catch (err) {
      if (!isCorsOrNetworkError(err)) throw err
      response.value = await sendProxy(finalUrl, hdrs, body)
      usedProxy.value = true
      ElMessage.info('浏览器直连失败（可能是 CORS），已改用服务端代理')
    }
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : '请求失败'
  } finally {
    sending.value = false
  }
}

function statusTagType(status: number) {
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'warning'
  if (status >= 400) return 'danger'
  return 'info'
}

function statusLabel(status: number) {
  if (status >= 200 && status < 300) return 'OK'
  if (status >= 300 && status < 400) return 'Redirect'
  if (status >= 400 && status < 500) return 'Client Error'
  if (status >= 500) return 'Server Error'
  return ''
}

function formatSize(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
</script>

<style scoped>
.debugger {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.req-bar {
  display: flex;
  gap: 8px;
  align-items: center;
}
.method-select {
  width: 120px;
  flex-shrink: 0;
}
.url-input {
  flex: 1;
}
.proxy-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.proxy-label {
  color: #64748b;
  font-size: 13px;
}
.body-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  gap: 8px;
  flex-wrap: wrap;
}
.body-textarea :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
}
.auth-form {
  max-width: 520px;
}
.resp-section {
  border-top: 1px solid #e2e8f0;
  padding-top: 12px;
}
.resp-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}
.resp-title {
  font-weight: 600;
  font-size: 15px;
}
.meta {
  color: #64748b;
  font-size: 13px;
}
.err-alert {
  margin-bottom: 8px;
}
.resp-pre {
  margin: 0;
  padding: 12px;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  max-height: 280px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>

<style>
.api-debugger-dialog .el-dialog__body {
  max-height: 78vh;
  overflow: auto;
  padding-top: 8px;
}
</style>
