import http, { request } from './http'
import type { Gateway } from '@/types'

export interface GatewayOption {
  id: number
  name: string
  network_zone: string
}

export function listGateways() {
  return request<Gateway[]>(() => http.get('/gateways'))
}

export function listGatewayOptions() {
  return request<GatewayOption[]>(() => http.get('/gateways-options'))
}

export function probeGateway(admin_api: string) {
  return request<{ ok: boolean }>(() => http.post('/gateways/probe', { admin_api }))
}

export function createGateway(data: { name: string; admin_api: string; domain: string; network_zone: string }) {
  return request<Gateway>(() => http.post('/gateways', data))
}

export function updateGateway(id: number, data: Partial<Gateway>) {
  return request<Gateway>(() => http.put(`/gateways/${id}`, data))
}

export function deleteGateway(id: number) {
  return request<null>(() => http.delete(`/gateways/${id}`))
}
