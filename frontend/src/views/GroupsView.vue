<template>
  <div>
    <el-alert
      v-if="!store.currentSpaceId"
      title="请先在左侧选择或创建一个空间"
      type="warning"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    />
    <div class="toolbar">
      <el-button type="primary" :disabled="!store.currentSpaceId" @click="openCreate">新建分组</el-button>
      <el-button :disabled="!store.currentSpaceId" @click="$router.push('/upstreams')">Upstream</el-button>
      <el-button :disabled="!store.currentSpaceId" @click="$router.push('/consumers')">Consumers</el-button>
      <el-button :disabled="!store.currentSpaceId" @click="$router.push('/plugins')">Plugins</el-button>
      <el-button :disabled="!store.currentSpaceId" @click="load">刷新</el-button>
      <span v-if="store.currentSpace" class="hint">
        当前空间：{{ store.currentSpace.name }}
        <template v-if="store.currentSpace.prefix">（前缀 {{ store.currentSpace.prefix }}）</template>
      </span>
    </div>

    <el-table :data="paged" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="分组名">
        <template #default="{ row }">
          <el-button link type="primary" @click="$router.push(`/groups/${row.id}/apis`)">
            {{ row.name }}
          </el-button>
        </template>
      </el-table-column>
      <el-table-column label="所属网关">
        <template #default="{ row }">
          {{ row.gateway?.name || row.gateway_id }}
          <el-tag size="small" style="margin-left: 8px">{{ row.gateway?.network_zone }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-tooltip :disabled="!row.api_count" content="分组下仍有 API，不能删除" placement="top">
            <span>
              <el-button link type="danger" @click="onDelete(row)" :disabled="!!row.api_count">删除</el-button>
            </span>
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>
    <ListPagination
      v-model:page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="pageSizes"
    />

    <el-dialog v-model="visible" :title="editing ? '编辑分组' : '新建分组'" width="480px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="分组名">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="所属网关">
          <el-select v-model="form.gateway_id" style="width: 100%" placeholder="选择网关" :disabled="!!editing">
            <el-option
              v-for="g in gateways"
              :key="g.id"
              :label="`${g.name} (${g.network_zone})`"
              :value="g.id"
            />
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
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ApiGroup, Gateway } from '@/types'
import * as apiMod from '@/api/api'
import * as gatewayApi from '@/api/gateway'
import { useUserStore } from '@/stores/user'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const store = useUserStore()
const list = ref<ApiGroup[]>([])
const { page, pageSize, total, paged, pageSizes, resetPage } = usePagination(list)
const gateways = ref<Pick<Gateway, 'id' | 'name' | 'network_zone'>[]>([])
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editing = ref<ApiGroup | null>(null)
const form = reactive({ name: '', gateway_id: undefined as number | undefined })

async function load() {
  if (!store.currentSpaceId) {
    list.value = []
    return
  }
  loading.value = true
  try {
    list.value = (await apiMod.listGroups(store.currentSpaceId)) || []
  } finally {
    loading.value = false
  }
}

async function loadGateways() {
  gateways.value = (await gatewayApi.listGatewayOptions()) || []
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.gateway_id = gateways.value[0]?.id
  visible.value = true
}

function openEdit(row: ApiGroup) {
  editing.value = row
  form.name = row.name
  form.gateway_id = row.gateway_id
  visible.value = true
}

async function save() {
  if (!form.name || !form.gateway_id || !store.currentSpaceId) {
    ElMessage.warning('请填写完整信息')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await apiMod.updateGroup(editing.value.id, { name: form.name })
      ElMessage.success('更新成功')
    } else {
      await apiMod.createGroup(store.currentSpaceId, { name: form.name, gateway_id: form.gateway_id })
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: ApiGroup) {
  await ElMessageBox.confirm(`确认删除分组「${row.name}」？`, '提示', { type: 'warning' })
  await apiMod.deleteGroup(row.id)
  ElMessage.success('已删除')
  await load()
}

watch(() => store.currentSpaceId, () => {
  resetPage()
  load()
})

onMounted(async () => {
  await loadGateways()
  await load()
})
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.hint {
  margin-left: 8px;
  color: #909399;
}
</style>
