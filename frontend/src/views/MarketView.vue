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
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-tooltip
            :disabled="!!row.auth_enabled"
            content="未启用认证的 API 无法关联 Consumer"
            placement="top"
          >
            <span>
              <el-button link type="primary" :disabled="!row.auth_enabled" @click="openLink(row)">
                关联API
              </el-button>
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

    <el-dialog v-model="linkVisible" :title="linkTitle" width="720px">
      <el-table
        ref="consumerTableRef"
        :data="consumerPaged"
        v-loading="consumerLoading"
        empty-text="暂无可关联的 Consumer"
        stripe
        row-key="id"
        @selection-change="onConsumerSelection"
      >
        <el-table-column type="selection" width="48" reserve-selection />
        <el-table-column prop="username" label="consumer名称" min-width="140" />
        <el-table-column label="所属空间" min-width="140">
          <template #default="{ row }">{{ row.space?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="凭证" min-width="180">
          <template #default="{ row }">
            <el-tag
              v-for="(c, i) in visibleCreds(row)"
              :key="i"
              size="small"
              style="margin: 2px 4px 2px 0"
            >
              {{ c.plugin }}
            </el-tag>
            <span v-if="!visibleCreds(row).length">-</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="dialog-pagination">
        <el-pagination
          :current-page="consumerPage"
          :page-size="consumerPageSize"
          :total="consumerList.length"
          :page-sizes="[5, 10, 20]"
          layout="total, sizes, prev, pager, next"
          small
          background
          @update:current-page="consumerPage = $event"
          @update:page-size="onConsumerPageSize"
        />
      </div>
      <template #footer>
        <el-button @click="linkVisible = false">取消</el-button>
        <el-button type="primary" :loading="linking" :disabled="!selectedConsumers.length" @click="confirmLink">
          确定关联
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TableInstance } from 'element-plus'
import type { ApiItem, ConsumerCredential, ConsumerItem } from '@/types'
import * as apiMod from '@/api/api'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const list = ref<ApiItem[]>([])
const { page, pageSize, total, paged, pageSizes } = usePagination(list)
const loading = ref(false)

const linkVisible = ref(false)
const linkTitle = ref('关联 API')
const linkingApi = ref<ApiItem | null>(null)
const consumerList = ref<ConsumerItem[]>([])
const consumerLoading = ref(false)
const linking = ref(false)
const selectedConsumers = ref<ConsumerItem[]>([])
const consumerTableRef = ref<TableInstance>()
const consumerPage = ref(1)
const consumerPageSize = ref(10)
const consumerPaged = computed(() => {
  const start = (consumerPage.value - 1) * consumerPageSize.value
  return consumerList.value.slice(start, start + consumerPageSize.value)
})

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

function visibleCreds(row: ConsumerItem): ConsumerCredential[] {
  return (row.credentials || []).filter((c) => !(c.plugin === 'acl' && /^G-\d+$/.test(c.config?.group || '')))
}

function onConsumerSelection(rows: ConsumerItem[]) {
  selectedConsumers.value = rows
}

function onConsumerPageSize(size: number) {
  consumerPageSize.value = size
  consumerPage.value = 1
}

async function openLink(row: ApiItem) {
  if (!row.auth_enabled) return
  linkingApi.value = row
  linkTitle.value = `关联 API「${row.name}」`
  linkVisible.value = true
  consumerList.value = []
  selectedConsumers.value = []
  consumerPage.value = 1
  consumerTableRef.value?.clearSelection()
  consumerLoading.value = true
  try {
    consumerList.value = (await apiMod.listMarketLinkableConsumers(row.id)) || []
  } finally {
    consumerLoading.value = false
  }
}

async function confirmLink() {
  if (!linkingApi.value || !selectedConsumers.value.length) return
  linking.value = true
  try {
    await apiMod.linkMarketConsumers(
      linkingApi.value.id,
      selectedConsumers.value.map((c) => c.id),
    )
    ElMessage.success('关联成功')
    linkVisible.value = false
  } finally {
    linking.value = false
  }
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
.dialog-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
