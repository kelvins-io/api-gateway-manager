import http, { request } from './http'
import type { ApiGroup, ApiItem, ApiVersion, ConsumerItem, PluginItem, UpstreamItem } from '@/types'

export function listGroups(spaceId: number) {
  return request<ApiGroup[]>(() => http.get(`/spaces/${spaceId}/groups`))
}

export function createGroup(spaceId: number, data: { name: string; gateway_id: number }) {
  return request<ApiGroup>(() => http.post(`/spaces/${spaceId}/groups`, data))
}

export function updateGroup(gid: number, data: { name?: string }) {
  return request<ApiGroup>(() => http.put(`/groups/${gid}`, data))
}

export function deleteGroup(gid: number) {
  return request<null>(() => http.delete(`/groups/${gid}`))
}

export function listSpaceApis(spaceId: number) {
  return request<ApiItem[]>(() => http.get(`/spaces/${spaceId}/apis`))
}

export function listApis(gid: number) {
  return request<ApiItem[]>(() => http.get(`/groups/${gid}/apis`))
}

export function createApi(gid: number, data: Partial<ApiItem>) {
  return request<ApiItem>(() => http.post(`/groups/${gid}/apis`, data))
}

export interface ImportOpenAPIItem {
  name: string
  access_path: string
  access_path_prefixed: string
  access_methods: string
  access_protocols: string
  service_protocol: string
  service_host: string
  service_port: number
  service_path: string
}

export interface ImportOpenAPIResult {
  items: ImportOpenAPIItem[]
  created?: ApiItem[]
  failed?: { name: string; path: string; error: string }[]
  total: number
}

export function importOpenAPI(
  gid: number,
  file: File,
  opts?: {
    dry_run?: boolean
    service_protocol?: string
    service_host?: string
    service_port?: number
    service_path?: string
  },
) {
  const form = new FormData()
  form.append('file', file)
  if (opts?.dry_run) form.append('dry_run', 'true')
  if (opts?.service_protocol) form.append('service_protocol', opts.service_protocol)
  if (opts?.service_host) form.append('service_host', opts.service_host)
  if (opts?.service_port) form.append('service_port', String(opts.service_port))
  if (opts?.service_path) form.append('service_path', opts.service_path)
  return request<ImportOpenAPIResult>(() =>
    http.post(`/groups/${gid}/apis/import-openapi`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    }),
  )
}

export function updateApi(aid: number, data: Partial<ApiItem>) {
  return request<ApiItem>(() => http.put(`/apis/${aid}`, data))
}

export function deleteApi(aid: number) {
  return request<null>(() => http.delete(`/apis/${aid}`))
}

export function publishApi(aid: number) {
  return request<ApiItem>(() => http.post(`/apis/${aid}/publish`))
}

export function offlineApi(aid: number) {
  return request<ApiItem>(() => http.post(`/apis/${aid}/offline`))
}

export function shareApi(aid: number) {
  return request<ApiItem>(() => http.post(`/apis/${aid}/share`))
}

export function unshareApi(aid: number) {
  return request<ApiItem>(() => http.post(`/apis/${aid}/unshare`))
}

export function listMarketApis() {
  return request<ApiItem[]>(() => http.get('/market/apis'))
}

export function switchVersion(aid: number, version: string) {
  return request<ApiItem>(() => http.post(`/apis/${aid}/switch-version`, { version }))
}

export function listVersions(aid: number) {
  return request<ApiVersion[]>(() => http.get(`/apis/${aid}/versions`))
}

export function listUpstreams(spaceId: number) {
  return request<UpstreamItem[]>(() => http.get(`/spaces/${spaceId}/upstreams`))
}

export function createUpstream(spaceId: number, data: Partial<UpstreamItem>) {
  return request<UpstreamItem>(() => http.post(`/spaces/${spaceId}/upstreams`, data))
}

export function updateUpstream(spaceId: number, id: number, data: Partial<UpstreamItem>) {
  return request<UpstreamItem>(() => http.put(`/spaces/${spaceId}/upstreams/${id}`, data))
}

export function deleteUpstream(spaceId: number, id: number) {
  return request<null>(() => http.delete(`/spaces/${spaceId}/upstreams/${id}`))
}

export function listConsumers(spaceId: number) {
  return request<ConsumerItem[]>(() => http.get(`/spaces/${spaceId}/consumers`))
}

export function createConsumer(spaceId: number, data: Partial<ConsumerItem>) {
  return request<ConsumerItem>(() => http.post(`/spaces/${spaceId}/consumers`, data))
}

export function updateConsumer(spaceId: number, id: number, data: Partial<ConsumerItem>) {
  return request<ConsumerItem>(() => http.put(`/spaces/${spaceId}/consumers/${id}`, data))
}

export function deleteConsumer(spaceId: number, id: number) {
  return request<null>(() => http.delete(`/spaces/${spaceId}/consumers/${id}`))
}

export function syncConsumers(spaceId: number) {
  return request<null>(() => http.post(`/spaces/${spaceId}/consumers/sync`))
}

export function listPlugins(spaceId: number) {
  return request<PluginItem[]>(() => http.get(`/spaces/${spaceId}/plugins`))
}

export function listPluginApis(spaceId: number, pluginId: number) {
  return request<ApiItem[]>(() => http.get(`/spaces/${spaceId}/plugins/${pluginId}/apis`))
}

export function createPlugin(spaceId: number, data: Partial<PluginItem>) {
  return request<PluginItem>(() => http.post(`/spaces/${spaceId}/plugins`, data))
}

export function updatePlugin(spaceId: number, id: number, data: Partial<PluginItem>) {
  return request<PluginItem>(() => http.put(`/spaces/${spaceId}/plugins/${id}`, data))
}

export function deletePlugin(spaceId: number, id: number) {
  return request<null>(() => http.delete(`/spaces/${spaceId}/plugins/${id}`))
}
