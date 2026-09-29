<template>
  <div class="kv-editor">
    <div v-for="(row, i) in model" :key="i" class="kv-row">
      <el-checkbox v-model="row.enabled" />
      <el-input v-model="row.key" :placeholder="keyPlaceholder" @input="ensureTail" />
      <el-input v-model="row.value" :placeholder="valuePlaceholder" @input="ensureTail" />
      <el-button link type="danger" :disabled="model.length <= 1" @click="remove(i)">删除</el-button>
    </div>
    <el-button link type="primary" @click="add">添加</el-button>
  </div>
</template>

<script setup lang="ts">
export interface KvPair {
  enabled: boolean
  key: string
  value: string
}

const model = defineModel<KvPair[]>({ required: true })

defineProps<{
  keyPlaceholder?: string
  valuePlaceholder?: string
}>()

function empty(): KvPair {
  return { enabled: true, key: '', value: '' }
}

function add() {
  model.value = [...model.value, empty()]
}

function remove(i: number) {
  const next = model.value.slice()
  next.splice(i, 1)
  if (!next.length) next.push(empty())
  model.value = next
}

function ensureTail() {
  const list = model.value
  const last = list[list.length - 1]
  if (last && (last.key || last.value)) {
    model.value = [...list, empty()]
  }
}
</script>

<style scoped>
.kv-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.kv-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.kv-row .el-input {
  flex: 1;
}
</style>
