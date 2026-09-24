<template>
  <div>
    <el-form-item label="类型">
      <el-select v-model="model.type" style="width: 100%">
        <el-option label="http" value="http" />
        <el-option label="https" value="https" />
        <el-option label="tcp" value="tcp" />
      </el-select>
    </el-form-item>
    <el-form-item v-if="active" label="HTTP Path">
      <el-input v-model="model.http_path" />
    </el-form-item>
    <el-form-item v-if="active" label="超时">
      <el-input-number v-model="model.timeout" :min="0" />
    </el-form-item>
    <el-form-item v-if="active" label="并发">
      <el-input-number v-model="model.concurrency" :min="1" />
    </el-form-item>
    <el-form-item v-if="active" label="校验证书">
      <el-switch v-model="model.https_verify_certificate" />
    </el-form-item>
    <el-form-item v-if="active" label="健康间隔">
      <el-input-number v-model="model.healthy_interval" :min="0" />
    </el-form-item>
    <el-form-item label="成功次数">
      <el-input-number v-model="model.healthy_successes" :min="0" />
    </el-form-item>
    <el-form-item label="健康状态码">
      <el-input :model-value="joinCodes(model.healthy_http_statuses)" @update:model-value="setCodes('healthy', $event)" />
    </el-form-item>
    <el-form-item v-if="active" label="不健康间隔">
      <el-input-number v-model="model.unhealthy_interval" :min="0" />
    </el-form-item>
    <el-form-item label="HTTP 失败">
      <el-input-number v-model="model.unhealthy_http_failures" :min="0" />
    </el-form-item>
    <el-form-item label="TCP 失败">
      <el-input-number v-model="model.unhealthy_tcp_failures" :min="0" />
    </el-form-item>
    <el-form-item label="超时次数">
      <el-input-number v-model="model.unhealthy_timeouts" :min="0" />
    </el-form-item>
    <el-form-item label="不健康状态码">
      <el-input :model-value="joinCodes(model.unhealthy_http_statuses)" @update:model-value="setCodes('unhealthy', $event)" />
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import type { UpstreamHealthSide } from '@/types'

defineProps<{ active?: boolean }>()
const model = defineModel<UpstreamHealthSide>({ required: true })

function joinCodes(codes?: number[]) {
  return (codes || []).join(',')
}

function setCodes(kind: 'healthy' | 'unhealthy', raw: string) {
  const codes = raw
    .split(/[,，\s]+/)
    .map((s) => Number(s.trim()))
    .filter((n) => Number.isInteger(n) && n >= 100 && n <= 599)
  if (kind === 'healthy') model.value.healthy_http_statuses = codes
  else model.value.unhealthy_http_statuses = codes
}
</script>
