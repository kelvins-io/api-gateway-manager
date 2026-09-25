<template>
  <div>
    <div class="toolbar">
      <el-button @click="load">刷新</el-button>
    </div>

    <el-table :data="paged" v-loading="loading" stripe empty-text="暂无已分享的 API">
      <el-table-column label="所属空间" min-width="140">
        <template #default="{ row }">{{ row.group?.space?.name || '-' }}</template>
      </el-table-column>
      <el-table-column prop="name" label="API 名" min-width="140" />
      <el-table-column label="接入协议" width="160">
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
      <el-table-column label="访问地址" min-width="320">
        <template #default="{ row }">
          <div v-for="p in splitPaths(row.access_path)" :key="p" class="access-url">
            {{ fullAccessPath(row, p) }}
          </div>
          <span v-if="!splitPaths(row.access_path).length">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="access_methods" label="请求方法" width="140" />
      <el-table-column label="认证类型" width="120">
        <template #default="{ row }">
          <span v-if="row.auth_enabled">{{ row.auth_plugin }}</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="current_version" label="当前发布版本" width="120" />
    </el-table>
    <ListPagination
      v-model:page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="pageSizes"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { ApiItem } from '@/types'
import * as apiMod from '@/api/api'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const list = ref<ApiItem[]>([])
const { page, pageSize, total, paged, pageSizes } = usePagination(list)
const loading = ref(false)

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

async function load() {
  loading.value = true
  try {
    list.value = (await apiMod.listMarketApis()) || []
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
}
.access-url {
  line-height: 28px;
  word-break: break-all;
}
</style>
