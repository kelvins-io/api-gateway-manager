<template>
  <div class="list-pagination">
    <el-pagination
      :current-page="page"
      :page-size="pageSize"
      :page-sizes="pageSizes"
      :total="total"
      layout="total, sizes, prev, pager, next, jumper"
      background
      @update:current-page="emit('update:page', $event)"
      @update:page-size="onSize"
    />
  </div>
</template>

<script setup lang="ts">
import { PAGE_SIZES } from '@/composables/usePagination'

defineProps<{
  page: number
  pageSize: number
  total: number
  pageSizes?: number[]
}>()

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()

function onSize(size: number) {
  emit('update:pageSize', size)
  emit('update:page', 1)
}
</script>

<style scoped>
.list-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
