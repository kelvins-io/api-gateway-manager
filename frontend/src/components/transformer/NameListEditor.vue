<template>
  <div class="section">
    <div class="section-title">{{ label }}</div>
    <div v-for="(_, i) in model" :key="i" class="row">
      <el-input v-model="model[i]" placeholder="字段名" />
      <el-button link type="danger" @click="model.splice(i, 1)">删除</el-button>
    </div>
    <el-button link type="primary" @click="model.push('')">添加</el-button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
}>()

const model = defineModel<string[]>({ required: true })
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
