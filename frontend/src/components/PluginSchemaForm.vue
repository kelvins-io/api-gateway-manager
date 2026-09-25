<template>
  <div>
    <el-form-item v-for="f in fields" :key="f.key" :label="f.label" :required="f.required">
      <el-switch v-if="f.type === 'boolean'" :model-value="Boolean(model[f.key])" @update:model-value="(v: boolean) => setField(f.key, v)" />
      <el-input-number
        v-else-if="f.type === 'number'"
        :model-value="numValue(model[f.key])"
        :min="f.min"
        :max="f.max"
        style="width: 100%"
        @update:model-value="(v: number | undefined) => setField(f.key, v)"
      />
      <el-select
        v-else-if="f.type === 'select'"
        :model-value="(model[f.key] as string | number | undefined)"
        style="width: 100%"
        clearable
        @update:model-value="(v: string | number | undefined) => setField(f.key, v)"
      >
        <el-option v-for="opt in f.options || []" :key="String(opt)" :label="String(opt)" :value="opt" />
      </el-select>
      <el-input
        v-else-if="f.type === 'textarea' || f.type === 'json'"
        :model-value="String(model[f.key] ?? '')"
        type="textarea"
        :rows="f.type === 'json' ? 6 : 4"
        :placeholder="f.placeholder"
        @update:model-value="(v: string) => setField(f.key, v)"
      />
      <el-input
        v-else
        :model-value="String(model[f.key] ?? '')"
        :placeholder="f.placeholder"
        @update:model-value="(v: string) => setField(f.key, v)"
      />
      <div v-if="f.hint" class="hint">{{ f.hint }}</div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { pluginFormSchemas, type FormField } from '@/constants/kongPluginFormSchemas'

const props = defineProps<{
  plugin: string
  model: Record<string, unknown>
}>()

const fields = computed<FormField[]>(() => pluginFormSchemas[props.plugin] || [])

function numValue(v: unknown) {
  if (v === '' || v === undefined || v === null) return undefined
  const n = Number(v)
  return Number.isNaN(n) ? undefined : n
}

function setField(key: string, value: unknown) {
  props.model[key] = value
}
</script>

<style scoped>
.hint {
  margin-top: 4px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
</style>
