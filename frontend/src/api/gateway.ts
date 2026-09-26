import http, { request } from './http'
import type { Gateway, Space } from '@/types'

export interface GatewayOption {
  id: number
  name: string
  network_zone: string
}

export function listGateways() {
  return request<Gateway[]>(() => http.get('/gateways'))
}

export function listGatewayOptions(spaceId?: number) {
  if (spaceId) {
    return request<GatewayOption[]>(() => http.get(`/spaces/${spaceId}/gateway-options`))
  }
  return request<GatewayOption[]>(() => http.get('/gateways-options'))
}

export function listUnauthorizedSpaces(gatewayId: number) {
  return request<Space[]>(() => http.get(`/gateways/${gatewayId}/unauthorized-spaces`))
}

export function authorizeGatewaySpaces(gatewayId: number, spaceIds: number[]) {
  return request<null>(() => http.post(`/gateways/${gatewayId}/authorize-spaces`, { space_ids: spaceIds }))
}

export function probeGateway(admin_api: string) {
  return request<{ ok: boolean }>(() => http.post('/gateways/probe', { admin_api }))
}

export function createGateway(data: {
  name: string
  admin_api: string
  domain: string
  network_zone: string
  shared?: boolean
}) {
  return request<Gateway>(() => http.post('/gateways', data))
}

export function updateGateway(
  id: number,
  data: { name?: string; domain?: string; network_zone?: string; shared?: boolean },
) {
  return request<Gateway>(() => http.put(`/gateways/${id}`, data))
}

export function deleteGateway(id: number) {
  return request<null>(() => http.delete(`/gateways/${id}`))
}
