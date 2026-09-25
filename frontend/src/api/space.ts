import http, { request } from './http'
import type { Space, SpaceMember, User } from '@/types'

export function listSpaces() {
  return request<Space[]>(() => http.get('/spaces'))
}

export function listAvailableSpaces() {
  return request<Space[]>(() => http.get('/spaces/available'))
}

export function createSpace(data: { name: string; description?: string; prefix: string }) {
  return request<Space>(() => http.post('/spaces', data))
}

export function updateSpace(id: number, data: { name?: string; description?: string }) {
  return request<Space>(() => http.put(`/spaces/${id}`, data))
}

export function deleteSpace(id: number) {
  return request<null>(() => http.delete(`/spaces/${id}`))
}

export function getSpace(id: number) {
  return request<Space>(() => http.get(`/spaces/${id}`))
}

export function joinSpace(id: number) {
  return request<null>(() => http.post(`/spaces/${id}/join`))
}

export function approveSpace(id: number) {
  return request<Space>(() => http.post(`/spaces/${id}/approve`))
}

export function rejectSpace(id: number) {
  return request<null>(() => http.post(`/spaces/${id}/reject`))
}

export function listMembers(spaceId: number) {
  return request<SpaceMember[]>(() => http.get(`/spaces/${spaceId}/members`))
}

export function listCandidateUsers(spaceId: number) {
  return request<User[]>(() => http.get(`/spaces/${spaceId}/members/candidates`))
}

export function addMember(spaceId: number, data: { user_id: number; role?: string }) {
  return request<null>(() => http.post(`/spaces/${spaceId}/members`, data))
}

export function approveMember(spaceId: number, uid: number) {
  return request<null>(() => http.post(`/spaces/${spaceId}/members/${uid}/approve`))
}

export function rejectMember(spaceId: number, uid: number) {
  return request<null>(() => http.post(`/spaces/${spaceId}/members/${uid}/reject`))
}

export function removeMember(spaceId: number, uid: number) {
  return request<null>(() => http.delete(`/spaces/${spaceId}/members/${uid}`))
}

export function updateMemberRole(spaceId: number, uid: number, role: string) {
  return request<null>(() => http.put(`/spaces/${spaceId}/members/${uid}/role`, { role }))
}
