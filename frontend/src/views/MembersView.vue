<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.back()">返回</el-button>
      <el-button v-if="canManage" type="primary" @click="openAdd">添加成员</el-button>
      <el-button @click="load">刷新</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="user_id" label="用户 ID" width="100" />
      <el-table-column label="用户名">
        <template #default="{ row }">{{ row.user?.username || '-' }}</template>
      </el-table-column>
      <el-table-column label="系统角色" width="140">
        <template #default="{ row }">{{ roleLabel(row.user?.role) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'pending' ? 'warning' : 'success'" size="small">
            {{ row.status === 'pending' ? '待确认' : '已加入' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="空间角色" width="180">
        <template #default="{ row }">
          <el-select
            v-if="canManage && row.status === 'active'"
            :model-value="row.role"
            size="small"
            @change="(v: string) => onRoleChange(row, v)"
          >
            <el-option label="空间管理员" value="space_admin" />
            <el-option label="成员" value="member" />
          </el-select>
          <span v-else>{{ spaceRoleLabel(row.role) }}</span>
        </template>
      </el-table-column>
      <el-table-column v-if="canManage" label="操作" width="160">
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <el-button link type="success" @click="onApprove(row)">通过</el-button>
            <el-button link type="danger" @click="onReject(row)">拒绝</el-button>
          </template>
          <el-button
            v-else
            link
            type="danger"
            :disabled="row.user_id === spaceOwnerId"
            @click="onRemove(row)"
          >
            移除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="addVisible" title="添加成员" width="520px">
      <el-form label-width="90px">
        <el-form-item label="选择用户">
          <el-select
            v-model="addForm.user_id"
            filterable
            placeholder="请选择用户"
            style="width: 100%"
            :loading="candidatesLoading"
          >
            <el-option
              v-for="u in candidates"
              :key="u.id"
              :label="`${u.username} (ID: ${u.id})`"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="空间角色">
          <el-select v-model="addForm.role" style="width: 100%">
            <el-option label="成员" value="member" />
            <el-option label="空间管理员" value="space_admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" :loading="adding" @click="onAdd">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SpaceMember, User } from '@/types'
import * as spaceApi from '@/api/space'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const store = useUserStore()
const list = ref<SpaceMember[]>([])
const candidates = ref<User[]>([])
const loading = ref(false)
const candidatesLoading = ref(false)
const adding = ref(false)
const addVisible = ref(false)
const spaceId = Number(route.params.id)
const spaceOwnerId = ref<number | null>(null)
const addForm = reactive({ user_id: null as number | null, role: 'member' })

const canManage = computed(() => {
  if (store.isSystemAdmin) return true
  const me = list.value.find((m) => m.user_id === store.user?.id)
  return me?.role === 'space_admin' && me?.status === 'active'
})

function roleLabel(role?: string) {
  if (role === 'system_admin') return '系统管理员'
  if (role === 'member') return '成员'
  return role || '-'
}

function spaceRoleLabel(role?: string) {
  if (role === 'space_admin') return '空间管理员'
  if (role === 'member') return '成员'
  return role || '-'
}

async function load() {
  loading.value = true
  try {
    const [members, space] = await Promise.all([
      spaceApi.listMembers(spaceId),
      spaceApi.getSpace(spaceId),
    ])
    list.value = members || []
    spaceOwnerId.value = space?.owner_id ?? null
  } finally {
    loading.value = false
  }
}

async function openAdd() {
  addForm.user_id = null
  addForm.role = 'member'
  addVisible.value = true
  candidatesLoading.value = true
  try {
    candidates.value = (await spaceApi.listCandidateUsers(spaceId)) || []
  } finally {
    candidatesLoading.value = false
  }
}

async function onAdd() {
  if (!addForm.user_id) {
    ElMessage.warning('请选择用户')
    return
  }
  adding.value = true
  try {
    await spaceApi.addMember(spaceId, { user_id: addForm.user_id, role: addForm.role })
    ElMessage.success('已添加成员')
    addVisible.value = false
    await load()
  } finally {
    adding.value = false
  }
}

async function onApprove(row: SpaceMember) {
  const name = row.user?.username || `用户 ${row.user_id}`
  await ElMessageBox.confirm(`确认通过「${name}」的加入申请？`, '提示', { type: 'info' })
  await spaceApi.approveMember(spaceId, row.user_id)
  ElMessage.success('已通过')
  await load()
}

async function onReject(row: SpaceMember) {
  const name = row.user?.username || `用户 ${row.user_id}`
  await ElMessageBox.confirm(`确认拒绝「${name}」的加入申请？`, '提示', { type: 'warning' })
  await spaceApi.rejectMember(spaceId, row.user_id)
  ElMessage.success('已拒绝')
  await load()
}

async function onRemove(row: SpaceMember) {
  const name = row.user?.username || `用户 ${row.user_id}`
  await ElMessageBox.confirm(`确认将「${name}」移出该空间？`, '提示', { type: 'warning' })
  await spaceApi.removeMember(spaceId, row.user_id)
  ElMessage.success('已移除')
  await load()
}

async function onRoleChange(row: SpaceMember, role: string) {
  await spaceApi.updateMemberRole(spaceId, row.user_id, role)
  ElMessage.success('角色已更新')
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
