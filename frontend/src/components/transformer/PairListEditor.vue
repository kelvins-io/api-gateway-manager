<template>
  <div class="section">
    <div class="section-title">{{ label }}</div>
    <div v-for="(row, i) in model" :key="i" class="row">
      <el-input v-model="row.key" :placeholder="keyLabel" />
      <el-input v-model="row.value" :placeholder="valueLabel" />
      <el-button link type="danger" @click="model.splice(i, 1)">删除</el-button>
    </div>
    <el-button link type="primary" @click="model.push({ key: '', value: '' })">添加</el-button>
  </div>
</template>

<script setup lang="ts">
import type { RtPair } from './types'

withDefaults(
  defineProps<{
    label: string
    keyLabel?: string
    valueLabel?: string
  }>(),
  {
    keyLabel: '名称',
    valueLabel: '值',
  },
)

const model = defineModel<RtPair[]>({ required: true })
</script>

<style scoped>
.section {
  margin-bottom: 14px;
}
.section-title {
  margin-bottom: 6px;
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}
.row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
}
</style>
