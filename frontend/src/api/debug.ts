import http, { request } from './http'

export interface DebugProxyRequest {
  method: string
  url: string
  headers?: Record<string, string>
  body?: string
  body_base64?: string
  timeout_ms?: number
}

export interface DebugProxyResponse {
  status: number
  status_text: string
  headers: Record<string, string[]>
  body: string
  duration_ms: number
  size: number
  truncated?: boolean
}

export interface DebugHistoryRequest {
  proxy_mode?: string
  params?: { enabled: boolean; key: string; value: string }[]
  headers?: { enabled: boolean; key: string; value: string }[]
  body_type?: string
  body_text?: string
  form_fields?: { enabled: boolean; key: string; value: string }[]
  form_data_fields?: {
    enabled: boolean
    key: string
    type: 'text' | 'file'
    value: string
    file_name?: string
  }[]
  auth?: Record<string, unknown>
}

export interface DebugHistoryItem {
  id: number
  space_id: number
  created_by: number
  api_id?: number
  api_name?: string
  title: string
  method: string
  url: string
  request: DebugHistoryRequest
  created_at: string
  creator?: { id: number; username: string }
}

export function proxyDebugRequest(data: DebugProxyRequest) {
  return request<DebugProxyResponse>(() =>
    http.post('/debug/proxy', data, { timeout: Math.max(data.timeout_ms || 30000, 35000) }),
  )
}

export function listDebugHistories(spaceId: number) {
  return request<DebugHistoryItem[]>(() => http.get(`/spaces/${spaceId}/debug-histories`))
}

export function createDebugHistory(
  spaceId: number,
  data: {
    title?: string
    api_id?: number
    api_name?: string
    method: string
    url: string
    request: DebugHistoryRequest
  },
) {
  return request<DebugHistoryItem>(() => http.post(`/spaces/${spaceId}/debug-histories`, data))
}

export function deleteDebugHistory(spaceId: number, id: number) {
  return request<null>(() => http.delete(`/spaces/${spaceId}/debug-histories/${id}`))
}
