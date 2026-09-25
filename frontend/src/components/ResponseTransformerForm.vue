<template>
  <div class="rt-form">
    <el-tabs v-model="activeTab" type="border-card" class="rt-tabs">
      <el-tab-pane label="Remove" name="remove">
        <p class="rt-hint">删除响应中的 Header / JSON 字段（仅名称）</p>
        <NameListEditor v-model="model.remove.headers" label="headers" />
        <NameListEditor v-model="model.remove.json" label="json" />
      </el-tab-pane>

      <el-tab-pane label="Rename" name="rename">
        <p class="rt-hint">重命名响应 Header，格式为 旧名 → 新名</p>
        <PairListEditor v-model="model.rename.headers" label="headers" key-label="旧名" value-label="新名" />
      </el-tab-pane>

      <el-tab-pane label="Replace" name="replace">
        <p class="rt-hint">替换已有字段的值；JSON 需指定值类型</p>
        <PairListEditor v-model="model.replace.headers" label="headers" key-label="名称" value-label="新值" />
        <JsonPairListEditor v-model="model.replace.json" label="json" key-label="名称" value-label="新值" />
      </el-tab-pane>

      <el-tab-pane label="Add" name="add">
        <p class="rt-hint">仅当字段不存在时添加</p>
        <PairListEditor v-model="model.add.headers" label="headers" key-label="名称" value-label="值" />
        <JsonPairListEditor v-model="model.add.json" label="json" key-label="名称" value-label="值" />
      </el-tab-pane>

      <el-tab-pane label="Append" name="append">
        <p class="rt-hint">追加字段值（已存在则附加）</p>
        <PairListEditor v-model="model.append.headers" label="headers" key-label="名称" value-label="值" />
        <JsonPairListEditor v-model="model.append.json" label="json" key-label="名称" value-label="值" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import NameListEditor from './transformer/NameListEditor.vue'
import PairListEditor from './transformer/PairListEditor.vue'
import JsonPairListEditor from './transformer/JsonPairListEditor.vue'
import type { ResponseTransformerModel } from './transformer/types'

defineProps<{
  model: ResponseTransformerModel
}>()

const activeTab = ref('remove')
</script>

<style scoped>
.rt-form {
  margin-top: 4px;
}
.rt-tabs {
  width: 100%;
}
.rt-hint {
  margin: 0 0 12px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
</style>
