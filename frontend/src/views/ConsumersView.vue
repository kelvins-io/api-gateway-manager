<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.push('/groups')">返回分组</el-button>
      <el-button type="primary" :disabled="!store.currentSpaceId" @click="openCreate">新建 Consumer</el-button>
      <el-button :disabled="!store.currentSpaceId" @click="onSync">同步到网关</el-button>
      <el-button @click="load">刷新</el-button>
      <span v-if="store.currentSpace" class="hint">当前空间：{{ store.currentSpace.name }}</span>
    </div>

    <el-table :data="paged" v-loading="loading" stripe>
      <el-table-column prop="username" label="用户名" />
      <el-table-column label="关联 API" width="100">
        <template #default="{ row }">
          <el-button
            v-if="(row.apis || []).length"
            link
            type="primary"
            @click="openApis(row)"
          >
            {{ (row.apis || []).length }}
          </el-button>
          <span v-else>0</span>
        </template>
      </el-table-column>
      <el-table-column label="凭证">
        <template #default="{ row }">
          <el-tag v-for="(c, i) in row.credentials || []" :key="i" size="small" style="margin-right: 4px">
            {{ c.plugin }}
          </el-tag>
          <span v-if="!(row.credentials || []).length">-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <ListPagination
      v-model:page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="pageSizes"
    />

    <el-dialog v-model="apiVisible" :title="apiTitle" width="1100px">
      <el-table :data="apiDialogPaged" empty-text="暂无关联 API" max-height="420" stripe>
        <el-table-column label="所属空间" min-width="110">
          <template #default="{ row }">{{ row.group?.space?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="API分组" min-width="110">
          <template #default="{ row }">{{ row.group?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="接入协议" width="110">
          <template #default="{ row }">
            <el-tag
              v-for="proto in protocolsOf(row)"
              :key="proto"
              size="small"
              style="margin: 2px 4px 2px 0"
            >
              {{ proto }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="完整请求路径" min-width="200">
          <template #default="{ row }">
            <div v-for="p in splitPaths(row.access_path)" :key="p" class="access-url">
              {{ fullAccessPath(row, p) }}
            </div>
            <span v-if="!splitPaths(row.access_path).length">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="access_methods" label="请求方法" width="110" />
        <el-table-column label="接入 Hosts" min-width="140">
          <template #default="{ row }">{{ row.access_hosts || '-' }}</template>
        </el-table-column>
        <el-table-column label="接入 Headers" min-width="180">
          <template #default="{ row }">{{ formatHeaders(row.access_headers) }}</template>
        </el-table-column>
      </el-table>
      <div class="dialog-pagination">
        <el-pagination
          :current-page="apiDialogPage"
          :page-size="apiDialogPageSize"
          :total="apiDialogList.length"
          :page-sizes="[5, 10, 20]"
          layout="total, sizes, prev, pager, next"
          small
          background
          @update:current-page="apiDialogPage = $event"
          @update:page-size="onApiDialogPageSize"
        />
      </div>
    </el-dialog>

    <el-dialog v-model="visible" :title="editing ? '编辑 Consumer' : '新建 Consumer'" width="720px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="字母、数字、点、下划线或中划线" />
        </el-form-item>
        <el-form-item label="Custom ID">
          <el-input v-model="form.custom_id" placeholder="可选" />
        </el-form-item>
        <el-form-item label="凭证">
          <div v-for="(c, i) in form.credentials" :key="i" class="cred">
            <div class="cred-head">
              <el-select v-model="c.plugin" style="width: 180px" :disabled="!!editing" @change="onPlugin(c)">
                <el-option v-for="p in plugins" :key="p" :label="p" :value="p" />
              </el-select>
              <el-button v-if="!editing" link type="danger" @click="form.credentials.splice(i, 1)">删除</el-button>
            </div>
            <el-input v-if="c.plugin === 'key-auth'" v-model="c.config.key" placeholder="key，留空则自动生成" />
            <template v-else-if="c.plugin === 'basic-auth'">
              <el-input v-model="c.config.username" placeholder="用户名" />
              <el-input v-model="c.config.password" placeholder="密码" show-password />
            </template>
            <template v-else-if="c.plugin === 'jwt'">
              <el-select v-model="c.config.algorithm" placeholder="算法" style="width: 100%">
                <el-option v-for="a in jwtAlgorithms" :key="a" :label="a" :value="a" />
              </el-select>
              <el-input v-model="c.config.key" placeholder="key / iss，留空则自动生成" />
              <el-input v-if="c.config.algorithm.startsWith('HS')" v-model="c.config.secret" placeholder="secret，留空则自动生成" />
              <el-input
                v-else
                v-model="c.config.rsa_public_key"
                type="textarea"
                :rows="3"
                placeholder="RSA 公钥"
              />
            </template>
            <template v-else-if="c.plugin === 'hmac-auth'">
              <el-input v-model="c.config.username" placeholder="用户名" />
              <el-input v-model="c.config.secret" placeholder="secret，留空则自动生成" />
            </template>
            <el-input v-else-if="c.plugin === 'acl'" v-model="c.config.group" placeholder="ACL 分组名" />
          </div>
          <el-button v-if="!editing" link type="primary" @click="form.credentials.push(emptyCred())">添加凭证</el-button>
        </el-form-item>
        <el-form-item label="关联 API">
          <el-select v-model="form.api_ids" multiple filterable style="width: 100%" placeholder="仅可选择已启用相同认证的 API">
            <el-option v-for="a in matchingApis" :key="a.id" :label="`${a.name} (${a.auth_plugin})`" :value="a.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ApiItem, ConsumerCredential, ConsumerItem } from '@/types'
import * as apiMod from '@/api/api'
import { useUserStore } from '@/stores/user'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const plugins = ['key-auth', 'basic-auth', 'jwt', 'hmac-auth', 'acl']
const jwtAlgorithms = ['HS256', 'HS384', 'HS512', 'RS256', 'RS384', 'RS512', 'ES256', 'ES384']

const store = useUserStore()
const list = ref<ConsumerItem[]>([])
const { page, pageSize, total, paged, pageSizes } = usePagination(list)
const apis = ref<ApiItem[]>([])
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editing = ref<ConsumerItem | null>(null)

const apiVisible = ref(false)
const apiTitle = ref('关联 API')
const apiDialogList = ref<ApiItem[]>([])
const apiDialogPage = ref(1)
const apiDialogPageSize = ref(5)
const apiDialogPaged = computed(() => {
  const start = (apiDialogPage.value - 1) * apiDialogPageSize.value
  return apiDialogList.value.slice(start, start + apiDialogPageSize.value)
})

function openApis(row: ConsumerItem) {
  apiTitle.value = `${row.username} 关联的 API`
  apiDialogList.value = row.apis || []
  apiDialogPage.value = 1
  apiVisible.value = true
}

function onApiDialogPageSize(size: number) {
  apiDialogPageSize.value = size
  apiDialogPage.value = 1
}

function protocolsOf(row: ApiItem): string[] {
  const items = (row.access_protocols || 'http')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  return items.length ? items : ['http']
}

function splitPaths(raw: string): string[] {
  return (raw || '')
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function fullAccessPath(row: ApiItem, path: string): string {
  let p = path.startsWith('/') ? path : `/${path}`
  const prefix = (row.group?.space?.prefix || '').replace(/\/$/, '')
  if (prefix && p !== prefix && !p.startsWith(`${prefix}/`)) {
    p = `${prefix}${p}`
  }
  return `${row.group?.gateway?.domain || ''}${p}`
}

function formatHeaders(headers?: Record<string, string[]>) {
  const entries = Object.entries(headers || {}).filter(([name]) => name)
  if (!entries.length) return '-'
  return entries.map(([name, values]) => `${name}: ${(values || []).join(',')}`).join('；')
}

function emptyConfig() {
  return { key: '', username: '', password: '', secret: '', algorithm: 'HS256', rsa_public_key: '', group: '' }
}

function emptyCred(): ConsumerCredential {
  return { plugin: 'key-auth', config: emptyConfig() }
}

const form = reactive({
  username: '',
  custom_id: '',
  credentials: [] as ConsumerCredential[],
  api_ids: [] as number[],
})

const matchingApis = computed(() => {
  const kinds = new Set(form.credentials.map((c) => c.plugin))
  return apis.value.filter((a) => a.auth_enabled && kinds.has(a.auth_plugin))
})

function onPlugin(c: ConsumerCredential) {
  const algorithm = c.config.algorithm || 'HS256'
  c.config = { ...emptyConfig(), algorithm }
}

async function load() {
  if (!store.currentSpaceId) return
  loading.value = true
  try {
    list.value = (await apiMod.listConsumers(store.currentSpaceId)) || []
    apis.value = (await apiMod.listSpaceApis(store.currentSpaceId)) || []
  } finally {
    loading.value = false
  }
}

function fill(row?: ConsumerItem) {
  form.username = row?.username || ''
  form.custom_id = row?.custom_id || ''
  form.credentials = (row?.credentials || [])
    .filter((c) => !(c.plugin === 'acl' && /^G-\d+$/.test(c.config?.group || '')))
    .map((c) => ({
      plugin: c.plugin,
      config: { ...emptyConfig(), ...(c.config || {}) },
    }))
  form.api_ids = (row?.apis || []).map((a) => a.id)
}

function openCreate() {
  editing.value = null
  fill()
  visible.value = true
}

function openEdit(row: ConsumerItem) {
  editing.value = row
  fill(row)
  visible.value = true
}

async function save() {
  if (!store.currentSpaceId || !form.username.trim()) {
    ElMessage.warning('请填写用户名')
    return
  }
  saving.value = true
  try {
    const payload = {
      username: form.username.trim(),
      custom_id: form.custom_id.trim(),
      credentials: form.credentials.map((c) => ({ plugin: c.plugin, config: { ...c.config } })),
      api_ids: form.api_ids.filter((id) => matchingApis.value.some((a) => a.id === id)),
    }
    if (editing.value) {
      await apiMod.updateConsumer(store.currentSpaceId, editing.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await apiMod.createConsumer(store.currentSpaceId, payload)
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: ConsumerItem) {
  if (!store.currentSpaceId) return
  await ElMessageBox.confirm(`确认删除 Consumer「${row.username}」？`, '提示', { type: 'warning' })
  await apiMod.deleteConsumer(store.currentSpaceId, row.id)
  ElMessage.success('已删除')
  await load()
}

async function onSync() {
  if (!store.currentSpaceId) return
  loading.value = true
  try {
    await apiMod.syncConsumers(store.currentSpaceId)
    ElMessage.success('已同步到空间绑定的网关')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  gap: 8px;
  align-items: center;
}
.hint {
  color: #909399;
}
.cred {
  width: 100%;
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cred-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.access-url {
  line-height: 24px;
  word-break: break-all;
}
.dialog-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
