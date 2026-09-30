<template>
  <div class="form-data-editor">
    <div v-for="(row, i) in model" :key="i" class="fd-row">
      <el-checkbox v-model="row.enabled" />
      <el-input v-model="row.key" placeholder="字段名" @input="ensureTail" />
      <el-select v-model="row.type" style="width: 100px" @change="onTypeChange(row)">
        <el-option label="Text" value="text" />
        <el-option label="File" value="file" />
      </el-select>
      <el-input
        v-if="row.type === 'text'"
        v-model="row.value"
        placeholder="字段值"
        @input="ensureTail"
      />
      <div v-else class="file-cell">
        <input
          type="file"
          class="file-input"
          @change="onFileChange(row, $event)"
        />
        <span class="file-name">{{ row.file?.name || '未选择文件' }}</span>
      </div>
      <el-button link type="danger" :disabled="model.length <= 1" @click="remove(i)">删除</el-button>
    </div>
    <el-button link type="primary" @click="add">添加</el-button>
  </div>
</template>

<script setup lang="ts">
export interface FormDataField {
  enabled: boolean
  key: string
  type: 'text' | 'file'
  value: string
  file: File | null
}

const model = defineModel<FormDataField[]>({ required: true })

function empty(): FormDataField {
  return { enabled: true, key: '', type: 'text', value: '', file: null }
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
  if (last && (last.key || last.value || last.file)) {
    model.value = [...list, empty()]
  }
}

function onTypeChange(row: FormDataField) {
  row.value = ''
  row.file = null
  ensureTail()
}

function onFileChange(row: FormDataField, ev: Event) {
  const input = ev.target as HTMLInputElement
  row.file = input.files?.[0] || null
  ensureTail()
}
</script>

<style scoped>
.form-data-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.fd-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fd-row > .el-input {
  flex: 1;
  min-width: 0;
}
.file-cell {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.file-input {
  max-width: 180px;
}
.file-name {
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
