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

export function proxyDebugRequest(data: DebugProxyRequest) {
  return request<DebugProxyResponse>(() =>
    http.post('/debug/proxy', data, { timeout: Math.max(data.timeout_ms || 30000, 35000) }),
  )
}
