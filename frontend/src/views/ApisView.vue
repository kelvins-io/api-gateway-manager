<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.push('/groups')">返回分组</el-button>
      <el-button type="primary" @click="openCreate">新建 API</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column label="路径" min-width="180">
        <template #default="{ row }">
          <el-tag
            v-for="p in splitPaths(row.path)"
            :key="p"
            size="small"
            style="margin: 2px 4px 2px 0"
          >
            {{ p }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="methods" label="方法" width="120" />
      <el-table-column label="上游" min-width="220">
        <template #default="{ row }">
          {{ upstreamLabel(row) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="current_version" label="当前版本" width="100" />
      <el-table-column label="操作" width="360" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="success" @click="onPublish(row)">发布</el-button>
          <el-button link type="warning" @click="onOffline(row)" :disabled="row.status !== 'published'">下线</el-button>
          <el-button link type="primary" @click="openVersions(row)">版本</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="visible" :title="editing ? '编辑 API' : '新建 API'" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="路径">
          <el-select
            v-model="form.pathList"
            multiple
            filterable
            allow-create
            default-first-option
            style="width: 100%"
            :placeholder="pathPlaceholder"
          />
          <div v-if="spacePrefix" class="path-hint">
            保存后自动拼接空间前缀 {{ spacePrefix }}
            <template v-if="previewPaths.length">，结果：{{ previewPaths.join('、') }}</template>
          </div>
        </el-form-item>
        <el-form-item label="方法">
          <el-select v-model="form.methodList" multiple style="width: 100%">
            <el-option v-for="m in methodOptions" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="接入协议">
          <el-select v-model="form.accessProtocolList" multiple style="width: 100%">
            <el-option v-for="p in protocolOptions" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="后端服务协议">
          <el-select v-model="form.protocol" style="width: 100%">
            <el-option v-for="p in protocolOptions" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="主机">
          <el-radio-group v-model="form.host_kind" style="margin-bottom: 8px">
            <el-radio value="direct">IP / 域名</el-radio>
            <el-radio value="upstream">Upstream</el-radio>
          </el-radio-group>
          <el-input v-if="form.host_kind === 'direct'" v-model="form.host" placeholder="127.0.0.1 或 example.com" />
          <el-select v-else v-model="form.upstream_id" filterable style="width: 100%" placeholder="选择本空间 Upstream">
            <el-option v-for="u in upstreams" :key="u.id" :label="u.name" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="form.port" :min="80" :max="65535" :step="1" />
        </el-form-item>
        <el-form-item label="Service Path">
          <el-input v-model="form.service_path" placeholder="/" />
        </el-form-item>
        <el-collapse>
          <el-collapse-item title="重试与超时" name="advanced">
            <el-form-item label="Retries">
              <el-input-number v-model="form.retries" :min="0" :step="1" />
            </el-form-item>
            <el-form-item label="连接超时">
              <el-input-number v-model="form.connect_timeout" :min="0" :step="1000" />
              <span class="unit">毫秒</span>
            </el-form-item>
            <el-form-item label="写超时">
              <el-input-number v-model="form.write_timeout" :min="0" :step="1000" />
              <span class="unit">毫秒</span>
            </el-form-item>
            <el-form-item label="读超时">
              <el-input-number v-model="form.read_timeout" :min="0" :step="1000" />
              <span class="unit">毫秒</span>
            </el-form-item>
          </el-collapse-item>
        </el-collapse>
        <el-form-item label="Strip Path">
          <el-switch v-model="form.strip_path" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="versionVisible" title="API 版本" width="720px">
      <el-table :data="versions" v-loading="versionLoading">
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column label="发布时间" width="180">
          <template #default="{ row }">{{ formatTime(row.published_at) }}</template>
        </el-table-column>
        <el-table-column label="配置摘要">
          <template #default="{ row }">
            {{ summarize(row.config_snapshot) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button
              link
              type="primary"
              :disabled="row.version === currentApi?.current_version && currentApi?.status === 'published'"
              @click="onSwitch(row.version)"
            >
              切换
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="detailVisible" :title="detailTitle" width="560px" append-to-body>
      <el-descriptions v-if="detailSnap" :column="1" border>
        <el-descriptions-item label="版本">{{ detailVersion?.version }}</el-descriptions-item>
        <el-descriptions-item label="发布时间">{{ formatTime(detailVersion?.published_at || '') }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ detailSnap.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="路径">
          <el-tag
            v-for="p in splitPaths(detailSnap.path)"
            :key="p"
            size="small"
            style="margin: 2px 4px 2px 0"
          >
            {{ p }}
          </el-tag>
          <span v-if="!splitPaths(detailSnap.path).length">-</span>
        </el-descriptions-item>
        <el-descriptions-item label="方法">{{ detailSnap.methods || '-' }}</el-descriptions-item>
        <el-descriptions-item label="接入协议">{{ detailSnap.access_protocols || 'http' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务协议">{{ detailSnap.protocol || '-' }}</el-descriptions-item>
        <el-descriptions-item label="主机">{{ detailSnap.kong_host || detailSnap.host || detailSnap.upstream_url || '-' }}</el-descriptions-item>
        <el-descriptions-item label="端口">{{ detailSnap.port || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Service Path">{{ detailSnap.service_path || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Retries">{{ detailSnap.retries ?? '-' }}</el-descriptions-item>
        <el-descriptions-item label="连接超时">{{ detailSnap.connect_timeout ?? '-' }} ms</el-descriptions-item>
        <el-descriptions-item label="写超时">{{ detailSnap.write_timeout ?? '-' }} ms</el-descriptions-item>
        <el-descriptions-item label="读超时">{{ detailSnap.read_timeout ?? '-' }} ms</el-descriptions-item>
        <el-descriptions-item label="Strip Path">{{ detailSnap.strip_path ? '开启' : '关闭' }}</el-descriptions-item>
      </el-descriptions>
      <el-empty v-else description="无法解析该版本配置" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ApiItem, ApiVersion, UpstreamItem } from '@/types'
import * as apiMod from '@/api/api'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const store = useUserStore()
const gid = Number(route.params.gid)
const spacePrefix = computed(() => (store.currentSpace?.prefix || '').replace(/\/$/, ''))
const list = ref<ApiItem[]>([])
const versions = ref<ApiVersion[]>([])
const loading = ref(false)
const saving = ref(false)
const versionLoading = ref(false)
const visible = ref(false)
const versionVisible = ref(false)
const detailVisible = ref(false)
const editing = ref<ApiItem | null>(null)
const currentApi = ref<ApiItem | null>(null)
const detailVersion = ref<ApiVersion | null>(null)
const methodOptions = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']
const protocolOptions = ['http', 'https', 'grpc', 'grpcs']
const upstreams = ref<UpstreamItem[]>([])

const form = reactive({
  name: '',
  pathList: ['/api/demo'] as string[],
  methodList: ['GET'] as string[],
  accessProtocolList: ['http'] as string[],
  protocol: 'http',
  host_kind: 'direct',
  host: '',
  upstream_id: undefined as number | undefined,
  port: 80,
  service_path: '/',
  retries: 5,
  connect_timeout: 60000,
  write_timeout: 60000,
  read_timeout: 60000,
  strip_path: true,
})

const pathPlaceholder = computed(() =>
  spacePrefix.value
    ? `输入相对路径后回车，将自动加上 ${spacePrefix.value}`
    : '输入路径后回车，可添加多个，如 /api/v1 /api/v2',
)

const previewPaths = computed(() =>
  form.pathList
    .map((p) => normalizePath(p))
    .filter(Boolean)
    .map((p) => applyPrefix(p)),
)

function normalizePath(raw: string) {
  const t = raw.trim()
  if (!t) return ''
  return t.startsWith('/') ? t : `/${t}`
}

function applyPrefix(path: string) {
  const prefix = spacePrefix.value
  if (!prefix || path === prefix || path.startsWith(`${prefix}/`)) return path
  return `${prefix}${path}`
}

function splitPaths(raw: string): string[] {
  return (raw || '')
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function statusLabel(s: string) {
  return ({ draft: '草稿', published: '已发布', offline: '已下线' } as Record<string, string>)[s] || s
}
function statusType(s: string) {
  return ({ draft: 'info', published: 'success', offline: 'warning' } as Record<string, string>)[s] || 'info'
}

interface VersionSnapshot {
  name: string
  path: string
  methods: string
  access_protocols: string
  upstream_url: string
  protocol: string
  host: string
  kong_host: string
  port: number
  service_path: string
  retries: number
  connect_timeout: number
  write_timeout: number
  read_timeout: number
  strip_path: boolean
}

const detailSnap = computed(() =>
  detailVersion.value ? parseSnapshot(detailVersion.value.config_snapshot) : null,
)
const detailTitle = computed(() =>
  detailVersion.value ? `版本 ${detailVersion.value.version} 详情` : '版本详情',
)

function parseSnapshot(snap: Record<string, unknown> | string): VersionSnapshot | null {
  try {
    const obj = (typeof snap === 'string' ? JSON.parse(snap) : snap) as Record<string, unknown>
    return {
      name: String(obj.name || ''),
      path: String(obj.path || ''),
      methods: String(obj.methods || ''),
      access_protocols: String(obj.access_protocols || ''),
      upstream_url: String(obj.upstream_url || ''),
      protocol: String(obj.protocol || ''),
      host: String(obj.host || ''),
      kong_host: String(obj.kong_host || ''),
      port: Number(obj.port || 0),
      service_path: String(obj.service_path || ''),
      retries: Number(obj.retries ?? 0),
      connect_timeout: Number(obj.connect_timeout ?? 0),
      write_timeout: Number(obj.write_timeout ?? 0),
      read_timeout: Number(obj.read_timeout ?? 0),
      strip_path: Boolean(obj.strip_path),
    }
  } catch {
    return null
  }
}

function summarize(snap: Record<string, unknown> | string) {
  const obj = parseSnapshot(snap)
  if (!obj) return String(snap)
  const dest = obj.kong_host || obj.host || obj.upstream_url
  return `${obj.methods} ${obj.path} -> ${obj.protocol}://${dest}:${obj.port}${obj.service_path}`
}

function upstreamLabel(row: ApiItem) {
  if (row.host) {
    return `${row.protocol || 'http'}://${row.host}:${row.port}${row.service_path || '/'}`
  }
  return row.upstream_url || '-'
}

function resetService() {
  form.protocol = 'http'
  form.host_kind = 'direct'
  form.host = 'httpbin.org'
  form.upstream_id = undefined
  form.port = 80
  form.service_path = '/'
  form.retries = 5
  form.connect_timeout = 60000
  form.write_timeout = 60000
  form.read_timeout = 60000
}

function formatTime(raw: string) {
  if (!raw) return '-'
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return raw
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

function openDetail(row: ApiVersion) {
  detailVersion.value = row
  detailVisible.value = true
}

async function load() {
  loading.value = true
  try {
    list.value = (await apiMod.listApis(gid)) || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.pathList = ['/api/demo']
  form.methodList = ['GET']
  form.accessProtocolList = ['http']
  resetService()
  form.strip_path = true
  visible.value = true
}

function openEdit(row: ApiItem) {
  editing.value = row
  form.name = row.name
  form.pathList = splitPaths(row.path)
  form.methodList = row.methods.split(',').map((s) => s.trim()).filter(Boolean)
  form.accessProtocolList = (row.access_protocols || 'http').split(',').map((s) => s.trim()).filter(Boolean)
  form.protocol = row.protocol || 'http'
  form.host_kind = row.host_kind || 'direct'
  form.host = row.host || ''
  form.upstream_id = row.upstream_id
  form.port = row.port || 80
  form.service_path = row.service_path || '/'
  form.retries = row.retries ?? 5
  form.connect_timeout = row.connect_timeout ?? 60000
  form.write_timeout = row.write_timeout ?? 60000
  form.read_timeout = row.read_timeout ?? 60000
  form.strip_path = row.strip_path
  visible.value = true
}

async function save() {
  if (!form.name || !form.pathList.length || !form.methodList.length || !form.accessProtocolList.length) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (form.host_kind === 'direct' && !form.host.trim()) {
    ElMessage.warning('请填写主机')
    return
  }
  if (form.host_kind === 'upstream' && !form.upstream_id) {
    ElMessage.warning('请选择 Upstream')
    return
  }
  if (!form.service_path.startsWith('/') || /[,;\n]/.test(form.service_path)) {
    ElMessage.warning('Service Path 只能填一个，且以 / 开头')
    return
  }
  const paths = form.pathList.map((p) => normalizePath(p)).filter(Boolean)
  if (!paths.length) {
    ElMessage.warning('请至少添加一个路径')
    return
  }
  const payload = {
    name: form.name,
    path: paths.join(','),
    methods: form.methodList.join(','),
    access_protocols: form.accessProtocolList.join(','),
    protocol: form.protocol,
    host_kind: form.host_kind,
    host: form.host_kind === 'direct' ? form.host.trim() : '',
    upstream_id: form.host_kind === 'upstream' ? form.upstream_id : undefined,
    port: form.port,
    service_path: form.service_path,
    retries: form.retries,
    connect_timeout: form.connect_timeout,
    write_timeout: form.write_timeout,
    read_timeout: form.read_timeout,
    strip_path: form.strip_path,
  }
  saving.value = true
  try {
    if (editing.value) {
      await apiMod.updateApi(editing.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await apiMod.createApi(gid, payload)
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onPublish(row: ApiItem) {
  await ElMessageBox.confirm(`确认发布 API「${row.name}」到 Kong？`, '发布确认')
  await apiMod.publishApi(row.id)
  ElMessage.success('发布成功，已生成新版本')
  await load()
}

async function onOffline(row: ApiItem) {
  await ElMessageBox.confirm(`确认下线 API「${row.name}」？`, '下线确认', { type: 'warning' })
  await apiMod.offlineApi(row.id)
  ElMessage.success('已下线')
  await load()
}

async function onDelete(row: ApiItem) {
  await ElMessageBox.confirm(`确认删除 API「${row.name}」？`, '提示', { type: 'warning' })
  await apiMod.deleteApi(row.id)
  ElMessage.success('已删除')
  await load()
}

async function openVersions(row: ApiItem) {
  currentApi.value = row
  versionVisible.value = true
  versionLoading.value = true
  try {
    versions.value = (await apiMod.listVersions(row.id)) || []
  } finally {
    versionLoading.value = false
  }
}

async function onSwitch(version: string) {
  if (!currentApi.value) return
  await ElMessageBox.confirm(`切换到版本 ${version}？将下线当前配置并重新发布。`, '版本切换')
  await apiMod.switchVersion(currentApi.value.id, version)
  ElMessage.success('切换成功')
  versionVisible.value = false
  await load()
}

onMounted(async () => {
  if (store.currentSpaceId) {
    upstreams.value = (await apiMod.listUpstreams(store.currentSpaceId)) || []
  }
  await load()
})
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  gap: 8px;
}
.path-hint {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
.unit {
  margin-left: 8px;
  color: #909399;
}
</style>
