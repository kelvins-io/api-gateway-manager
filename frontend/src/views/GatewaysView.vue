<template>
  <div>
    <template v-if="!allowed">
      <el-result icon="warning" title="无权限" sub-title="网关管理仅系统管理员可访问">
        <template #extra>
          <el-button type="primary" @click="$router.push('/spaces')">返回空间管理</el-button>
        </template>
      </el-result>
    </template>
    <template v-else>
      <el-alert
        title="添加网关时会自动探测 Admin API（Kong /status）是否可用；不可达将拒绝保存。Admin API 创建后不可修改。"
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 16px"
      />
      <div class="toolbar">
        <el-button type="primary" @click="openCreate">添加网关</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe empty-text="暂无网关，请先添加">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="网关名" min-width="140" />
        <el-table-column prop="admin_api" label="Admin API" min-width="220" />
        <el-table-column prop="domain" label="Domain" min-width="180" />
        <el-table-column prop="network_zone" label="网络区域" width="140" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-dialog v-model="visible" :title="editing ? '编辑网关' : '添加网关'" width="520px">
        <el-form :model="form" label-width="100px">
          <el-form-item label="网关名" required>
            <el-input v-model="form.name" placeholder="如：本地 Kong" />
          </el-form-item>
          <el-form-item label="Admin API" required>
            <div class="admin-api-row">
              <el-input v-model="form.admin_api" placeholder="http://localhost:8001" :disabled="!!editing" />
              <el-button :loading="probing" @click="onProbe">探测</el-button>
            </div>
          </el-form-item>
          <el-form-item label="Domain" required>
            <el-input v-model="form.domain" placeholder="10.0.0.1:8000 或 api.example.com:443" />
          </el-form-item>
          <el-form-item label="网络区域" required>
            <el-select v-model="form.network_zone" style="width: 100%" placeholder="请选择网络区域">
              <el-option v-for="z in networkZoneOptions" :key="z" :label="z" :value="z" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="visible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="save">确定</el-button>
        </template>
      </el-dialog>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Gateway } from '@/types'
import * as gatewayApi from '@/api/gateway'
import { useUserStore } from '@/stores/user'

const store = useUserStore()
const { user } = storeToRefs(store)
const allowed = computed(() => user.value?.role === 'system_admin')

const list = ref<Gateway[]>([])
const loading = ref(false)
const saving = ref(false)
const probing = ref(false)
const visible = ref(false)
const editing = ref<Gateway | null>(null)
const networkZoneOptions = ['内网', 'DMZ'] as const
const form = reactive({ name: '', admin_api: '', domain: '', network_zone: '' })

async function load() {
  if (!allowed.value) return
  loading.value = true
  try {
    list.value = (await gatewayApi.listGateways()) || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.admin_api = 'http://localhost:8001'
  form.domain = ''
  form.network_zone = '内网'
  visible.value = true
}

function openEdit(row: Gateway) {
  editing.value = row
  form.name = row.name
  form.admin_api = row.admin_api
  form.domain = row.domain || ''
  form.network_zone = networkZoneOptions.some((z) => z === row.network_zone) ? row.network_zone : ''
  visible.value = true
}

async function onProbe() {
  if (!form.admin_api.trim()) {
    ElMessage.warning('请先填写 Admin API')
    return
  }
  probing.value = true
  try {
    await gatewayApi.probeGateway(form.admin_api.trim())
    ElMessage.success('Admin API 探测成功')
  } finally {
    probing.value = false
  }
}

function isGatewayDomain(raw: string) {
  const value = raw.trim()
  let host = ''
  let portText = ''
  if (value.startsWith('[')) {
    const end = value.indexOf(']:')
    if (end <= 1) return false
    host = value.slice(1, end)
    portText = value.slice(end + 2)
  } else {
    const idx = value.lastIndexOf(':')
    if (idx <= 0 || value.indexOf(':') !== idx) return false
    host = value.slice(0, idx)
    portText = value.slice(idx + 1)
  }
  if (!/^\d+$/.test(portText)) return false
  const port = Number(portText)
  if (port < 1 || port > 65535) return false
  if (/^(?:\d{1,3}\.){3}\d{1,3}$/.test(host)) {
    return host.split('.').every((part) => {
      if (!/^\d{1,3}$/.test(part) || (part.length > 1 && part.startsWith('0'))) return false
      const n = Number(part)
      return n >= 0 && n <= 255
    })
  }
  if (host.includes(':')) return /^[0-9a-fA-F:]+$/.test(host) && host.includes(':')
  if (!/[A-Za-z]/.test(host) || host.length > 253 || host.includes('..')) return false
  return host.split('.').every((label) => /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/.test(label) || /^[A-Za-z0-9]$/.test(label))
}

async function save() {
  form.domain = form.domain.trim()
  if (!form.name || !form.admin_api || !form.domain || !form.network_zone) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (!isGatewayDomain(form.domain)) {
    ElMessage.warning('Domain 须为 IP:端口 或 域名:端口，例如 10.0.0.1:8000、api.example.com:443')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await gatewayApi.updateGateway(editing.value.id, {
        name: form.name,
        domain: form.domain,
        network_zone: form.network_zone,
      })
      ElMessage.success('更新成功')
    } else {
      await gatewayApi.createGateway({ ...form })
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: Gateway) {
  await ElMessageBox.confirm(`确认删除网关「${row.name}」？`, '提示', { type: 'warning' })
  await gatewayApi.deleteGateway(row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(async () => {
  if (!user.value) {
    try {
      await store.fetchMe()
    } catch {
      return
    }
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
.admin-api-row {
  display: flex;
  gap: 8px;
  width: 100%;
}
</style>
