<template>
  <div>
    <div class="toolbar">
      <el-button @click="$router.push('/groups')">返回分组</el-button>
      <el-button type="primary" @click="openCreate">新建 API</el-button>
      <el-button @click="openImport">导入 OpenAPI</el-button>
      <el-input
        v-model="nameQuery"
        clearable
        placeholder="按名称搜索"
        style="width: 180px"
        @keyup.enter="applyFilter"
      />
      <el-select v-model="statusQuery" clearable placeholder="状态" style="width: 120px">
        <el-option label="草稿" value="draft" />
        <el-option label="已发布" value="published" />
        <el-option label="已下线" value="offline" />
      </el-select>
      <el-select v-model="sharedQuery" clearable placeholder="分享" style="width: 120px">
        <el-option label="已分享" value="shared" />
        <el-option label="未分享" value="unshared" />
      </el-select>
      <el-button type="primary" @click="applyFilter">搜索</el-button>
      <el-button @click="load">刷新</el-button>
      <el-button type="success" :disabled="!selected.length" :loading="batchBusy" @click="batchPublish">
        发布
      </el-button>
      <el-button type="warning" :disabled="!selected.length" :loading="batchBusy" @click="batchOffline">
        下线
      </el-button>
      <el-button type="danger" :disabled="!selected.length" :loading="batchBusy" @click="batchDelete">
        删除
      </el-button>
    </div>

    <el-table
      ref="tableRef"
      :data="paged"
      row-key="id"
      v-loading="loading"
      stripe
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="48" />
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column label="接入协议" width="160">
        <template #default="{ row }">
          <el-tag
            v-for="proto in protocolsOf(row)"
            :key="proto"
            size="small"
            style="margin: 2px 4px 2px 0"
          >
            {{ proto }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="接入路径" min-width="320">
        <template #default="{ row }">
          <div v-for="p in splitPaths(row.access_path)" :key="p" class="access-url">
            {{ fullAccessPath(row, p) }}
          </div>
          <span v-if="!splitPaths(row.access_path).length">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="access_methods" label="接入方法" width="140" />
      <el-table-column label="认证" width="120">
        <template #default="{ row }">
          <span v-if="row.auth_enabled">{{ row.auth_plugin }}</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="上游" min-width="220">
        <template #default="{ row }">
          {{ upstreamLabel(row) }}
        </template>
      </el-table-column>
      <el-table-column prop="current_version" label="当前版本" width="100" />
      <el-table-column label="分享" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.shared" type="success" size="small">已分享</el-tag>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="680" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="success" @click="onPublish(row)">发布</el-button>
          <el-tooltip :disabled="canOffline(row)" :content="offlineHint(row)" placement="top">
            <span>
              <el-button link type="warning" @click="onOffline(row)" :disabled="!canOffline(row)">下线</el-button>
            </span>
          </el-tooltip>
          <el-tooltip
            :disabled="row.status === 'published' || !!row.shared"
            content="仅已发布的 API 可以分享"
            placement="top"
          >
            <span>
              <el-button
                v-if="!row.shared"
                link
                type="primary"
                :disabled="row.status !== 'published'"
                @click="onShare(row)"
              >
                分享
              </el-button>
              <el-button v-else link type="warning" @click="onUnshare(row)">取消分享</el-button>
            </span>
          </el-tooltip>
          <el-button link type="primary" @click="openVersions(row)">版本</el-button>
          <el-button link type="primary" @click="openPlugins(row)">Plugins</el-button>
          <el-button link type="primary" @click="openConsumers(row)">Consumers</el-button>
          <el-button link type="primary" @click="copyCurl(row)">复制 Curl</el-button>
          <el-tooltip :disabled="canDelete(row)" :content="deleteHint(row)" placement="top">
            <span>
              <el-button link type="danger" @click="onDelete(row)" :disabled="!canDelete(row)">删除</el-button>
            </span>
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>
    <ListPagination
      v-model:page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="pageSizes"
    />

    <el-dialog v-model="pluginVisible" :title="pluginTitle" width="720px">
      <el-table :data="pluginRows" empty-text="暂无绑定插件" stripe>
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="plugin" label="类型" width="160" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">{{ row.enabled ? '启用' : '停用' }}</template>
        </el-table-column>
        <el-table-column label="配置" min-width="220">
          <template #default="{ row }">{{ pluginSummary(row) }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="consumerVisible" :title="consumerTitle" width="640px">
      <el-table :data="consumerRows" empty-text="暂无关联 Consumer" stripe>
        <el-table-column prop="username" label="名称" />
        <el-table-column label="所属空间">
          <template #default="{ row }">{{ row.space?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at || '') }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="visible" body-class="api-dialog-body" :title="editing ? '编辑 API' : '新建 API'" width="640px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="接入路径">
          <el-select
            v-model="form.pathList"
            multiple
            filterable
            allow-create
            default-first-option
            style="width: 100%"
            :placeholder="pathPlaceholder"
          />
          <div v-if="spacePrefix" class="path-hint">
            保存后自动拼接空间前缀 {{ spacePrefix }}
            <template v-if="previewPaths.length">，结果：{{ previewPaths.join('、') }}</template>
          </div>
        </el-form-item>
        <el-form-item label="接入方法">
          <el-select v-model="form.methodList" multiple style="width: 100%">
            <el-option v-for="m in methodOptions" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="接入协议">
          <el-select v-model="form.accessProtocolList" multiple style="width: 100%">
            <el-option v-for="p in protocolOptions" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="接入 Hosts">
          <el-select
            v-model="form.accessHostList"
            multiple
            filterable
            allow-create
            default-first-option
            style="width: 100%"
            placeholder="可选，输入域名后回车，如 example.com"
          />
        </el-form-item>
        <el-form-item label="接入 Headers">
          <div v-for="(h, i) in form.headerList" :key="i" class="header-row">
            <el-input v-model="h.name" placeholder="Header 名" />
            <el-input v-model="h.value" placeholder="值，多个用逗号分隔" />
            <el-button link type="danger" @click="form.headerList.splice(i, 1)">删除</el-button>
          </div>
          <el-button link type="primary" @click="form.headerList.push({ name: '', value: '' })">添加 Header</el-button>
        </el-form-item>
        <el-form-item label="接入 Strip Path">
          <el-switch v-model="form.access_strip_path" />
        </el-form-item>
        <el-form-item label="Request Buffering">
          <el-switch v-model="form.request_buffering" />
        </el-form-item>
        <el-form-item label="Response Buffering">
          <el-switch v-model="form.response_buffering" />
        </el-form-item>
        <el-divider />
        <el-form-item label="后端服务协议">
          <el-select v-model="form.service_protocol" style="width: 100%">
            <el-option v-for="p in protocolOptions" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="后端服务主机">
          <el-radio-group v-model="form.service_host_kind" style="margin-bottom: 8px">
            <el-radio value="direct">IP / 域名</el-radio>
            <el-radio value="upstream">Upstream</el-radio>
          </el-radio-group>
          <el-input v-if="form.service_host_kind === 'direct'" v-model="form.service_host" placeholder="127.0.0.1 或 example.com" />
          <el-select v-else v-model="form.service_upstream_id" filterable style="width: 100%" placeholder="选择本空间 Upstream">
            <el-option v-for="u in upstreams" :key="u.id" :label="u.name" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="后端服务端口">
          <el-input-number v-model="form.service_port" :min="80" :max="65535" :step="1" />
        </el-form-item>
        <el-form-item label="后端服务Path">
          <el-input v-model="form.service_path" placeholder="/" />
        </el-form-item>
        <el-form-item label="Plugins">
          <el-select v-model="form.plugin_ids" multiple filterable style="width: 100%" placeholder="可关联多个本空间 Plugin">
            <el-option v-for="p in plugins" :key="p.id" :label="`${p.name} (${p.plugin})`" :value="p.id" :disabled="!p.enabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用认证">
          <el-switch v-model="form.auth_enabled" :disabled="!!editing" />
        </el-form-item>
        <template v-if="form.auth_enabled">
          <el-form-item label="认证类型">
            <el-select v-model="form.auth_plugin" style="width: 100%" :disabled="!!editing">
              <el-option v-for="p in authPlugins" :key="p" :label="p" :value="p" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.auth_plugin === 'key-auth'" label="Key 名称">
            <el-input v-model="form.auth_key_names" placeholder="apikey，多个用逗号分隔" />
          </el-form-item>
          <el-form-item v-if="form.auth_plugin === 'jwt'" label="Header">
            <el-input v-model="form.auth_header_names" placeholder="authorization" />
          </el-form-item>
          <el-form-item v-if="form.auth_plugin === 'hmac-auth'" label="Clock Skew">
            <el-input-number v-model="form.auth_clock_skew" :min="0" />
          </el-form-item>
          <el-form-item v-if="form.auth_plugin !== 'jwt'" label="隐藏凭证">
            <el-switch v-model="form.auth_hide_credentials" />
          </el-form-item>
        </template>
        <el-collapse>
          <el-collapse-item title="后端服务重试与超时" name="advanced">
            <el-form-item label="Retries">
              <el-input-number v-model="form.service_retries" :min="0" :step="1" />
            </el-form-item>
            <el-form-item label="连接超时">
              <el-input-number v-model="form.service_connect_timeout" :min="0" :step="1000" />
              <span class="unit">毫秒</span>
            </el-form-item>
            <el-form-item label="写超时">
              <el-input-number v-model="form.service_write_timeout" :min="0" :step="1000" />
              <span class="unit">毫秒</span>
            </el-form-item>
            <el-form-item label="读超时">
              <el-input-number v-model="form.service_read_timeout" :min="0" :step="1000" />
              <span class="unit">毫秒</span>
            </el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="versionVisible" title="API 版本" width="720px">
      <el-table :data="versions" v-loading="versionLoading">
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column label="发布时间" width="180">
          <template #default="{ row }">{{ formatTime(row.published_at) }}</template>
        </el-table-column>
        <el-table-column label="配置摘要">
          <template #default="{ row }">
            {{ summarize(row.config_snapshot) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button
              link
              type="primary"
              :disabled="row.version === currentApi?.current_version && currentApi?.status === 'published'"
              @click="onSwitch(row.version)"
            >
              切换
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="detailVisible" :title="detailTitle" width="720px" append-to-body>
      <el-descriptions v-if="detailSnap" :column="1" border>
        <el-descriptions-item label="版本">{{ detailVersion?.version }}</el-descriptions-item>
        <el-descriptions-item label="发布时间">{{ formatTime(detailVersion?.published_at || '') }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ detailSnap.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="接入路径">
          <el-tag
            v-for="p in splitPaths(detailSnap.access_path)"
            :key="p"
            size="small"
            style="margin: 2px 4px 2px 0"
          >
            {{ p }}
          </el-tag>
          <span v-if="!splitPaths(detailSnap.access_path).length">-</span>
        </el-descriptions-item>
        <el-descriptions-item label="接入方法">{{ detailSnap.access_methods || '-' }}</el-descriptions-item>
        <el-descriptions-item label="接入协议">{{ detailSnap.access_protocols || 'http' }}</el-descriptions-item>
        <el-descriptions-item label="接入 Hosts">{{ detailSnap.access_hosts || '-' }}</el-descriptions-item>
        <el-descriptions-item label="接入 Headers">{{ formatHeaders(detailSnap.access_headers) }}</el-descriptions-item>
        <el-descriptions-item label="接入 Strip Path">{{ detailSnap.access_strip_path ? '开启' : '关闭' }}</el-descriptions-item>
        <el-descriptions-item label="Request Buffering">{{ detailSnap.request_buffering !== false ? '开启' : '关闭' }}</el-descriptions-item>
        <el-descriptions-item label="Response Buffering">{{ detailSnap.response_buffering !== false ? '开启' : '关闭' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务协议">{{ detailSnap.service_protocol || '-' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务主机">{{ detailSnap.kong_host || detailSnap.service_host || detailSnap.upstream_url || '-' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务端口">{{ detailSnap.service_port || '-' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务Path">{{ detailSnap.service_path || '-' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务Retries">{{ detailSnap.service_retries ?? '-' }}</el-descriptions-item>
        <el-descriptions-item label="后端服务连接超时">{{ detailSnap.service_connect_timeout ?? '-' }} ms</el-descriptions-item>
        <el-descriptions-item label="后端服务写超时">{{ detailSnap.service_write_timeout ?? '-' }} ms</el-descriptions-item>
        <el-descriptions-item label="后端服务读超时">{{ detailSnap.service_read_timeout ?? '-' }} ms</el-descriptions-item>
      </el-descriptions>
      <div v-if="detailSnap" class="version-plugins">
        <div class="version-plugins-title">关联插件</div>
        <el-table :data="detailPlugins" empty-text="该版本未关联插件" stripe>
          <el-table-column prop="name" label="插件名字" min-width="140" />
          <el-table-column prop="plugin" label="插件类型" width="180" />
          <el-table-column label="插件配置" min-width="260">
            <template #default="{ row }">{{ formatPluginConfig(row.config) }}</template>
          </el-table-column>
        </el-table>
      </div>
      <el-empty v-else description="无法解析该版本配置" />
    </el-dialog>

    <el-dialog
      v-model="importVisible"
      title="导入 OpenAPI"
      width="820px"
      body-class="api-dialog-body"
      @closed="resetImport"
    >
      <el-form label-width="120px">
        <el-form-item label="OpenAPI 文件" required>
          <el-upload
            ref="importUploadRef"
            :auto-upload="false"
            :limit="1"
            accept=".json,.yaml,.yml,application/json,application/x-yaml,text/yaml"
            :on-change="onImportFileChange"
            :on-remove="onImportFileRemove"
            :on-exceed="onImportExceed"
          >
            <el-button>选择文件</el-button>
            <template #tip>
              <div class="el-upload__tip">支持 OpenAPI 3 / Swagger 2 的 JSON、YAML 文件（最大 8MB）</div>
            </template>
          </el-upload>
        </el-form-item>
        <el-form-item label="后端服务协议">
          <el-select v-model="importForm.service_protocol" clearable placeholder="默认读取文档 servers" style="width: 100%">
            <el-option v-for="p in protocolOptions" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="后端服务主机">
          <el-input v-model="importForm.service_host" placeholder="可选，覆盖文档中的 host" />
        </el-form-item>
        <el-form-item label="后端服务端口">
          <el-input-number v-model="importForm.service_port" :min="0" :max="65535" :step="1" />
          <span class="unit">0 表示使用文档默认</span>
        </el-form-item>
        <el-form-item label="后端服务Path">
          <el-input v-model="importForm.service_path" placeholder="可选，如 /v1" />
        </el-form-item>
        <div v-if="spacePrefix" class="path-hint import-hint">
          导入后接入路径将自动拼接空间前缀 {{ spacePrefix }}；当前分组内同名 API 会更新而非新建。
        </div>
        <div v-else class="path-hint import-hint">
          当前分组内同名 API 会更新而非新建。
        </div>
      </el-form>

      <div v-if="importPreview.length" class="import-preview">
        <div class="import-preview-title">预览（共 {{ importPreview.length }} 条）</div>
        <el-table :data="importPreview" stripe max-height="280" size="small">
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column label="操作" width="90">
            <template #default="{ row }">
              <el-tag v-if="row.action === 'update'" type="warning" size="small">更新</el-tag>
              <el-tag v-else type="success" size="small">新建</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="接入路径" min-width="200">
            <template #default="{ row }">
              <div>{{ row.access_path }}</div>
              <div v-if="row.access_path_prefixed !== row.access_path" class="path-hint">
                → {{ row.access_path_prefixed }}
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="access_methods" label="方法" width="140" />
          <el-table-column label="上游" min-width="180">
            <template #default="{ row }">
              {{ row.service_protocol }}://{{ row.service_host }}:{{ row.service_port }}{{ row.service_path }}
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <el-button @click="importVisible = false">取消</el-button>
        <el-button :disabled="!importFile" :loading="importParsing" @click="parseImport">解析预览</el-button>
        <el-button type="primary" :disabled="!importPreview.length" :loading="importing" @click="confirmImport">
          确认导入
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TableInstance, UploadFile, UploadInstance, UploadRawFile } from 'element-plus'
import type { ApiItem, ApiVersion, ConsumerItem, PluginItem, UpstreamItem } from '@/types'
import * as apiMod from '@/api/api'
import type { ImportOpenAPIItem } from '@/api/api'
import { useUserStore } from '@/stores/user'
import ListPagination from '@/components/ListPagination.vue'
import { usePagination } from '@/composables/usePagination'

const route = useRoute()
const store = useUserStore()
const gid = Number(route.params.gid)
const spacePrefix = computed(() => (store.currentSpace?.prefix || '').replace(/\/$/, ''))
const list = ref<ApiItem[]>([])
const nameQuery = ref('')
const statusQuery = ref('')
const sharedQuery = ref('')
const filtered = computed(() => {
  const name = nameQuery.value.trim().toLowerCase()
  const status = statusQuery.value
  const shared = sharedQuery.value
  return list.value.filter((row) => {
    if (name && !row.name.toLowerCase().includes(name)) return false
    if (status && row.status !== status) return false
    if (shared === 'shared' && !row.shared) return false
    if (shared === 'unshared' && row.shared) return false
    return true
  })
})
const { page, pageSize, total, paged, pageSizes, resetPage } = usePagination(filtered)
const tableRef = ref<TableInstance>()
const selected = ref<ApiItem[]>([])
const batchBusy = ref(false)

watch([nameQuery, statusQuery, sharedQuery], () => {
  resetPage()
})

function applyFilter() {
  resetPage()
}
const versions = ref<ApiVersion[]>([])
const loading = ref(false)
const saving = ref(false)
const versionLoading = ref(false)
const visible = ref(false)
const versionVisible = ref(false)
const consumerVisible = ref(false)
const consumerTitle = ref('Consumers')
const consumerRows = ref<ConsumerItem[]>([])
const pluginVisible = ref(false)
const pluginTitle = ref('Plugins')
const pluginRows = ref<PluginItem[]>([])
const detailVisible = ref(false)
const editing = ref<ApiItem | null>(null)
const currentApi = ref<ApiItem | null>(null)
const detailVersion = ref<ApiVersion | null>(null)
const authPlugins = ['key-auth', 'basic-auth', 'jwt', 'hmac-auth']
const methodOptions = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']
const protocolOptions = ['http', 'https', 'grpc', 'grpcs']
const upstreams = ref<UpstreamItem[]>([])
const plugins = ref<PluginItem[]>([])

const importVisible = ref(false)
const importParsing = ref(false)
const importing = ref(false)
const importFile = ref<File | null>(null)
const importPreview = ref<ImportOpenAPIItem[]>([])
const importUploadRef = ref<UploadInstance>()
const importForm = reactive({
  service_protocol: '' as string,
  service_host: '',
  service_port: 0,
  service_path: '',
})

const form = reactive({
  name: '',
  pathList: ['/api/demo'] as string[],
  methodList: ['GET'] as string[],
  accessProtocolList: ['http'] as string[],
  accessHostList: [] as string[],
  headerList: [] as { name: string; value: string }[],
  service_protocol: 'http',
  service_host_kind: 'direct',
  service_host: '',
  service_upstream_id: undefined as number | undefined,
  service_port: 80,
  service_path: '/',
  service_retries: 5,
  service_connect_timeout: 60000,
  service_write_timeout: 60000,
  service_read_timeout: 60000,
  access_strip_path: true,
  request_buffering: true,
  response_buffering: true,
  plugin_ids: [] as number[],
  auth_enabled: false,
  auth_plugin: 'key-auth',
  auth_key_names: 'apikey',
  auth_header_names: 'authorization',
  auth_hide_credentials: false,
  auth_clock_skew: 300,
})

function authConfig() {
  if (!form.auth_enabled) return {}
  if (form.auth_plugin === 'key-auth') {
    return {
      key_names: form.auth_key_names,
      hide_credentials: form.auth_hide_credentials,
      key_in_header: true,
      key_in_query: true,
      key_in_body: false,
    }
  }
  if (form.auth_plugin === 'jwt') {
    return { header_names: form.auth_header_names }
  }
  if (form.auth_plugin === 'hmac-auth') {
    return { hide_credentials: form.auth_hide_credentials, clock_skew: form.auth_clock_skew }
  }
  return { hide_credentials: form.auth_hide_credentials }
}

const pathPlaceholder = computed(() =>
  spacePrefix.value
    ? `输入相对路径后回车，将自动加上 ${spacePrefix.value}`
    : '输入路径后回车，可添加多个，如 /api/v1 /api/v2',
)

const previewPaths = computed(() =>
  form.pathList
    .map((p) => normalizePath(p))
    .filter(Boolean)
    .map((p) => applyPrefix(p)),
)

function normalizePath(raw: string) {
  const t = raw.trim()
  if (!t) return ''
  return t.startsWith('/') ? t : `/${t}`
}

function applyPrefix(path: string) {
  const prefix = spacePrefix.value
  if (!prefix || path === prefix || path.startsWith(`${prefix}/`)) return path
  return `${prefix}${path}`
}

function hasConsumers(row: ApiItem) {
  return (row.consumers || []).length > 0
}

function canOffline(row: ApiItem) {
  return row.status === 'published' && !hasConsumers(row)
}

function offlineHint(row: ApiItem) {
  if (row.status !== 'published') return '仅已发布的 API 可以下线'
  if (hasConsumers(row)) return '已关联 Consumer 的 API 不允许下线'
  return ''
}

function canDelete(row: ApiItem) {
  return row.status !== 'published' && !hasConsumers(row)
}

function deleteHint(row: ApiItem) {
  if (row.status === 'published') return '已发布的 API 不允许删除'
  if (hasConsumers(row)) return '已关联 Consumer 的 API 不允许删除'
  return ''
}

function onSelectionChange(rows: ApiItem[]) {
  selected.value = rows
}

async function runBatch(
  action: string,
  targets: ApiItem[],
  skipped: number,
  confirmType: '' | 'warning' | 'error' | undefined,
  run: (row: ApiItem) => Promise<unknown>,
) {
  if (!targets.length) {
    ElMessage.warning(
      skipped > 0 ? `已跳过 ${skipped} 个不符合条件的 API，没有可${action}的项` : `请先勾选 API`,
    )
    return
  }
  const skipTip = skipped > 0 ? `将跳过 ${skipped} 个不符合条件的项。` : ''
  await ElMessageBox.confirm(`确认批量${action} ${targets.length} 个 API？${skipTip}`, `批量${action}`, {
    type: confirmType,
  })
  batchBusy.value = true
  let ok = 0
  let fail = 0
  try {
    for (const row of targets) {
      try {
        await run(row)
        ok++
      } catch {
        fail++
      }
    }
    const parts = [`成功 ${ok}`]
    if (fail) parts.push(`失败 ${fail}`)
    if (skipped) parts.push(`跳过 ${skipped}`)
    ElMessage.success(`批量${action}完成：${parts.join('，')}`)
    selected.value = []
    tableRef.value?.clearSelection()
    await load()
  } finally {
    batchBusy.value = false
  }
}

async function batchPublish() {
  await runBatch('发布', selected.value, 0, undefined, (row) => apiMod.publishApi(row.id))
}

async function batchOffline() {
  const all = selected.value
  const targets = all.filter(canOffline)
  await runBatch('下线', targets, all.length - targets.length, 'warning', (row) => apiMod.offlineApi(row.id))
}

async function batchDelete() {
  const all = selected.value
  const targets = all.filter(canDelete)
  await runBatch('删除', targets, all.length - targets.length, 'warning', (row) => apiMod.deleteApi(row.id))
}

function protocolsOf(row: ApiItem): string[] {
  const list = (row.access_protocols || 'http')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  return list.length ? list : ['http']
}

function gatewayDomain(row: ApiItem): string {
  return row.group?.gateway?.domain || ''
}

function spacePrefixOf(row: ApiItem): string {
  return (row.group?.space?.prefix || '').replace(/\/$/, '')
}

function fullAccessPath(row: ApiItem, path: string): string {
  let p = path.startsWith('/') ? path : `/${path}`
  const prefix = spacePrefixOf(row)
  if (prefix && p !== prefix && !p.startsWith(`${prefix}/`)) {
    p = `${prefix}${p}`
  }
  return `${gatewayDomain(row)}${p}`
}

function accessPathOnly(row: ApiItem, path: string): string {
  let p = path.startsWith('/') ? path : `/${path}`
  const prefix = spacePrefixOf(row)
  if (prefix && p !== prefix && !p.startsWith(`${prefix}/`)) {
    p = `${prefix}${p}`
  }
  return p
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`
}

function firstConfigName(value: unknown, fallback: string): string {
  if (Array.isArray(value) && value.length) return String(value[0]).trim() || fallback
  if (typeof value === 'string' && value.trim()) {
    return value.split(/[,;\s]+/).map((s) => s.trim()).filter(Boolean)[0] || fallback
  }
  return fallback
}

function authCurlFlags(row: ApiItem): string[] {
  if (!row.auth_enabled || !row.auth_plugin) return []
  const cfg = row.auth_config || {}
  switch (row.auth_plugin) {
    case 'key-auth': {
      const key = firstConfigName(cfg.key_names, 'apikey')
      return [`  -H ${shellQuote(`${key}: YOUR_API_KEY`)}`]
    }
    case 'basic-auth':
      return [`  -u ${shellQuote('USERNAME:PASSWORD')}`]
    case 'jwt': {
      const header = firstConfigName(cfg.header_names, 'Authorization')
      const value = header.toLowerCase() === 'authorization' ? 'Bearer YOUR_JWT' : 'YOUR_JWT'
      return [`  -H ${shellQuote(`${header}: ${value}`)}`]
    }
    case 'hmac-auth':
      return [
        `  -H ${shellQuote('Date: Tue, 07 Jun 2014 20:51:35 GMT')}`,
        `  -H ${shellQuote('Authorization: hmac username="USERNAME", algorithm="hmac-sha256", headers="date request-line", signature="SIGNATURE"')}`,
      ]
    default:
      return []
  }
}

function buildCurl(row: ApiItem): string {
  const protocol = protocolsOf(row)[0] || 'http'
  const method =
    (row.access_methods || 'GET')
      .split(/[,;\s]+/)
      .map((s) => s.trim())
      .filter(Boolean)[0] || 'GET'
  const path = accessPathOnly(row, splitPaths(row.access_path)[0] || '/')
  const domain = gatewayDomain(row).replace(/^(https?:)?\/\//i, '').replace(/\/$/, '')
  const accessHosts = (row.access_hosts || '')
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
  const urlHost = domain || accessHosts[0] || 'localhost'
  const url = `${protocol}://${urlHost}${path}`

  const lines = [`curl -X ${method} ${shellQuote(url)}`]
  if (domain && accessHosts[0] && accessHosts[0].toLowerCase() !== domain.toLowerCase()) {
    lines.push(`  -H ${shellQuote(`Host: ${accessHosts[0]}`)}`)
  }
  const headers = row.access_headers || {}
  for (const [name, values] of Object.entries(headers)) {
    if (!name.trim()) continue
    for (const value of values || []) {
      lines.push(`  -H ${shellQuote(`${name}: ${value}`)}`)
    }
  }
  lines.push(...authCurlFlags(row))
  return lines.join(' \\\n')
}

async function copyCurl(row: ApiItem) {
  const text = buildCurl(row)
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const el = document.createElement('textarea')
      el.value = text
      el.style.position = 'fixed'
      el.style.left = '-9999px'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    }
    ElMessage.success('Curl 已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

function splitPaths(raw: string): string[] {
  return (raw || '')
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function statusLabel(s: string) {
  return ({ draft: '草稿', published: '已发布', offline: '已下线' } as Record<string, string>)[s] || s
}
function statusType(s: string) {
  return ({ draft: 'info', published: 'success', offline: 'warning' } as Record<string, string>)[s] || 'info'
}

interface VersionSnapshot {
  name: string
  access_path: string
  access_methods: string
  access_protocols: string
  access_hosts: string
  access_headers: Record<string, string[]>
  upstream_url: string
  service_protocol: string
  service_host: string
  kong_host: string
  service_port: number
  service_path: string
  service_retries: number
  service_connect_timeout: number
  service_write_timeout: number
  service_read_timeout: number
  access_strip_path: boolean
  request_buffering?: boolean
  response_buffering?: boolean
  plugins: PluginItem[]
  pluginsRecorded: boolean
}

const detailSnap = computed(() =>
  detailVersion.value ? parseSnapshot(detailVersion.value.config_snapshot) : null,
)
const detailTitle = computed(() =>
  detailVersion.value ? `版本 ${detailVersion.value.version} 详情` : '版本详情',
)
const detailPlugins = computed(() => {
  if (!detailSnap.value) return []
  if (detailSnap.value.pluginsRecorded) return detailSnap.value.plugins
  if (detailVersion.value?.version === currentApi.value?.current_version) {
    return currentApi.value?.plugins || []
  }
  return []
})

function parseSnapshot(snap: Record<string, unknown> | string): VersionSnapshot | null {
  try {
    const obj = (typeof snap === 'string' ? JSON.parse(snap) : snap) as Record<string, unknown>
    return {
      name: String(obj.name || ''),
      access_path: String(obj.access_path || obj.path || ''),
      access_methods: String(obj.access_methods || obj.methods || ''),
      access_protocols: String(obj.access_protocols || ''),
      access_hosts: String(obj.access_hosts || ''),
      access_headers: (obj.access_headers && typeof obj.access_headers === 'object'
        ? obj.access_headers
        : {}) as Record<string, string[]>,
      upstream_url: String(obj.upstream_url || ''),
      service_protocol: String(obj.service_protocol || obj.protocol || ''),
      service_host: String(obj.service_host || obj.host || ''),
      kong_host: String(obj.kong_host || ''),
      service_port: Number(obj.service_port || obj.port || 0),
      service_path: String(obj.service_path || ''),
      service_retries: Number(obj.service_retries ?? obj.retries ?? 0),
      service_connect_timeout: Number(obj.service_connect_timeout ?? obj.connect_timeout ?? 0),
      service_write_timeout: Number(obj.service_write_timeout ?? obj.write_timeout ?? 0),
      service_read_timeout: Number(obj.service_read_timeout ?? obj.read_timeout ?? 0),
      access_strip_path: readStripPath(obj),
      request_buffering: readBuffering(obj, 'request_buffering'),
      response_buffering: readBuffering(obj, 'response_buffering'),
      plugins: parsePluginSnapshots(obj.plugins),
      pluginsRecorded: Object.prototype.hasOwnProperty.call(obj, 'plugins'),
    }
  } catch {
    return null
  }
}

function parsePluginSnapshots(raw: unknown): PluginItem[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item, index) => {
    const row = (item && typeof item === 'object' ? item : {}) as Record<string, unknown>
    return {
      id: Number(row.id || index + 1),
      space_id: Number(row.space_id || 0),
      name: String(row.name || ''),
      plugin: String(row.plugin || ''),
      config: (row.config && typeof row.config === 'object' ? row.config : {}) as Record<string, unknown>,
      enabled: row.enabled !== false,
    }
  })
}

function formatPluginConfig(config?: Record<string, unknown>) {
  const entries = Object.entries(config || {}).filter(([, v]) => v !== '' && v !== undefined)
  if (!entries.length) return '-'
  try {
    return JSON.stringify(Object.fromEntries(entries))
  } catch {
    return '-'
  }
}

function summarize(snap: Record<string, unknown> | string) {
  const obj = parseSnapshot(snap)
  if (!obj) return String(snap)
  const dest = obj.kong_host || obj.service_host || obj.upstream_url
  return `${obj.access_methods} ${obj.access_path} -> ${obj.service_protocol}://${dest}:${obj.service_port}${obj.service_path}`
}

function readStripPath(obj: Record<string, unknown>) {
  if (typeof obj.access_strip_path === 'boolean') return obj.access_strip_path
  if (typeof obj.strip_path === 'boolean') return obj.strip_path
  return true
}

function readBuffering(obj: Record<string, unknown>, key: string) {
  if (typeof obj[key] === 'boolean') return obj[key] as boolean
  return true
}

function formatHeaders(headers?: Record<string, string[]>) {
  const entries = Object.entries(headers || {}).filter(([name]) => name)
  if (!entries.length) return '-'
  return entries.map(([name, values]) => `${name}: ${(values || []).join(',')}`).join('；')
}

function headersToRows(headers?: Record<string, string[]>) {
  return Object.entries(headers || {}).map(([name, values]) => ({
    name,
    value: (values || []).join(','),
  }))
}

function rowsToHeaders(rows: { name: string; value: string }[]) {
  const out: Record<string, string[]> = {}
  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue
    const values = row.value
      .split(/[,，]/)
      .map((s) => s.trim())
      .filter(Boolean)
    if (!values.length) continue
    out[name] = [...(out[name] || []), ...values]
  }
  return out
}

function upstreamLabel(row: ApiItem) {
  if (row.service_host) {
    return `${row.service_protocol || 'http'}://${row.service_host}:${row.service_port}${row.service_path || '/'}`
  }
  return row.upstream_url || '-'
}

function resetService() {
  form.service_protocol = 'http'
  form.service_host_kind = 'direct'
  form.service_host = 'httpbin.org'
  form.service_upstream_id = undefined
  form.service_port = 80
  form.service_path = '/'
  form.service_retries = 5
  form.service_connect_timeout = 60000
  form.service_write_timeout = 60000
  form.service_read_timeout = 60000
}

function formatTime(raw: string) {
  if (!raw) return '-'
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return raw
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

function openConsumers(row: ApiItem) {
  consumerTitle.value = `${row.name} 的 Consumers`
  consumerRows.value = row.consumers || []
  consumerVisible.value = true
}

function pluginSummary(row: PluginItem) {
  const cfg = row.config || {}
  const parts = Object.entries(cfg)
    .filter(([, v]) => v !== '' && v !== 0 && v !== false && !(Array.isArray(v) && !v.length))
    .slice(0, 4)
    .map(([k, v]) => `${k}=${Array.isArray(v) ? v.join(',') : v}`)
  return parts.join('，') || '-'
}

function openPlugins(row: ApiItem) {
  pluginTitle.value = `${row.name} 的 Plugins`
  pluginRows.value = row.plugins || []
  pluginVisible.value = true
}

function openDetail(row: ApiVersion) {
  detailVersion.value = row
  detailVisible.value = true
}

async function load() {
  loading.value = true
  try {
    list.value = (await apiMod.listApis(gid)) || []
    selected.value = []
    tableRef.value?.clearSelection()
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.pathList = ['/api/demo']
  form.methodList = ['GET']
  form.accessProtocolList = ['http']
  form.accessHostList = []
  form.headerList = []
  resetService()
  form.access_strip_path = true
  form.request_buffering = true
  form.response_buffering = true
  form.plugin_ids = []
  form.auth_enabled = false
  form.auth_plugin = 'key-auth'
  form.auth_key_names = 'apikey'
  form.auth_header_names = 'authorization'
  form.auth_hide_credentials = false
  form.auth_clock_skew = 300
  visible.value = true
}

function openImport() {
  resetImport()
  importVisible.value = true
}

function resetImport() {
  importFile.value = null
  importPreview.value = []
  importForm.service_protocol = ''
  importForm.service_host = ''
  importForm.service_port = 0
  importForm.service_path = ''
  importUploadRef.value?.clearFiles()
}

function onImportFileChange(file: UploadFile) {
  importPreview.value = []
  importFile.value = (file.raw as UploadRawFile) || null
}

function onImportFileRemove() {
  importFile.value = null
  importPreview.value = []
}

function onImportExceed(files: File[]) {
  const file = files[0]
  if (!file) return
  importUploadRef.value?.clearFiles()
  importFile.value = file
  importPreview.value = []
  // Re-bind for display: el-upload keeps the new file when we clear then start.
  const raw = file as UploadRawFile
  if (!raw.uid) raw.uid = Date.now()
  importUploadRef.value?.handleStart(raw)
}

function importOptions() {
  return {
    service_protocol: importForm.service_protocol || undefined,
    service_host: importForm.service_host.trim() || undefined,
    service_port: importForm.service_port > 0 ? importForm.service_port : undefined,
    service_path: importForm.service_path.trim() || undefined,
  }
}

async function parseImport() {
  if (!importFile.value) {
    ElMessage.warning('请先选择 OpenAPI 文件')
    return
  }
  importParsing.value = true
  try {
    const result = await apiMod.importOpenAPI(gid, importFile.value, {
      ...importOptions(),
      dry_run: true,
    })
    importPreview.value = result.items || []
    if (!importPreview.value.length) {
      ElMessage.warning('文档中未解析到可用的 API 路径')
    } else {
      const updateCount = importPreview.value.filter((i) => i.action === 'update').length
      const createCount = importPreview.value.length - updateCount
      ElMessage.success(
        `解析到 ${importPreview.value.length} 条（新建 ${createCount}，更新同名 ${updateCount}）`,
      )
    }
  } finally {
    importParsing.value = false
  }
}

async function confirmImport() {
  if (!importFile.value) {
    ElMessage.warning('请先选择 OpenAPI 文件')
    return
  }
  if (!importPreview.value.length) {
    ElMessage.warning('请先解析预览')
    return
  }
  const updateCount = importPreview.value.filter((i) => i.action === 'update').length
  const createCount = importPreview.value.length - updateCount
  const parts = [
    createCount > 0 ? `新建 ${createCount} 条` : '',
    updateCount > 0 ? `更新同名 ${updateCount} 条` : '',
  ].filter(Boolean)
  await ElMessageBox.confirm(
    `确认导入？将${parts.join('，')}；接入路径会自动加上空间前缀。`,
    '导入确认',
  )
  importing.value = true
  try {
    const result = await apiMod.importOpenAPI(gid, importFile.value, importOptions())
    const created = result.created?.length || 0
    const updated = result.updated?.length || 0
    const fail = result.failed?.length || 0
    if (fail > 0) {
      const first = result.failed?.[0]
      ElMessage.warning(
        `新建 ${created}、更新 ${updated}、失败 ${fail}${first ? `：${first.name} ${first.error}` : ''}`,
      )
    } else {
      ElMessage.success(`导入完成：新建 ${created} 条，更新 ${updated} 条`)
    }
    importVisible.value = false
    await load()
  } finally {
    importing.value = false
  }
}

function openEdit(row: ApiItem) {
  editing.value = row
  form.name = row.name
  form.pathList = splitPaths(row.access_path)
  form.methodList = row.access_methods.split(',').map((s) => s.trim()).filter(Boolean)
  form.accessProtocolList = (row.access_protocols || 'http').split(',').map((s) => s.trim()).filter(Boolean)
  form.accessHostList = (row.access_hosts || '').split(/[,;\n]/).map((s) => s.trim()).filter(Boolean)
  form.headerList = headersToRows(row.access_headers)
  form.service_protocol = row.service_protocol || 'http'
  form.service_host_kind = row.service_host_kind || 'direct'
  form.service_host = row.service_host || ''
  form.service_upstream_id = row.service_upstream_id
  form.service_port = row.service_port || 80
  form.service_path = row.service_path || '/'
  form.service_retries = row.service_retries ?? 5
  form.service_connect_timeout = row.service_connect_timeout ?? 60000
  form.service_write_timeout = row.service_write_timeout ?? 60000
  form.service_read_timeout = row.service_read_timeout ?? 60000
  form.access_strip_path = row.access_strip_path
  form.request_buffering = row.request_buffering !== false
  form.response_buffering = row.response_buffering !== false
  form.plugin_ids = (row.plugins || []).map((p) => p.id)
  form.auth_enabled = !!row.auth_enabled
  form.auth_plugin = row.auth_plugin || 'key-auth'
  const cfg = row.auth_config || {}
  form.auth_key_names = Array.isArray(cfg.key_names) ? cfg.key_names.join(',') : 'apikey'
  form.auth_header_names = Array.isArray(cfg.header_names) ? cfg.header_names.join(',') : 'authorization'
  form.auth_hide_credentials = !!cfg.hide_credentials
  form.auth_clock_skew = Number(cfg.clock_skew || 300)
  visible.value = true
}

async function save() {
  if (!form.name || !form.pathList.length || !form.methodList.length || !form.accessProtocolList.length) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (form.service_host_kind === 'direct' && !form.service_host.trim()) {
    ElMessage.warning('请填写后端服务主机')
    return
  }
  if (form.service_host_kind === 'upstream' && !form.service_upstream_id) {
    ElMessage.warning('请选择 Upstream')
    return
  }
  if (!form.service_path.startsWith('/') || /[,;\n]/.test(form.service_path)) {
    ElMessage.warning('后端服务Path 只能填一个，且以 / 开头')
    return
  }
  const paths = form.pathList.map((p) => normalizePath(p)).filter(Boolean)
  if (!paths.length) {
    ElMessage.warning('请至少添加一个路径')
    return
  }
  const payload = {
    name: form.name,
    access_path: paths.join(','),
    access_methods: form.methodList.join(','),
    access_protocols: form.accessProtocolList.join(','),
    access_hosts: form.accessHostList.map((h) => h.trim()).filter(Boolean).join(','),
    access_headers: rowsToHeaders(form.headerList),
    service_protocol: form.service_protocol,
    service_host_kind: form.service_host_kind,
    service_host: form.service_host_kind === 'direct' ? form.service_host.trim() : '',
    service_upstream_id: form.service_host_kind === 'upstream' ? form.service_upstream_id : undefined,
    service_port: form.service_port,
    service_path: form.service_path,
    service_retries: form.service_retries,
    service_connect_timeout: form.service_connect_timeout,
    service_write_timeout: form.service_write_timeout,
    service_read_timeout: form.service_read_timeout,
    access_strip_path: form.access_strip_path,
    request_buffering: form.request_buffering,
    response_buffering: form.response_buffering,
    plugin_ids: form.plugin_ids,
    auth_enabled: form.auth_enabled,
    auth_plugin: form.auth_enabled ? form.auth_plugin : '',
    auth_config: authConfig(),
  }
  saving.value = true
  try {
    if (editing.value) {
      await apiMod.updateApi(editing.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await apiMod.createApi(gid, payload)
      ElMessage.success('创建成功')
    }
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onPublish(row: ApiItem) {
  await ElMessageBox.confirm(`确认发布 API「${row.name}」到 Kong？`, '发布确认')
  await apiMod.publishApi(row.id)
  ElMessage.success('发布成功，已生成新版本')
  await load()
}

async function onOffline(row: ApiItem) {
  await ElMessageBox.confirm(`确认下线 API「${row.name}」？`, '下线确认', { type: 'warning' })
  await apiMod.offlineApi(row.id)
  ElMessage.success('已下线')
  await load()
}

async function onShare(row: ApiItem) {
  await ElMessageBox.confirm(`确认将 API「${row.name}」分享到 API 市场？`, '分享确认')
  await apiMod.shareApi(row.id)
  ElMessage.success('已分享到 API 市场')
  await load()
}

async function onUnshare(row: ApiItem) {
  await ElMessageBox.confirm(`确认取消分享 API「${row.name}」？取消后将从 API 市场移除。`, '取消分享', {
    type: 'warning',
  })
  await apiMod.unshareApi(row.id)
  ElMessage.success('已取消分享')
  await load()
}

async function onDelete(row: ApiItem) {
  await ElMessageBox.confirm(`确认删除 API「${row.name}」？`, '提示', { type: 'warning' })
  await apiMod.deleteApi(row.id)
  ElMessage.success('已删除')
  await load()
}

async function openVersions(row: ApiItem) {
  currentApi.value = row
  versionVisible.value = true
  versionLoading.value = true
  try {
    versions.value = (await apiMod.listVersions(row.id)) || []
  } finally {
    versionLoading.value = false
  }
}

async function onSwitch(version: string) {
  if (!currentApi.value) return
  await ElMessageBox.confirm(`切换到版本 ${version}？将下线当前配置并重新发布。`, '版本切换')
  await apiMod.switchVersion(currentApi.value.id, version)
  ElMessage.success('切换成功')
  versionVisible.value = false
  await load()
}

onMounted(async () => {
  if (store.currentSpaceId) {
    upstreams.value = (await apiMod.listUpstreams(store.currentSpaceId)) || []
    plugins.value = (await apiMod.listPlugins(store.currentSpaceId)) || []
  }
  await load()
})
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.access-url {
  display: flex;
  align-items: center;
  gap: 0;
  line-height: 28px;
  word-break: break-all;
}
.path-hint {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
.import-hint {
  margin: 0 0 12px 120px;
}
.import-preview {
  margin-top: 8px;
}
.import-preview-title {
  margin-bottom: 8px;
  font-weight: 600;
}
.unit {
  margin-left: 8px;
  color: #909399;
}
.header-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  width: 100%;
}
.version-plugins {
  margin-top: 16px;
}
.version-plugins-title {
  margin-bottom: 8px;
  font-weight: 600;
}
</style>

<style>
.api-dialog-body {
  max-height: 52vh;
  overflow: auto;
}
</style>
