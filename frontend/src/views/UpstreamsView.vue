<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.push('/groups')">返回分组</el-button>
      <el-button type="primary" :disabled="!store.currentSpaceId" @click="openCreate">新建 Upstream</el-button>
      <el-input
        v-model="nameQuery"
        clearable
        placeholder="按名称搜索"
        style="width: 220px"
        :disabled="!store.currentSpaceId"
        @keyup.enter="applyFilter"
      />
      <el-button type="primary" :disabled="!store.currentSpaceId" @click="applyFilter">搜索</el-button>
      <el-button @click="load">刷新</el-button>
      <span v-if="store.currentSpace" class="hint">当前空间：{{ store.currentSpace.name }}</span>
    </div>

    <el-table :data="paged" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="algorithm" label="算法" width="180" />
      <el-table-column label="Target">
        <template #default="{ row }">
          <el-tag v-for="t in row.targets || []" :key="t.target" size="small" style="margin-right: 4px">
            {{ t.target }} ({{ t.weight }})
          </el-tag>
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

    <el-dialog v-model="visible" :title="editing ? '编辑 Upstream' : '新建 Upstream'" width="680px">
      <el-form :model="form" label-width="140px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="字母、数字、下划线或中划线" />
        </el-form-item>
        <el-form-item label="算法">
          <el-select v-model="form.algorithm" style="width: 100%">
            <el-option label="round-robin" value="round-robin" />
            <el-option label="least-connections" value="least-connections" />
            <el-option label="consistent-hashing" value="consistent-hashing" />
            <el-option label="latency" value="latency" />
          </el-select>
        </el-form-item>
        <el-form-item label="Slots">
          <el-input-number v-model="form.slots" :min="10" :max="65536" />
        </el-form-item>
        <template v-if="form.algorithm === 'consistent-hashing'">
          <el-form-item label="Hash On">
            <el-select v-model="form.hash_on" style="width: 100%">
              <el-option v-for="h in hashOptions" :key="h" :label="h" :value="h" />
            </el-select>
          </el-form-item>
          <el-form-item label="Hash Fallback">
            <el-select v-model="form.hash_fallback" clearable style="width: 100%">
              <el-option v-for="h in hashOptions" :key="h" :label="h" :value="h" />
            </el-select>
          </el-form-item>
          <el-form-item label="Hash Header">
            <el-input v-model="form.hash_on_header" />
          </el-form-item>
          <el-form-item label="Fallback Header">
            <el-input v-model="form.hash_fallback_header" />
          </el-form-item>
          <el-form-item label="Hash Cookie">
            <el-input v-model="form.hash_on_cookie" />
          </el-form-item>
          <el-form-item label="Cookie Path">
            <el-input v-model="form.hash_on_cookie_path" />
          </el-form-item>
          <el-form-item label="Query Arg">
            <el-input v-model="form.hash_on_query_arg" />
          </el-form-item>
          <el-form-item label="Fallback Query">
            <el-input v-model="form.hash_fallback_query_arg" />
          </el-form-item>
          <el-form-item label="URI Capture">
            <el-input v-model="form.hash_on_uri_capture" />
          </el-form-item>
          <el-form-item label="Fallback URI">
            <el-input v-model="form.hash_fallback_uri_capture" />
          </el-form-item>
        </template>
        <el-form-item label="Target">
          <div v-for="(t, i) in form.targets" :key="i" class="target-row">
            <el-input v-model="t.target" placeholder="host:port" />
            <el-input-number v-model="t.weight" :min="0" :max="65535" />
            <el-button link type="danger" @click="form.targets.splice(i, 1)">删除</el-button>
          </div>
          <el-button link type="primary" @click="form.targets.push({ target: '', weight: 100 })">添加 Target</el-button>
        </el-form-item>
        <el-collapse>
          <el-collapse-item title="健康检查" name="health">
            <el-form-item label="阈值">
              <el-input-number v-model="form.healthchecks.threshold" :min="0" :max="100" :step="1" />
            </el-form-item>
            <div class="side-title">主动检查</div>
            <health-fields v-model="form.healthchecks.active" active />
            <div class="side-title">被动检查</div>
            <health-fields v-model="form.healthchecks.passive" />
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { UpstreamHealthSide, UpstreamItem } from '@/types'
import * as apiMod from '@/api/api'
import { useUserStore } from '@/stores/user'
import HealthFields from '@/components/HealthFields.vue'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const store = useUserStore()
const list = ref<UpstreamItem[]>([])
const nameQuery = ref('')
const filtered = computed(() => {
  const name = nameQuery.value.trim().toLowerCase()
  if (!name) return list.value
  return list.value.filter((row) => row.name.toLowerCase().includes(name))
})
const { page, pageSize, total, paged, pageSizes, resetPage } = usePagination(filtered)
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editing = ref<UpstreamItem | null>(null)
const hashOptions = ['none', 'consumer', 'ip', 'header', 'cookie', 'path', 'query_arg', 'uri_capture']

watch(nameQuery, () => {
  resetPage()
})

function applyFilter() {
  resetPage()
}

function emptyActive(): UpstreamHealthSide {
  return {
    type: 'http',
    http_path: '/',
    timeout: 1,
    concurrency: 10,
    https_verify_certificate: true,
    healthy_interval: 0,
    healthy_successes: 0,
    healthy_http_statuses: [200, 302],
    unhealthy_interval: 0,
    unhealthy_http_failures: 0,
    unhealthy_tcp_failures: 0,
    unhealthy_timeouts: 0,
    unhealthy_http_statuses: [429, 404, 500, 501, 502, 503, 504, 505],
  }
}

function emptyPassive(): UpstreamHealthSide {
  return {
    type: 'http',
    healthy_successes: 0,
    healthy_http_statuses: [200, 201, 202, 203, 204, 205, 206, 207, 208, 226, 300, 301, 302, 303, 304, 305, 306, 307, 308],
    unhealthy_http_failures: 0,
    unhealthy_tcp_failures: 0,
    unhealthy_timeouts: 0,
    unhealthy_http_statuses: [429, 500, 503],
  }
}

const form = reactive({
  name: '',
  algorithm: 'round-robin',
  slots: 10000,
  hash_on: 'none',
  hash_fallback: 'none',
  hash_on_header: '',
  hash_fallback_header: '',
  hash_on_cookie: '',
  hash_on_cookie_path: '/',
  hash_on_query_arg: '',
  hash_fallback_query_arg: '',
  hash_on_uri_capture: '',
  hash_fallback_uri_capture: '',
  targets: [] as { target: string; weight: number }[],
  healthchecks: {
    threshold: 0,
    active: emptyActive(),
    passive: emptyPassive(),
  },
})

async function load() {
  if (!store.currentSpaceId) return
  loading.value = true
  try {
    list.value = (await apiMod.listUpstreams(store.currentSpaceId)) || []
  } finally {
    loading.value = false
  }
}

function fill(row?: UpstreamItem) {
  form.name = row?.name || ''
  form.algorithm = row?.algorithm || 'round-robin'
  form.slots = row?.slots || 10000
  form.hash_on = row?.hash_on || 'none'
  form.hash_fallback = row?.hash_fallback || 'none'
  form.hash_on_header = row?.hash_on_header || ''
  form.hash_fallback_header = row?.hash_fallback_header || ''
  form.hash_on_cookie = row?.hash_on_cookie || ''
  form.hash_on_cookie_path = row?.hash_on_cookie_path || '/'
  form.hash_on_query_arg = row?.hash_on_query_arg || ''
  form.hash_fallback_query_arg = row?.hash_fallback_query_arg || ''
  form.hash_on_uri_capture = row?.hash_on_uri_capture || ''
  form.hash_fallback_uri_capture = row?.hash_fallback_uri_capture || ''
  form.targets = (row?.targets || []).map((t) => ({ target: t.target, weight: t.weight }))
  form.healthchecks.threshold = row?.healthchecks?.threshold || 0
  form.healthchecks.active = { ...emptyActive(), ...(row?.healthchecks?.active || {}) }
  form.healthchecks.passive = { ...emptyPassive(), ...(row?.healthchecks?.passive || {}) }
}

function openCreate() {
  editing.value = null
  fill()
  visible.value = true
}

function openEdit(row: UpstreamItem) {
  editing.value = row
  fill(row)
  visible.value = true
}

async function save() {
  if (!store.currentSpaceId || !form.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  const payload = {
    ...form,
    targets: form.targets.filter((t) => t.target.trim()),
  }
  saving.value = true
  try {
    if (editing.value) {
      await apiMod.updateUpstream(store.currentSpaceId, editing.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await apiMod.createUpstream(store.currentSpaceId, payload)
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: UpstreamItem) {
  if (!store.currentSpaceId) return
  await ElMessageBox.confirm(`确认删除 Upstream「${row.name}」？`, '提示', { type: 'warning' })
  await apiMod.deleteUpstream(store.currentSpaceId, row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.hint {
  color: #909399;
}
.target-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  width: 100%;
}
.side-title {
  margin: 8px 0;
  font-weight: 600;
}
</style>
