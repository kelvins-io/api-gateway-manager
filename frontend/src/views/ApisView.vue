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
      <el-table-column prop="upstream_url" label="上游" min-width="180" />
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
        <el-form-item label="上游地址">
          <el-input v-model="form.upstream_url" placeholder="http://host.docker.internal:9000" />
        </el-form-item>
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
        <el-descriptions-item label="上游地址">{{ detailSnap.upstream_url || '-' }}</el-descriptions-item>
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
import type { ApiItem, ApiVersion } from '@/types'
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

const form = reactive({
  name: '',
  pathList: ['/api/demo'] as string[],
  methodList: ['GET'] as string[],
  upstream_url: '',
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
  upstream_url: string
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
      upstream_url: String(obj.upstream_url || ''),
      strip_path: Boolean(obj.strip_path),
    }
  } catch {
    return null
  }
}

function summarize(snap: Record<string, unknown> | string) {
  const obj = parseSnapshot(snap)
  if (!obj) return String(snap)
  return `${obj.methods} ${obj.path} -> ${obj.upstream_url}`
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
  form.upstream_url = 'http://httpbin.org'
  form.strip_path = true
  visible.value = true
}

function openEdit(row: ApiItem) {
  editing.value = row
  form.name = row.name
  form.pathList = splitPaths(row.path)
  form.methodList = row.methods.split(',').map((s) => s.trim()).filter(Boolean)
  form.upstream_url = row.upstream_url
  form.strip_path = row.strip_path
  visible.value = true
}

async function save() {
  if (!form.name || !form.pathList.length || !form.methodList.length || !form.upstream_url) {
    ElMessage.warning('请填写完整信息')
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
    upstream_url: form.upstream_url,
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

onMounted(load)
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
</style>
