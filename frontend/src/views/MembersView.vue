<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.back()">返回</el-button>
      <el-button @click="load">刷新</el-button>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="user_id" label="用户 ID" width="100" />
      <el-table-column label="用户名">
        <template #default="{ row }">{{ row.user?.username || '-' }}</template>
      </el-table-column>
      <el-table-column label="系统角色" width="140">
        <template #default="{ row }">{{ row.user?.role || '-' }}</template>
      </el-table-column>
      <el-table-column label="空间角色" width="180">
        <template #default="{ row }">
          <el-select
            :model-value="row.role"
            size="small"
            @change="(v: string) => onRoleChange(row, v)"
          >
            <el-option label="空间管理员" value="space_admin" />
            <el-option label="成员" value="member" />
          </el-select>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { SpaceMember } from '@/types'
import * as spaceApi from '@/api/space'

const route = useRoute()
const list = ref<SpaceMember[]>([])
const loading = ref(false)
const spaceId = Number(route.params.id)

async function load() {
  loading.value = true
  try {
    list.value = (await spaceApi.listMembers(spaceId)) || []
  } finally {
    loading.value = false
  }
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
