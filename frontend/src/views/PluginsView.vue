<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.push('/groups')">返回分组</el-button>
      <el-button type="primary" :disabled="!store.currentSpaceId" @click="openCreate">新建 Plugin</el-button>
      <el-button @click="load">刷新</el-button>
      <span v-if="store.currentSpace" class="hint">当前空间：{{ store.currentSpace.name }}</span>
    </div>

    <el-table :data="paged" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="plugin" label="类型" width="200" />
      <el-table-column label="启用" width="90">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="配置" min-width="240">
        <template #default="{ row }">{{ summarize(row) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" @click="openApis(row)">关联API</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <ListPagination
      v-model:page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="pageSizes"
    />

    <el-dialog v-model="apiVisible" :title="apiTitle" width="720px">
      <el-table :data="apiRows" v-loading="apiLoading" empty-text="暂无关联 API" stripe>
        <el-table-column prop="name" label="API 名称" />
        <el-table-column label="所属分组">
          <template #default="{ row }">{{ row.group?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="当前发布版本" width="140">
          <template #default="{ row }">{{ row.status === 'published' && row.current_version ? row.current_version : '-' }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="visible" :title="editing ? '编辑 Plugin' : '新建 Plugin'" width="720px">
      <el-form :model="form" label-width="180px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="字母、数字、点、下划线或中划线" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.plugin" style="width: 100%" :disabled="!!editing" filterable @change="onKind">
            <el-option-group v-for="cat in pluginCategories" :key="cat.label" :label="cat.label">
              <el-option v-for="p in cat.plugins" :key="p" :label="p" :value="p" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <PluginSchemaForm v-if="useSchemaForm" :plugin="form.plugin" :model="schemaConfig" />
        <RequestTransformerForm v-else-if="form.plugin === 'request-transformer'" :model="rtConfig" />
        <ResponseTransformerForm v-else-if="form.plugin === 'response-transformer'" :model="respConfig" />
        <template v-else-if="form.plugin === 'rate-limiting'">
          <el-form-item label="每分钟">
            <el-input-number v-model="form.config.minute" :min="0" />
          </el-form-item>
          <el-form-item label="每小时">
            <el-input-number v-model="form.config.hour" :min="0" />
          </el-form-item>
          <el-form-item label="每天">
            <el-input-number v-model="form.config.day" :min="0" />
          </el-form-item>
          <el-form-item label="按什么限流">
            <el-select v-model="form.config.limit_by" style="width: 100%">
              <el-option v-for="v in ['consumer', 'credential', 'ip', 'service', 'header', 'path']" :key="v" :label="v" :value="v" />
            </el-select>
          </el-form-item>
          <el-form-item label="策略">
            <el-select v-model="form.config.policy" style="width: 100%">
              <el-option v-for="v in ['local', 'cluster', 'redis']" :key="v" :label="v" :value="v" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'cors'">
          <el-form-item label="Origins">
            <el-input v-model="form.config.origins" placeholder="多个用逗号分隔，默认 *" />
          </el-form-item>
          <el-form-item label="Methods">
            <el-select v-model="form.config.methods" multiple collapse-tags collapse-tags-tooltip filterable style="width: 100%" placeholder="留空使用全部方法">
              <el-option v-for="v in HTTP_METHODS" :key="v" :label="v" :value="v" />
            </el-select>
          </el-form-item>
          <el-form-item label="允许凭证">
            <el-switch v-model="form.config.credentials" />
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'key-auth'">
          <el-form-item label="Key 名称">
            <el-input v-model="form.config.key_names" placeholder="默认 apikey，多个用逗号分隔" />
          </el-form-item>
          <el-form-item label="隐藏凭证">
            <el-switch v-model="form.config.hide_credentials" />
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'acl'">
          <el-form-item label="Allow">
            <el-input v-model="form.config.allow" placeholder="分组名，逗号分隔" />
          </el-form-item>
          <el-form-item label="Deny">
            <el-input v-model="form.config.deny" placeholder="与 Allow 互斥" />
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'ip-restriction'">
          <el-form-item label="Allow">
            <el-input v-model="form.config.allow" placeholder="CIDR 或 IP，逗号分隔" />
          </el-form-item>
          <el-form-item label="Deny">
            <el-input v-model="form.config.deny" placeholder="与 Allow 互斥" />
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'request-size-limiting'">
          <el-form-item label="允许大小">
            <el-input-number v-model="form.config.allowed_payload_size" :min="1" />
          </el-form-item>
          <el-form-item label="单位">
            <el-select v-model="form.config.size_unit" style="width: 100%">
              <el-option label="megabytes" value="megabytes" />
              <el-option label="kilobytes" value="kilobytes" />
              <el-option label="bytes" value="bytes" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'jwt'">
          <el-form-item label="Header">
            <el-input v-model="form.config.header_names" placeholder="默认 authorization" />
          </el-form-item>
          <el-form-item label="校验声明">
            <el-select v-model="form.config.claims_to_verify" multiple collapse-tags style="width: 100%" placeholder="可选 exp / nbf">
              <el-option v-for="v in JWT_CLAIMS" :key="v" :label="v" :value="v" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'basic-auth' || form.plugin === 'hmac-auth'">
          <el-form-item label="隐藏凭证">
            <el-switch v-model="form.config.hide_credentials" />
          </el-form-item>
          <el-form-item v-if="form.plugin === 'hmac-auth'" label="时钟偏移(秒)">
            <el-input-number v-model="form.config.clock_skew" :min="0" />
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'request-termination'">
          <el-form-item label="状态码">
            <el-input-number v-model="form.config.status_code" :min="100" :max="599" />
          </el-form-item>
          <el-form-item label="消息">
            <el-input v-model="form.config.message" />
          </el-form-item>
        </template>
        <template v-else-if="form.plugin === 'correlation-id'">
          <el-form-item label="Header">
            <el-input v-model="form.config.header_name" placeholder="Kong-Request-ID" />
          </el-form-item>
          <el-form-item label="生成器">
            <el-select v-model="form.config.generator" style="width: 100%">
              <el-option label="uuid" value="uuid" />
              <el-option label="uuid#counter" value="uuid#counter" />
              <el-option label="tracker" value="tracker" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="Config JSON">
            <el-input
              v-model="form.configJson"
              type="textarea"
              :rows="10"
              placeholder='Kong 插件 config，例如 {"second":5}'
            />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ApiItem, PluginItem } from '@/types'
import * as apiMod from '@/api/api'
import { useUserStore } from '@/stores/user'
import ListPagination from '@/components/ListPagination.vue'
import PluginSchemaForm from '@/components/PluginSchemaForm.vue'
import RequestTransformerForm from '@/components/RequestTransformerForm.vue'
import ResponseTransformerForm from '@/components/ResponseTransformerForm.vue'
import {
  buildRequestTransformerPayload,
  buildResponseTransformerPayload,
  emptyRequestTransformer,
  emptyResponseTransformer,
  fillRequestTransformer,
  fillResponseTransformer,
  type RequestTransformerModel,
  type ResponseTransformerModel,
} from '@/components/transformer/types'
import { usePagination } from '@/composables/usePagination'
import { hasPluginForm, pluginCategories } from '@/constants/kongPlugins'
import {
  buildSchemaPayload,
  defaultSchemaValues,
  fillSchemaValues,
  hasSchemaForm,
  HTTP_METHODS,
  JWT_CLAIMS,
} from '@/constants/kongPluginFormSchemas'

const store = useUserStore()
const list = ref<PluginItem[]>([])
const { page, pageSize, total, paged, pageSizes } = usePagination(list)
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editing = ref<PluginItem | null>(null)
const apiVisible = ref(false)
const apiTitle = ref('关联 API')
const apiRows = ref<ApiItem[]>([])
const apiLoading = ref(false)
const schemaConfig = reactive<Record<string, unknown>>({})
const rtConfig = reactive<RequestTransformerModel>(emptyRequestTransformer())
const respConfig = reactive<ResponseTransformerModel>(emptyResponseTransformer())

const useSchemaForm = computed(() => !hasPluginForm(form.plugin) && hasSchemaForm(form.plugin))

function emptyConfig() {
  return {
    minute: 0,
    hour: 0,
    day: 0,
    limit_by: 'consumer',
    policy: 'local',
    origins: '*',
    methods: [] as string[],
    credentials: false,
    key_names: 'apikey',
    hide_credentials: false,
    clock_skew: 300,
    allow: '',
    deny: '',
    allowed_payload_size: 1,
    size_unit: 'megabytes',
    header_names: 'authorization',
    claims_to_verify: [] as string[],
    status_code: 503,
    message: '',
    header_name: 'Kong-Request-ID',
    generator: 'uuid',
  }
}

const form = reactive({
  name: '',
  plugin: 'rate-limiting',
  enabled: true,
  config: emptyConfig(),
  configJson: '{}',
})

function resetSchemaConfig(plugin: string, src?: Record<string, unknown>) {
  const next = src ? fillSchemaValues(plugin, src) : defaultSchemaValues(plugin)
  Object.keys(schemaConfig).forEach((k) => delete schemaConfig[k])
  Object.assign(schemaConfig, next)
}

function resetRtConfig(src?: Record<string, unknown>) {
  const next = fillRequestTransformer(src)
  Object.assign(rtConfig, emptyRequestTransformer())
  rtConfig.http_method = next.http_method
  rtConfig.remove = next.remove
  rtConfig.rename = next.rename
  rtConfig.replace = next.replace
  rtConfig.add = next.add
  rtConfig.append = next.append
}

function resetRespConfig(src?: Record<string, unknown>) {
  const next = fillResponseTransformer(src)
  Object.assign(respConfig, emptyResponseTransformer())
  respConfig.remove = next.remove
  respConfig.rename = next.rename
  respConfig.replace = next.replace
  respConfig.add = next.add
  respConfig.append = next.append
}

function asText(value: unknown) {
  if (Array.isArray(value)) return value.join(',')
  if (value == null) return ''
  return String(value)
}

function asList(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String).filter(Boolean)
  if (value == null || value === '') return []
  return String(value)
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function fillConfig(row?: PluginItem) {
  const cfg = emptyConfig()
  const src = (row?.config || {}) as Record<string, unknown>
  cfg.minute = Number(src.minute || 0)
  cfg.hour = Number(src.hour || 0)
  cfg.day = Number(src.day || 0)
  cfg.limit_by = asText(src.limit_by) || 'consumer'
  cfg.policy = asText(src.policy) || 'local'
  cfg.origins = asText(src.origins) || '*'
  cfg.methods = asList(src.methods)
  cfg.credentials = Boolean(src.credentials)
  cfg.key_names = asText(src.key_names) || 'apikey'
  cfg.hide_credentials = Boolean(src.hide_credentials)
  cfg.clock_skew = Number(src.clock_skew || 300)
  cfg.allow = asText(src.allow)
  cfg.deny = asText(src.deny)
  cfg.allowed_payload_size = Number(src.allowed_payload_size || 1)
  cfg.size_unit = asText(src.size_unit) || 'megabytes'
  cfg.header_names = asText(src.header_names) || 'authorization'
  cfg.claims_to_verify = asList(src.claims_to_verify)
  cfg.status_code = Number(src.status_code || 503)
  cfg.message = asText(src.message)
  cfg.header_name = asText(src.header_name) || 'Kong-Request-ID'
  cfg.generator = asText(src.generator) || 'uuid'
  return cfg
}

function fillConfigJson(row?: PluginItem) {
  const src = row?.config
  if (!src || typeof src !== 'object') return '{}'
  try {
    return JSON.stringify(src, null, 2)
  } catch {
    return '{}'
  }
}

function payloadConfig(): Record<string, unknown> | null {
  if (form.plugin === 'request-transformer') {
    return buildRequestTransformerPayload(rtConfig)
  }
  if (form.plugin === 'response-transformer') {
    return buildResponseTransformerPayload(respConfig)
  }
  if (hasSchemaForm(form.plugin) && !hasPluginForm(form.plugin)) {
    const result = buildSchemaPayload(form.plugin, schemaConfig)
    if (!result.ok) {
      ElMessage.warning(result.error)
      return null
    }
    return result.config
  }
  if (!hasPluginForm(form.plugin)) {
    try {
      const parsed = JSON.parse(form.configJson || '{}')
      if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        ElMessage.warning('Config JSON 必须是对象')
        return null
      }
      return parsed as Record<string, unknown>
    } catch {
      ElMessage.warning('Config JSON 格式无效')
      return null
    }
  }
  const c = form.config
  if (form.plugin === 'rate-limiting') {
    return { minute: c.minute, hour: c.hour, day: c.day, limit_by: c.limit_by, policy: c.policy }
  }
  if (form.plugin === 'cors') {
    return { origins: c.origins, methods: c.methods, credentials: c.credentials }
  }
  if (form.plugin === 'key-auth') {
    return { key_names: c.key_names, hide_credentials: c.hide_credentials }
  }
  if (form.plugin === 'acl' || form.plugin === 'ip-restriction') {
    return { allow: c.allow, deny: c.deny }
  }
  if (form.plugin === 'request-size-limiting') {
    return { allowed_payload_size: c.allowed_payload_size, size_unit: c.size_unit }
  }
  if (form.plugin === 'jwt') {
    return { header_names: c.header_names, claims_to_verify: c.claims_to_verify }
  }
  if (form.plugin === 'basic-auth') {
    return { hide_credentials: c.hide_credentials }
  }
  if (form.plugin === 'hmac-auth') {
    return { hide_credentials: c.hide_credentials, clock_skew: c.clock_skew }
  }
  if (form.plugin === 'request-termination') {
    return { status_code: c.status_code, message: c.message }
  }
  return { header_name: c.header_name, generator: c.generator }
}

function summarize(row: PluginItem) {
  const cfg = row.config || {}
  const parts = Object.entries(cfg)
    .filter(([, v]) => v !== '' && v !== 0 && v !== false && !(Array.isArray(v) && !v.length))
    .slice(0, 4)
    .map(([k, v]) => `${k}=${Array.isArray(v) ? v.join(',') : typeof v === 'object' ? JSON.stringify(v) : v}`)
  return parts.join('，') || '-'
}

async function load() {
  if (!store.currentSpaceId) {
    list.value = []
    return
  }
  loading.value = true
  try {
    list.value = (await apiMod.listPlugins(store.currentSpaceId)) || []
  } finally {
    loading.value = false
  }
}

function onKind() {
  form.config = emptyConfig()
  form.configJson = '{}'
  resetSchemaConfig(form.plugin)
  resetRtConfig()
  resetRespConfig()
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.plugin = 'rate-limiting'
  form.enabled = true
  form.config = emptyConfig()
  form.configJson = '{}'
  resetSchemaConfig(form.plugin)
  resetRtConfig()
  resetRespConfig()
  visible.value = true
}

async function openApis(row: PluginItem) {
  if (!store.currentSpaceId) return
  apiTitle.value = `${row.name} 关联的 API`
  apiRows.value = []
  apiVisible.value = true
  apiLoading.value = true
  try {
    apiRows.value = (await apiMod.listPluginApis(store.currentSpaceId, row.id)) || []
  } finally {
    apiLoading.value = false
  }
}

function openEdit(row: PluginItem) {
  editing.value = row
  form.name = row.name
  form.plugin = row.plugin
  form.enabled = row.enabled
  form.config = fillConfig(row)
  form.configJson = fillConfigJson(row)
  resetSchemaConfig(row.plugin, (row.config || {}) as Record<string, unknown>)
  resetRtConfig((row.config || {}) as Record<string, unknown>)
  resetRespConfig((row.config || {}) as Record<string, unknown>)
  visible.value = true
}

async function save() {
  if (!store.currentSpaceId || !form.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  const config = payloadConfig()
  if (config == null) return
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      plugin: form.plugin,
      enabled: form.enabled,
      config,
    }
    if (editing.value) {
      await apiMod.updatePlugin(store.currentSpaceId, editing.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await apiMod.createPlugin(store.currentSpaceId, payload)
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: PluginItem) {
  if (!store.currentSpaceId) return
  await ElMessageBox.confirm(`确认删除 Plugin「${row.name}」？已发布 API 上的该插件会一并移除。`, '提示', { type: 'warning' })
  await apiMod.deletePlugin(store.currentSpaceId, row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  gap: 8px;
  align-items: center;
}
.hint {
  color: #909399;
}
</style>
