import http, { request } from './http'
import type { ApiGroup, ApiItem, ApiVersion } from '@/types'

export function listGroups(spaceId: number) {
  return request<ApiGroup[]>(() => http.get(`/spaces/${spaceId}/groups`))
}

export function createGroup(spaceId: number, data: { name: string; gateway_id: number }) {
  return request<ApiGroup>(() => http.post(`/spaces/${spaceId}/groups`, data))
}

export function updateGroup(gid: number, data: { name?: string; gateway_id?: number }) {
  return request<ApiGroup>(() => http.put(`/groups/${gid}`, data))
}

export function deleteGroup(gid: number) {
  return request<null>(() => http.delete(`/groups/${gid}`))
}

export function listApis(gid: number) {
  return request<ApiItem[]>(() => http.get(`/groups/${gid}/apis`))
}

export function createApi(gid: number, data: Partial<ApiItem>) {
  return request<ApiItem>(() => http.post(`/groups/${gid}/apis`, data))
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

export function switchVersion(aid: number, version: string) {
  return request<ApiItem>(() => http.post(`/apis/${aid}/switch-version`, { version }))
}

export function listVersions(aid: number) {
  return request<ApiVersion[]>(() => http.get(`/apis/${aid}/versions`))
}
