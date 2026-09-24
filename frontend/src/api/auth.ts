import http, { request } from './http'
import type { User } from '@/types'

export function login(username: string, password: string) {
  return request<{ token: string; user: User }>(() =>
    http.post('/auth/login', { username, password }),
  )
}

export function register(username: string, password: string) {
  return request<{ token: string; user: User }>(() =>
    http.post('/auth/register', { username, password }),
  )
}

export function fetchMe() {
  return request<User>(() => http.get('/auth/me'))
}
