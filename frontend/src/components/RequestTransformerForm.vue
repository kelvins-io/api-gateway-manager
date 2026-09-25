<template>
  <div class="rt-form">
    <el-form-item label="http_method">
      <el-select v-model="model.http_method" clearable filterable style="width: 100%" placeholder="可选，改写请求方法">
        <el-option v-for="m in HTTP_METHODS" :key="m" :label="m" :value="m" />
      </el-select>
    </el-form-item>

    <el-tabs v-model="activeTab" type="border-card" class="rt-tabs">
      <el-tab-pane label="Remove" name="remove">
        <p class="rt-hint">删除请求中的 Header / Query / Body 字段（仅名称）</p>
        <NameListEditor v-model="model.remove.headers" label="headers" />
        <NameListEditor v-model="model.remove.querystring" label="querystring" />
        <NameListEditor v-model="model.remove.body" label="body" />
      </el-tab-pane>

      <el-tab-pane label="Rename" name="rename">
        <p class="rt-hint">重命名字段，格式为 旧名 → 新名</p>
        <PairListEditor v-model="model.rename.headers" label="headers" key-label="旧名" value-label="新名" />
        <PairListEditor v-model="model.rename.querystring" label="querystring" key-label="旧名" value-label="新名" />
        <PairListEditor v-model="model.rename.body" label="body" key-label="旧名" value-label="新名" />
      </el-tab-pane>

      <el-tab-pane label="Replace" name="replace">
        <p class="rt-hint">替换已有字段的值；可同时改写 URI</p>
        <el-form-item label="uri">
          <el-input v-model="model.replace.uri" placeholder="可选，如 /new/path" />
        </el-form-item>
        <PairListEditor v-model="model.replace.headers" label="headers" key-label="名称" value-label="新值" />
        <PairListEditor v-model="model.replace.querystring" label="querystring" key-label="名称" value-label="新值" />
        <PairListEditor v-model="model.replace.body" label="body" key-label="名称" value-label="新值" />
      </el-tab-pane>

      <el-tab-pane label="Add" name="add">
        <p class="rt-hint">仅当字段不存在时添加</p>
        <PairListEditor v-model="model.add.headers" label="headers" key-label="名称" value-label="值" />
        <PairListEditor v-model="model.add.querystring" label="querystring" key-label="名称" value-label="值" />
        <PairListEditor v-model="model.add.body" label="body" key-label="名称" value-label="值" />
      </el-tab-pane>

      <el-tab-pane label="Append" name="append">
        <p class="rt-hint">追加字段值（已存在则附加）</p>
        <PairListEditor v-model="model.append.headers" label="headers" key-label="名称" value-label="值" />
        <PairListEditor v-model="model.append.querystring" label="querystring" key-label="名称" value-label="值" />
        <PairListEditor v-model="model.append.body" label="body" key-label="名称" value-label="值" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { HTTP_METHODS } from '@/constants/kongPluginFormSchemas'
import NameListEditor from './transformer/NameListEditor.vue'
import PairListEditor from './transformer/PairListEditor.vue'
import type { RequestTransformerModel } from './transformer/types'

defineProps<{
  model: RequestTransformerModel
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
