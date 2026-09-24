<template>
  <div>
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">申请空间</el-button>
      <el-button @click="openJoin">加入空间</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <el-table :data="paged" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称">
        <template #default="{ row }">
          <el-button link type="primary" @click="enterSpace(row)">{{ row.name }}</el-button>
        </template>
      </el-table-column>
      <el-table-column prop="prefix" label="路径前缀" width="160" />
      <el-table-column prop="description" label="描述" />
      <el-table-column prop="owner_id" label="所有者 ID" width="120" />
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button link type="primary" @click="$router.push(`/spaces/${row.id}/members`)">成员</el-button>
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

    <el-dialog v-model="createVisible" :title="editing ? '编辑空间' : '申请空间'" width="480px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="路径前缀">
          <el-input
            v-model="form.prefix"
            :disabled="!!editing"
            placeholder="如 /order，创建后不可修改"
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="joinVisible" title="加入空间" width="480px">
      <el-table :data="available" v-loading="joinLoading" height="320">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="prefix" label="路径前缀" width="140" />
        <el-table-column prop="description" label="描述" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="onJoin(row)">加入</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Space } from '@/types'
import * as spaceApi from '@/api/space'
import { useUserStore } from '@/stores/user'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const store = useUserStore()
const router = useRouter()
const list = ref<Space[]>([])
const { page, pageSize, total, paged, pageSizes } = usePagination(list)
const available = ref<Space[]>([])
const loading = ref(false)
const joinLoading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const joinVisible = ref(false)
const editing = ref<Space | null>(null)
const form = reactive({ name: '', description: '', prefix: '' })

async function load() {
  loading.value = true
  try {
    await store.loadSpaces()
    list.value = store.spaces
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.description = ''
  form.prefix = ''
  createVisible.value = true
}

function openEdit(row: Space) {
  editing.value = row
  form.name = row.name
  form.description = row.description
  form.prefix = row.prefix || ''
  createVisible.value = true
}

async function save() {
  if (!form.name.trim()) {
    ElMessage.warning('请输入名称')
    return
  }
  if (!editing.value && !form.prefix.trim()) {
    ElMessage.warning('请输入路径前缀')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await spaceApi.updateSpace(editing.value.id, { name: form.name, description: form.description })
      ElMessage.success('更新成功')
    } else {
      const space = await spaceApi.createSpace({ ...form })
      store.setCurrentSpace(space.id)
      ElMessage.success('空间创建成功')
    }
    createVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: Space) {
  await ElMessageBox.confirm(`确认删除空间「${row.name}」？`, '提示', { type: 'warning' })
  await spaceApi.deleteSpace(row.id)
  ElMessage.success('已删除')
  await load()
}

function enterSpace(row: Space) {
  store.setCurrentSpace(row.id)
  router.push('/groups')
}

async function openJoin() {
  joinVisible.value = true
  joinLoading.value = true
  try {
    available.value = (await spaceApi.listAvailableSpaces()) || []
  } finally {
    joinLoading.value = false
  }
}

async function onJoin(row: Space) {
  await spaceApi.joinSpace(row.id)
  ElMessage.success('加入成功')
  joinVisible.value = false
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
</style>
